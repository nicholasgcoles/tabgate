package tui

import (
	"sort"

	"github.com/nic/tabgate/internal/adapter"
)

// Project represents a group of tabs sharing the same git repo.
type Project struct {
	Name      string
	Directory string // repo root path, or empty for "Other"
	Tabs      []adapter.Tab
}

// GroupByProject groups tabs by their RepoRoot. Projects are sorted
// alphabetically by name, with "Other" (tabs not in a git repo) at the end.
func GroupByProject(tabs []adapter.Tab) []Project {
	groups := make(map[string]*Project)
	var order []string

	for _, tab := range tabs {
		key := tab.RepoRoot
		name := tab.RepoName
		dir := tab.RepoRoot
		if key == "" {
			key = "_other"
			name = "Other"
			dir = ""
		}
		if _, ok := groups[key]; !ok {
			groups[key] = &Project{Name: name, Directory: dir}
			order = append(order, key)
		}
		p := groups[key]
		p.Tabs = append(p.Tabs, tab)
	}

	// Sort keys: alphabetical, but "_other" always last.
	sort.Slice(order, func(i, j int) bool {
		if order[i] == "_other" {
			return false
		}
		if order[j] == "_other" {
			return true
		}
		return groups[order[i]].Name < groups[order[j]].Name
	})

	var projects []Project
	for _, key := range order {
		projects = append(projects, *groups[key])
	}
	return projects
}

// FlatIndex maps a flat cursor position to a (project index, tab index) pair.
// Returns -1, -1 if pos is out of range.
func FlatIndex(projects []Project, pos int) (int, int) {
	i := 0
	for pi, p := range projects {
		for ti := range p.Tabs {
			if i == pos {
				return pi, ti
			}
			i++
		}
	}
	return -1, -1
}

// TotalTabs returns the total number of tabs across all projects.
func TotalTabs(projects []Project) int {
	n := 0
	for _, p := range projects {
		n += len(p.Tabs)
	}
	return n
}

// FlatPos returns the flat cursor position for a given tab ID,
// or -1 if not found.
func FlatPos(projects []Project, tabID string) int {
	i := 0
	for _, p := range projects {
		for _, tab := range p.Tabs {
			if tab.ID == tabID {
				return i
			}
			i++
		}
	}
	return -1
}
