package adapter

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nic/tabgate/internal/applescript"
)

// rawTab holds the fields parsed directly from the AppleScript output before
// enrichment (CWD lookup, git info, etc.).
type rawTab struct {
	WindowID string
	TabIndex string
	TTY      string
}

// TerminalAppAdapter implements TerminalAdapter for macOS Terminal.app.
type TerminalAppAdapter struct {
	ownTTY string
}

func (a *TerminalAppAdapter) Name() string { return "Terminal.app" }

// NewTerminalAppAdapter creates an adapter that talks to Terminal.app.
// It detects the current process's TTY so it can exclude its own tab from
// listings. If TTY detection fails the adapter still works — it just won't
// filter anything.
func NewTerminalAppAdapter() *TerminalAppAdapter {
	return &TerminalAppAdapter{ownTTY: ttyname()}
}

// listTabsScript is the AppleScript that enumerates Terminal.app windows and
// tabs, outputting one line per tab: windowID|tabIndex|tty
const listTabsScript = `
tell application "Terminal"
	set output to ""
	repeat with w in windows
		set wID to id of w
		set tabList to tabs of w
		repeat with i from 1 to count of tabList
			set t to item i of tabList
			set ttyName to tty of t
			set output to output & wID & "|" & i & "|" & ttyName & linefeed
		end repeat
	end repeat
	return output
end tell
`

// ListTabs enumerates all open Terminal.app tabs, resolves each tab's working
// directory, and returns Tab structs. The tab running TabGate itself is
// included with IsSelf set to true.
func (a *TerminalAppAdapter) ListTabs() ([]Tab, error) {
	out, err := applescript.Run(listTabsScript)
	if err != nil {
		if strings.Contains(err.Error(), "not allowed") || strings.Contains(err.Error(), "1002") {
			return nil, fmt.Errorf("terminal_app: permission denied. Grant access in System Settings > Privacy & Security > Automation")
		}
		return nil, fmt.Errorf("terminal_app: list tabs: %w", err)
	}

	raws := parseListTabsOutput(out)
	var tabs []Tab
	for _, r := range raws {
		dir := cwdForTTY(r.TTY)

		tabs = append(tabs, Tab{
			ID:           r.TTY,
			WindowID:     r.WindowID,
			Directory:    dir,
			IsSelf:       a.ownTTY != "" && r.TTY == a.ownTTY,
			TerminalType: "terminal.app",
		})
	}
	return tabs, nil
}

// parseListTabsOutput splits the raw AppleScript output into rawTab records.
// Lines that don't have exactly 3 pipe-delimited fields are silently skipped.
func parseListTabsOutput(output string) []rawTab {
	var tabs []rawTab
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
			continue
		}
		tabs = append(tabs, rawTab{
			WindowID: parts[0],
			TabIndex: parts[1],
			TTY:      parts[2],
		})
	}
	return tabs
}

// cwdForTTY resolves the current working directory for the shell process
// attached to the given TTY path (e.g. /dev/ttys003).
func cwdForTTY(tty string) string {
	pid := shellPIDForTTY(tty)
	if pid == "" {
		return ""
	}

	out, err := exec.Command("lsof", "-a", "-p", pid, "-d", "cwd", "-Fn").Output()
	if err != nil {
		return ""
	}
	return parseLsofOutput(string(out))
}

// shellPIDForTTY finds the PID of the shell process attached to the given TTY.
// It uses `ps` and picks the first process whose command looks like a shell.
func shellPIDForTTY(tty string) string {
	// Strip /dev/ prefix — ps expects just the device name.
	dev := strings.TrimPrefix(tty, "/dev/")

	out, err := exec.Command("ps", "-t", dev, "-o", "pid=,comm=").Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid := fields[0]
		comm := fields[1]

		// Validate PID is numeric.
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}

		// Match common shell names.
		base := comm
		if idx := strings.LastIndex(comm, "/"); idx >= 0 {
			base = comm[idx+1:]
		}
		base = strings.TrimPrefix(base, "-") // login shells like -zsh
		switch base {
		case "zsh", "bash", "fish", "sh", "dash", "ksh", "tcsh", "csh":
			return pid
		}
	}
	return ""
}

// parseLsofOutput extracts the directory path from lsof -Fn output. The output
// typically looks like:
//
//	p12345
//	fcwd
//	n/Users/someone/project
//
// We want the value after the last line starting with 'n'.
func parseLsofOutput(output string) string {
	var dir string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "n") {
			dir = line[1:]
		}
	}
	return dir
}

// findTabScript generates AppleScript that iterates Terminal.app windows/tabs
// to find the one matching the given TTY. It returns a snippet that sets
// variables `targetWindow` and `targetTabIndex` if found.
func findTabScript(tty string) string {
	return fmt.Sprintf(`
tell application "Terminal"
	set targetWindow to missing value
	set targetTabIndex to -1
	repeat with w in windows
		set tabList to tabs of w
		repeat with i from 1 to count of tabList
			set t to item i of tabList
			if tty of t is "%s" then
				set targetWindow to w
				set targetTabIndex to i
				exit repeat
			end if
		end repeat
		if targetTabIndex is not -1 then exit repeat
	end repeat
`, tty)
}

// SwitchTo activates the Terminal.app tab matching the given TTY path.
func (a *TerminalAppAdapter) SwitchTo(tabID string) error {
	script := findTabScript(tabID) + `
	if targetWindow is not missing value then
		set frontmost of targetWindow to true
		set selected tab of targetWindow to tab targetTabIndex of targetWindow
		activate
	end if
end tell
`
	_, err := applescript.Run(script)
	if err != nil {
		return fmt.Errorf("terminal_app: switch to %s: %w", tabID, err)
	}
	return nil
}

// Close closes the Terminal.app tab matching the given TTY path.
func (a *TerminalAppAdapter) Close(tabID string) error {
	script := findTabScript(tabID) + `
	if targetWindow is not missing value then
		close tab targetTabIndex of targetWindow
	end if
end tell
`
	_, err := applescript.Run(script)
	if err != nil {
		return fmt.Errorf("terminal_app: close %s: %w", tabID, err)
	}
	return nil
}

// Create opens a new Terminal.app tab and changes to the given directory.
func (a *TerminalAppAdapter) Create(directory string) error {
	script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script "cd %s"
end tell`, directory)
	_, err := applescript.Run(script)
	if err != nil {
		return fmt.Errorf("terminal_app: create tab in %s: %w", directory, err)
	}
	return nil
}

// Rename sets the custom title of the Terminal.app tab matching the given TTY.
func (a *TerminalAppAdapter) Rename(tabID string, name string) error {
	script := findTabScript(tabID) + fmt.Sprintf(`
	if targetWindow is not missing value then
		set custom title of tab targetTabIndex of targetWindow to "%s"
	end if
end tell
`, name)
	_, err := applescript.Run(script)
	if err != nil {
		return fmt.Errorf("terminal_app: rename %s: %w", tabID, err)
	}
	return nil
}
