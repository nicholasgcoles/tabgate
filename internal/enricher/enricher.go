package enricher

import (
	"github.com/nic/tabgate/internal/adapter"
)

// TabEnricher enriches raw tab data with git and process information.
type TabEnricher struct {
	git *GitResolver
}

// NewTabEnricher creates a new TabEnricher.
func NewTabEnricher() *TabEnricher {
	return &TabEnricher{
		git: NewGitResolver(),
	}
}

// Enrich fills in git and process metadata for each tab.
// If any enrichment step fails, the corresponding fields are left empty.
func (e *TabEnricher) Enrich(tabs []adapter.Tab) []adapter.Tab {
	for i := range tabs {
		if tabs[i].Directory != "" {
			repoRoot, repoName, branch, isWorktree, err := e.git.Resolve(tabs[i].Directory)
			if err == nil {
				tabs[i].RepoRoot = repoRoot
				tabs[i].RepoName = repoName
				tabs[i].Branch = branch
				tabs[i].IsWorktree = isWorktree
			}
		}

		if tabs[i].ID != "" {
			cmd, err := ResolveForTTY(tabs[i].ID)
			if err == nil {
				tabs[i].RunningCommand = cmd
			}
		}
	}
	return tabs
}
