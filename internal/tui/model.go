package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nic/tabgate/internal/adapter"
	"github.com/nic/tabgate/internal/poller"
)

// actionDoneMsg is sent when an async adapter action completes.
type actionDoneMsg struct {
	statusMsg string
	err       error
	repoll    bool
}

// Model is the Bubble Tea model for TabGate.
type Model struct {
	projects      []Project
	tabs          []adapter.Tab
	adapters      []adapter.TerminalAdapter
	cursor        int
	selectedTabID string
	width         int
	height        int
	quitting      bool
	poller        *poller.Poller
	confirmClose  bool
	renaming      bool
	renameInput   textinput.Model
	statusMsg     string
}

// NewModel creates a new TUI model with the given tabs and optional poller.
func NewModel(tabs []adapter.Tab, p *poller.Poller, adapters ...adapter.TerminalAdapter) Model {
	projects := GroupByProject(tabs)
	ti := textinput.New()
	ti.Placeholder = "new name"
	ti.CharLimit = 64
	return Model{
		projects:    projects,
		tabs:        tabs,
		adapters:    adapters,
		width:       80,
		height:      24,
		poller:      p,
		renameInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	if m.poller != nil {
		return m.poller.Poll()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			m.statusMsg = "Error: " + msg.err.Error()
		} else if msg.statusMsg != "" {
			m.statusMsg = msg.statusMsg
		}
		// If the action requires a repoll, trigger one.
		if msg.repoll && m.poller != nil {
			return m, m.poller.Poll()
		}
		return m, nil

	case poller.TabsUpdatedMsg:
		m.tabs = msg.Tabs
		m.projects = GroupByProject(m.tabs)

		// Preserve cursor position by finding the previously selected tab.
		if m.selectedTabID != "" {
			pos := FlatPos(m.projects, m.selectedTabID)
			if pos >= 0 {
				m.cursor = pos
			} else {
				// Selected tab is gone — clamp cursor to nearest valid position.
				total := TotalTabs(m.projects)
				if m.cursor >= total && total > 0 {
					m.cursor = total - 1
				}
				if total == 0 {
					m.cursor = 0
					m.selectedTabID = ""
				} else {
					m.updateSelectedID()
				}
			}
		}

		if m.poller != nil {
			return m, m.poller.Poll()
		}
		return m, nil

	case tea.KeyMsg:
		// Handle confirmation mode.
		if m.confirmClose {
			switch msg.String() {
			case "y":
				m.confirmClose = false
				m.statusMsg = ""
				pi, ti := FlatIndex(m.projects, m.cursor)
				if pi >= 0 && ti >= 0 {
					tab := m.projects[pi].Tabs[ti]
					return m, m.closeTab(tab.ID)
				}
				return m, nil
			case "n", "esc":
				m.confirmClose = false
				m.statusMsg = ""
				return m, nil
			}
			return m, nil
		}

		// Handle rename mode.
		if m.renaming {
			switch msg.Type {
			case tea.KeyEnter:
				m.renaming = false
				name := m.renameInput.Value()
				m.renameInput.SetValue("")
				m.renameInput.Blur()
				if name == "" {
					return m, nil
				}
				pi, ti := FlatIndex(m.projects, m.cursor)
				if pi >= 0 && ti >= 0 {
					tab := m.projects[pi].Tabs[ti]
					return m, m.renameTab(tab.ID, name)
				}
				return m, nil
			case tea.KeyEsc:
				m.renaming = false
				m.renameInput.SetValue("")
				m.renameInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.renameInput, cmd = m.renameInput.Update(msg)
				return m, cmd
			}
		}

		total := TotalTabs(m.projects)
		if total == 0 {
			if key.Matches(msg, keys.Quit) {
				m.quitting = true
				return m, tea.Quit
			}
			if key.Matches(msg, keys.New) {
				return m, m.createTab("")
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			m.statusMsg = ""
			m.updateSelectedID()
		case key.Matches(msg, keys.Down):
			if m.cursor < total-1 {
				m.cursor++
			}
			m.statusMsg = ""
			m.updateSelectedID()
		case key.Matches(msg, keys.Enter):
			pi, ti := FlatIndex(m.projects, m.cursor)
			if pi >= 0 && ti >= 0 {
				tab := m.projects[pi].Tabs[ti]
				m.quitting = true
				return m, tea.Batch(m.switchToTab(tab.ID), tea.Quit)
			}
		case key.Matches(msg, keys.New):
			pi, _ := FlatIndex(m.projects, m.cursor)
			dir := ""
			if pi >= 0 {
				dir = m.projects[pi].Directory
			}
			return m, m.createTab(dir)
		case key.Matches(msg, keys.Delete):
			m.confirmClose = true
			m.statusMsg = "Close this tab? (y/n)"
			return m, nil
		case key.Matches(msg, keys.Rename):
			m.renaming = true
			m.renameInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	return renderView(m)
}

func (m *Model) updateSelectedID() {
	pi, ti := FlatIndex(m.projects, m.cursor)
	if pi >= 0 && ti >= 0 {
		m.selectedTabID = m.projects[pi].Tabs[ti].ID
	}
}

// adapter action helpers — each returns a tea.Cmd that runs the action async.

func (m Model) firstAdapter() adapter.TerminalAdapter {
	if len(m.adapters) == 0 {
		return nil
	}
	return m.adapters[0]
}

func (m Model) switchToTab(tabID string) tea.Cmd {
	a := m.firstAdapter()
	if a == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.SwitchTo(tabID)
		return actionDoneMsg{err: err}
	}
}

func (m Model) closeTab(tabID string) tea.Cmd {
	a := m.firstAdapter()
	if a == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.Close(tabID)
		return actionDoneMsg{statusMsg: "Tab closed", err: err, repoll: true}
	}
}

func (m Model) createTab(directory string) tea.Cmd {
	a := m.firstAdapter()
	if a == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.Create(directory)
		return actionDoneMsg{statusMsg: "New tab created", err: err, repoll: true}
	}
}

func (m Model) renameTab(tabID string, name string) tea.Cmd {
	a := m.firstAdapter()
	if a == nil {
		return nil
	}
	return func() tea.Msg {
		err := a.Rename(tabID, name)
		return actionDoneMsg{statusMsg: "Tab renamed", err: err}
	}
}
