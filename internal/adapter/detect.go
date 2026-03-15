package adapter

import (
	"fmt"

	"github.com/nic/tabgate/internal/applescript"
)

// isAppRunning checks if a macOS application is running via System Events.
// This is more reliable than pgrep, which cannot see system-bundled apps
// like Terminal.app under /System/Applications/.
func isAppRunning(appName string) bool {
	script := fmt.Sprintf(
		`tell application "System Events" to return (name of every process whose name is "%s") contains "%s"`,
		appName, appName,
	)
	out, err := applescript.Run(script)
	return err == nil && out == "true"
}

// DetectAdapters returns adapters for all currently running terminal emulators.
func DetectAdapters() []TerminalAdapter {
	var adapters []TerminalAdapter

	if isAppRunning("Terminal") {
		adapters = append(adapters, NewTerminalAppAdapter())
	}

	if isAppRunning("ghostty") {
		adapters = append(adapters, NewGhosttyAdapter())
	}

	return adapters
}
