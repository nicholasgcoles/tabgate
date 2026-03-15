package poller

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nic/tabgate/internal/adapter"
	"github.com/nic/tabgate/internal/enricher"
)

// TabsUpdatedMsg is sent to the TUI when new tab data is available.
type TabsUpdatedMsg struct {
	Tabs []adapter.Tab
}

// Poller handles periodic tab discovery and enrichment.
type Poller struct {
	adapters []adapter.TerminalAdapter
	enricher *enricher.TabEnricher
}

// NewPoller creates a new Poller with the given adapters and enricher.
func NewPoller(adapters []adapter.TerminalAdapter, enricher *enricher.TabEnricher) *Poller {
	return &Poller{
		adapters: adapters,
		enricher: enricher,
	}
}

// Poll returns a tea.Cmd that sleeps 2 seconds, then runs all adapters + enricher,
// returning a TabsUpdatedMsg.
func (p *Poller) Poll() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		var allTabs []adapter.Tab
		for _, a := range p.adapters {
			tabs, err := a.ListTabs()
			if err != nil {
				continue
			}
			allTabs = append(allTabs, tabs...)
		}
		allTabs = p.enricher.Enrich(allTabs)
		return TabsUpdatedMsg{Tabs: allTabs}
	}
}
