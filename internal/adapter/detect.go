package adapter

import "os/exec"

// DetectAdapters returns adapters for all currently running terminal emulators.
// Uses `pgrep` to check if each supported terminal is running.
func DetectAdapters() []TerminalAdapter {
	var adapters []TerminalAdapter

	// Check if Terminal.app is running.
	// Use -f to match against the full process path — pgrep -x "Terminal"
	// doesn't work on macOS because the process name doesn't match exactly.
	if err := exec.Command("pgrep", "-f", "Terminal.app").Run(); err == nil {
		adapters = append(adapters, NewTerminalAppAdapter())
	}

	// Check if Ghostty is running.
	if err := exec.Command("pgrep", "-x", "ghostty").Run(); err == nil {
		adapters = append(adapters, NewGhosttyAdapter())
	}

	return adapters
}
