package poller

import (
	"fmt"
	"testing"

	"github.com/nic/tabgate/internal/adapter"
	"github.com/nic/tabgate/internal/enricher"
)

// mockAdapter is a test adapter that returns fixed tabs.
type mockAdapter struct {
	tabs []adapter.Tab
	err  error
}

func (m *mockAdapter) ListTabs() ([]adapter.Tab, error) {
	return m.tabs, m.err
}

func (m *mockAdapter) SwitchTo(tabID string) error       { return nil }
func (m *mockAdapter) Close(tabID string) error           { return nil }
func (m *mockAdapter) Create(directory string) error       { return nil }
func (m *mockAdapter) Rename(tabID, name string) error     { return nil }

func TestPollReturnsTabsUpdatedMsg(t *testing.T) {
	fixedTabs := []adapter.Tab{
		{ID: "tab1", Directory: "/tmp/a", TerminalType: "mock"},
		{ID: "tab2", Directory: "/tmp/b", TerminalType: "mock"},
	}

	mock := &mockAdapter{tabs: fixedTabs}
	e := enricher.NewTabEnricher()
	p := NewPoller([]adapter.TerminalAdapter{mock}, e)

	cmd := p.Poll()
	msg := cmd()

	updMsg, ok := msg.(TabsUpdatedMsg)
	if !ok {
		t.Fatalf("expected TabsUpdatedMsg, got %T", msg)
	}

	if len(updMsg.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(updMsg.Tabs))
	}

	if updMsg.Tabs[0].ID != "tab1" {
		t.Errorf("expected first tab ID 'tab1', got %q", updMsg.Tabs[0].ID)
	}
	if updMsg.Tabs[1].ID != "tab2" {
		t.Errorf("expected second tab ID 'tab2', got %q", updMsg.Tabs[1].ID)
	}
}

func TestPollSkipsFailingAdapter(t *testing.T) {
	goodTabs := []adapter.Tab{
		{ID: "good1", Directory: "/tmp/good", TerminalType: "mock"},
	}

	good := &mockAdapter{tabs: goodTabs}
	bad := &mockAdapter{err: errMock}
	e := enricher.NewTabEnricher()
	p := NewPoller([]adapter.TerminalAdapter{bad, good}, e)

	cmd := p.Poll()
	msg := cmd()

	updMsg, ok := msg.(TabsUpdatedMsg)
	if !ok {
		t.Fatalf("expected TabsUpdatedMsg, got %T", msg)
	}

	if len(updMsg.Tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(updMsg.Tabs))
	}

	if updMsg.Tabs[0].ID != "good1" {
		t.Errorf("expected tab ID 'good1', got %q", updMsg.Tabs[0].ID)
	}
}

var errMock = fmt.Errorf("mock error")
