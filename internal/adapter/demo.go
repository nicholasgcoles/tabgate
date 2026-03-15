package adapter

import (
	"fmt"
	"sync"
)

// DemoAdapter provides fake tab data for demonstration and testing.
type DemoAdapter struct {
	mu   sync.Mutex
	tabs []Tab
	next int
}

// NewDemoAdapter creates a DemoAdapter pre-populated with realistic fake tabs.
func NewDemoAdapter() *DemoAdapter {
	tabs := []Tab{
		{
			ID:             "demo-1",
			WindowID:       "win-1",
			Directory:      "~/Projects/tabgate",
			RepoRoot:       "~/Projects/tabgate",
			RepoName:       "tabgate",
			Branch:         "main",
			RunningCommand: "tabgate --demo",
			TerminalType:   "demo",
			IsSelf:         true,
		},
		{
			ID:           "demo-2",
			WindowID:     "win-1",
			Directory:    "~/Projects/tabgate",
			RepoRoot:     "~/Projects/tabgate",
			RepoName:     "tabgate",
			Branch:       "feat/demo-mode",
			IsWorktree:   true,
			TerminalType: "demo",
		},
		{
			ID:             "demo-3",
			WindowID:       "win-2",
			Directory:      "~/Projects/acme-api",
			RepoRoot:       "~/Projects/acme-api",
			RepoName:       "acme-api",
			Branch:         "main",
			RunningCommand: "docker compose up",
			TerminalType:   "demo",
		},
		{
			ID:             "demo-4",
			WindowID:       "win-2",
			Directory:      "~/Projects/acme-api",
			RepoRoot:       "~/Projects/acme-api",
			RepoName:       "acme-api",
			Branch:         "dev",
			RunningCommand: "vim",
			TerminalType:   "demo",
		},
		{
			ID:           "demo-5",
			WindowID:     "win-2",
			Directory:    "~/Projects/acme-api",
			RepoRoot:     "~/Projects/acme-api",
			RepoName:     "acme-api",
			Branch:       "staging",
			TerminalType: "demo",
		},
		{
			ID:           "demo-6",
			WindowID:     "win-3",
			Directory:    "~/dotfiles",
			RepoRoot:     "~/dotfiles",
			RepoName:     "dotfiles",
			Branch:       "main",
			TerminalType: "demo",
		},
		{
			ID:           "demo-7",
			WindowID:     "win-4",
			Directory:    "~/Downloads",
			TerminalType: "demo",
		},
		{
			ID:             "demo-8",
			WindowID:       "win-4",
			Directory:      "~/Documents",
			RunningCommand: "python3",
			TerminalType:   "demo",
		},
	}
	return &DemoAdapter{tabs: tabs, next: 9}
}

func (d *DemoAdapter) Name() string { return "demo" }

func (d *DemoAdapter) ListTabs() ([]Tab, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]Tab, len(d.tabs))
	copy(out, d.tabs)
	return out, nil
}

func (d *DemoAdapter) SwitchTo(tabID string) error { return nil }

func (d *DemoAdapter) Close(tabID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, t := range d.tabs {
		if t.ID == tabID {
			d.tabs = append(d.tabs[:i], d.tabs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("tab %s not found", tabID)
}

func (d *DemoAdapter) Create(directory string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tabs = append(d.tabs, Tab{
		ID:           fmt.Sprintf("demo-%d", d.next),
		WindowID:     "win-new",
		Directory:    directory,
		TerminalType: "demo",
	})
	d.next++
	return nil
}

func (d *DemoAdapter) Rename(tabID string, name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, t := range d.tabs {
		if t.ID == tabID {
			d.tabs[i].RunningCommand = name
			return nil
		}
	}
	return fmt.Errorf("tab %s not found", tabID)
}
