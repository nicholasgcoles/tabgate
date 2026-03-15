package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	projectNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("12"))

	projectDirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	worktreeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Padding(0, 1)

	confirmStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9")).
			Padding(0, 1)
)

func renderView(m Model) string {
	var b strings.Builder

	totalTabs := TotalTabs(m.projects)

	// Header.
	title := "TabGate"
	stats := fmt.Sprintf("%d projects · %d sessions", len(m.projects), totalTabs)
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(stats) - 2
	if gap < 1 {
		gap = 1
	}
	header := headerStyle.Width(m.width).Render(title + strings.Repeat(" ", gap) + stats)
	b.WriteString(header)
	b.WriteString("\n")

	if totalTabs == 0 {
		b.WriteString("\n  No terminal sessions found.\n\n")
		b.WriteString("  Make sure Terminal.app is running with at least one tab open.\n")
		b.WriteString("  TabGate will auto-refresh when tabs are detected.\n")
	} else {
		flatIdx := 0
		for _, p := range m.projects {
			b.WriteString("\n")
			// Project header.
			name := projectNameStyle.Render(p.Name)
			if p.Directory != "" {
				dir := projectDirStyle.Render("  " + shortPath(p.Directory))
				b.WriteString("  " + name + dir + "\n")
			} else {
				b.WriteString("  " + name + "\n")
			}

			for _, tab := range p.Tabs {
				selected := flatIdx == m.cursor
				flatIdx++

				cursor := "  "
				style := normalStyle
				if selected {
					cursor = "❯ "
					style = selectedStyle
				}

				// Branch or directory for "Other" tabs.
				label := tab.Branch
				if label == "" && tab.RepoRoot == "" {
					label = shortPath(tab.Directory)
				}
				if label == "" {
					label = "(unknown)"
				}

				// Worktree badge.
				wt := ""
				if tab.IsWorktree {
					wt = worktreeStyle.Render("  [worktree]")
				}

				// Running command.
				cmd := ""
				if tab.RunningCommand != "" {
					cmd = commandStyle.Render(tab.RunningCommand)
				}

				left := fmt.Sprintf("  %s%s%s", cursor, style.Render(label), wt)
				right := cmd

				pad := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
				if pad < 1 {
					pad = 1
				}
				b.WriteString(left + strings.Repeat(" ", pad) + right + "\n")
			}
		}
	}

	// Status / confirmation / rename input.
	if m.confirmClose {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render("Close this tab? (y/n)"))
		b.WriteString("\n")
	} else if m.renaming {
		b.WriteString("\n")
		b.WriteString("  Rename: " + m.renameInput.View())
		b.WriteString("\n")
	} else if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render(m.statusMsg))
		b.WriteString("\n")
	}

	// Footer.
	b.WriteString("\n")
	footer := footerStyle.Width(m.width).Render("↑↓/jk navigate  enter switch  n new  r rename  d close  q quit")
	b.WriteString(footer)

	return b.String()
}

// shortPath replaces the home directory prefix with ~.
func shortPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}
