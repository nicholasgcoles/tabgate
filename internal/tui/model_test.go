package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nic/tabgate/internal/adapter"
)

func testTabs() []adapter.Tab {
	return []adapter.Tab{
		{ID: "/dev/ttys001", RepoRoot: "/Users/nic/project-a", RepoName: "project-a", Branch: "main", RunningCommand: "nvim"},
		{ID: "/dev/ttys002", RepoRoot: "/Users/nic/project-a", RepoName: "project-a", Branch: "feature", IsWorktree: true, RunningCommand: "go test"},
		{ID: "/dev/ttys003", RepoRoot: "/Users/nic/project-b", RepoName: "project-b", Branch: "main", RunningCommand: "zsh (idle)"},
		{ID: "/dev/ttys004", Directory: "/Users/nic/Downloads", RunningCommand: "zsh (idle)"},
	}
}

func TestGroupByProject(t *testing.T) {
	projects := GroupByProject(testTabs())

	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}

	// Alphabetical order, "Other" last.
	if projects[0].Name != "project-a" {
		t.Errorf("expected first project 'project-a', got %q", projects[0].Name)
	}
	if projects[1].Name != "project-b" {
		t.Errorf("expected second project 'project-b', got %q", projects[1].Name)
	}
	if projects[2].Name != "Other" {
		t.Errorf("expected last project 'Other', got %q", projects[2].Name)
	}

	if len(projects[0].Tabs) != 2 {
		t.Errorf("expected project-a to have 2 tabs, got %d", len(projects[0].Tabs))
	}
	if len(projects[2].Tabs) != 1 {
		t.Errorf("expected Other to have 1 tab, got %d", len(projects[2].Tabs))
	}
}

func TestGroupByProjectEmpty(t *testing.T) {
	projects := GroupByProject(nil)
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCursorNavigation(t *testing.T) {
	m := NewModel(testTabs(), nil, nil)
	total := TotalTabs(m.projects)

	if m.cursor != 0 {
		t.Fatalf("expected initial cursor 0, got %d", m.cursor)
	}

	// Move down to last item.
	for i := 0; i < total; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
	}
	if m.cursor != total-1 {
		t.Errorf("expected cursor clamped at %d, got %d", total-1, m.cursor)
	}

	// Move up past beginning.
	for i := 0; i < total+5; i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = updated.(Model)
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", m.cursor)
	}
}

func TestQuit(t *testing.T) {
	m := NewModel(testTabs(), nil, nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestFlatIndex(t *testing.T) {
	projects := GroupByProject(testTabs())

	pi, ti := FlatIndex(projects, 0)
	if pi != 0 || ti != 0 {
		t.Errorf("pos 0: expected (0,0), got (%d,%d)", pi, ti)
	}

	pi, ti = FlatIndex(projects, 2)
	if pi != 1 || ti != 0 {
		t.Errorf("pos 2: expected (1,0), got (%d,%d)", pi, ti)
	}

	pi, ti = FlatIndex(projects, 99)
	if pi != -1 || ti != -1 {
		t.Errorf("pos 99: expected (-1,-1), got (%d,%d)", pi, ti)
	}
}

func TestFlatPos(t *testing.T) {
	projects := GroupByProject(testTabs())

	pos := FlatPos(projects, "/dev/ttys003")
	if pos != 2 {
		t.Errorf("expected pos 2 for ttys003, got %d", pos)
	}

	pos = FlatPos(projects, "nonexistent")
	if pos != -1 {
		t.Errorf("expected -1 for nonexistent, got %d", pos)
	}
}

func TestViewRenders(t *testing.T) {
	m := NewModel(testTabs(), nil, nil)
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view")
	}
}

func TestConfirmCloseToggle(t *testing.T) {
	m := NewModel(testTabs(), nil, nil)

	// Press 'd' to enter confirmation mode.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)

	if !m.confirmClose {
		t.Fatal("expected confirmClose to be true after pressing 'd'")
	}
	if m.statusMsg != "Close this tab? (y/n)" {
		t.Errorf("expected confirmation status message, got %q", m.statusMsg)
	}

	// Press 'n' to cancel.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)

	if m.confirmClose {
		t.Error("expected confirmClose to be false after pressing 'n'")
	}
	if m.statusMsg != "" {
		t.Errorf("expected empty status message after cancel, got %q", m.statusMsg)
	}
}

func TestRenameMode(t *testing.T) {
	m := NewModel(testTabs(), nil, nil)

	// Press 'r' to enter rename mode.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(Model)

	if !m.renaming {
		t.Fatal("expected renaming to be true after pressing 'r'")
	}

	// Press escape to cancel.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.renaming {
		t.Error("expected renaming to be false after pressing escape")
	}
}
