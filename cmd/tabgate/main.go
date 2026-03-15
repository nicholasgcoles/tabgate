package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nic/tabgate/internal/adapter"
	"github.com/nic/tabgate/internal/enricher"
	"github.com/nic/tabgate/internal/poller"
	"github.com/nic/tabgate/internal/tui"
)

func main() {
	adapters := adapter.DetectAdapters()
	if len(adapters) == 0 {
		fmt.Fprintln(os.Stderr, "No supported terminal emulators detected.")
		os.Exit(1)
	}

	e := enricher.NewTabEnricher()

	// Collect initial tabs for the first render.
	var tabs []adapter.Tab
	var initErrors []error
	for _, a := range adapters {
		t, err := a.ListTabs()
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("%s: %w", a.Name(), err))
			continue
		}
		tabs = append(tabs, t...)
	}
	tabs = e.Enrich(tabs)

	pl := poller.NewPoller(adapters, e)
	m := tui.NewModel(tabs, pl, initErrors, adapters...)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
