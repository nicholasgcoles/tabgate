package adapter

// Tab represents a terminal session with its metadata.
type Tab struct {
	ID             string // opaque, adapter-specific (Terminal.app uses TTY, Ghostty uses UUIDs)
	WindowID       string
	Directory      string
	RepoRoot       string // git repo root, empty if not in a repo
	RepoName       string // derived from repo root directory name
	Branch         string // current git branch
	IsWorktree     bool
	IsSelf         bool   // true when this tab is running TabGate
	RunningCommand string // foreground process name
	TerminalType   string // "terminal.app" or "ghostty"
}

// TerminalAdapter is the interface each terminal emulator implements.
type TerminalAdapter interface {
	// Name returns the human-readable name of this adapter (e.g. "Terminal.app").
	Name() string
	// ListTabs returns raw tab data: ID, WindowID, Directory, TerminalType.
	// Git and process enrichment is handled separately by the TabEnricher.
	ListTabs() ([]Tab, error)
	SwitchTo(tabID string) error
	Close(tabID string) error
	Create(directory string) error
	Rename(tabID string, name string) error
}
