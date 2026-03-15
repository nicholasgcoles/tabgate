package adapter

import (
	"fmt"
	"strings"

	"github.com/nic/tabgate/internal/applescript"
)

// ghosttyRawTab holds fields parsed from the AppleScript output for a Ghostty tab.
type ghosttyRawTab struct {
	WindowID   string
	TabID      string
	TerminalID string
	WorkingDir string
	Name       string
}

// GhosttyAdapter implements TerminalAdapter for the Ghostty terminal emulator.
type GhosttyAdapter struct{}

func (a *GhosttyAdapter) Name() string { return "Ghostty" }

// NewGhosttyAdapter creates an adapter that talks to Ghostty via AppleScript.
func NewGhosttyAdapter() *GhosttyAdapter {
	return &GhosttyAdapter{}
}

// ghosttyListScript enumerates Ghostty windows → tabs → focused terminal,
// outputting one line per tab: windowID|tabID|terminalID|workingDir|name
const ghosttyListScript = `
tell application "Ghostty"
	set output to ""
	repeat with w in windows
		set wID to id of w
		set tabList to tabs of w
		repeat with t in tabList
			set tID to id of t
			set tName to name of t
			set term to focused terminal of t
			set termID to id of term
			set termDir to working directory of term
			set output to output & wID & "|" & tID & "|" & termID & "|" & termDir & "|" & tName & linefeed
		end repeat
	end repeat
	return output
end tell
`

// ListTabs enumerates all open Ghostty tabs and returns Tab structs.
func (a *GhosttyAdapter) ListTabs() ([]Tab, error) {
	out, err := applescript.Run(ghosttyListScript)
	if err != nil {
		if strings.Contains(err.Error(), "not allowed") || strings.Contains(err.Error(), "1002") {
			return nil, fmt.Errorf("ghostty: permission denied. Grant access in System Settings > Privacy & Security > Automation")
		}
		return nil, fmt.Errorf("ghostty: list tabs: %w", err)
	}

	raws := parseGhosttyListOutput(out)
	var tabs []Tab
	for _, r := range raws {
		tabs = append(tabs, Tab{
			ID:           r.TabID,
			WindowID:     r.WindowID,
			Directory:    r.WorkingDir,
			TerminalType: "ghostty",
		})
	}
	return tabs, nil
}

// parseGhosttyListOutput splits AppleScript output into ghosttyRawTab records.
// Lines that don't have exactly 5 pipe-delimited fields are silently skipped.
func parseGhosttyListOutput(output string) []ghosttyRawTab {
	var tabs []ghosttyRawTab
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) != 5 || parts[0] == "" || parts[1] == "" {
			continue
		}
		tabs = append(tabs, ghosttyRawTab{
			WindowID:   parts[0],
			TabID:      parts[1],
			TerminalID: parts[2],
			WorkingDir: parts[3],
			Name:       parts[4],
		})
	}
	return tabs
}

// ghosttyFindTabScript generates AppleScript to locate a tab by its ID.
func ghosttyFindTabScript(tabID string) string {
	return fmt.Sprintf(`
tell application "Ghostty"
	set targetWindow to missing value
	set targetTab to missing value
	repeat with w in windows
		repeat with t in tabs of w
			if id of t is "%s" then
				set targetWindow to w
				set targetTab to t
				exit repeat
			end if
		end repeat
		if targetTab is not missing value then exit repeat
	end repeat
`, tabID)
}

// SwitchTo activates the Ghostty tab matching the given tab ID.
func (a *GhosttyAdapter) SwitchTo(tabID string) error {
	script := ghosttyFindTabScript(tabID) + `
	if targetTab is not missing value then
		select tab targetTab
		activate window targetWindow
	end if
end tell
`
	_, err := applescript.Run(script)
	if err != nil {
		return fmt.Errorf("ghostty: switch to %s: %w", tabID, err)
	}
	return nil
}

// Close closes the Ghostty tab matching the given tab ID.
func (a *GhosttyAdapter) Close(tabID string) error {
	script := ghosttyFindTabScript(tabID) + `
	if targetTab is not missing value then
		close tab targetTab
	end if
end tell
`
	_, err := applescript.Run(script)
	if err != nil {
		return fmt.Errorf("ghostty: close %s: %w", tabID, err)
	}
	return nil
}

// Create opens a new Ghostty tab. If directory is non-empty, the tab's
// initial working directory is set via a cd command.
func (a *GhosttyAdapter) Create(directory string) error {
	var script string
	if directory != "" {
		script = fmt.Sprintf(`tell application "Ghostty"
	activate
	tell front window
		make new tab with properties {command:"cd %s && exec $SHELL"}
	end tell
end tell`, directory)
	} else {
		script = `tell application "Ghostty"
	activate
	tell front window
		make new tab
	end tell
end tell`
	}
	_, err := applescript.Run(script)
	if err != nil {
		return fmt.Errorf("ghostty: create tab in %s: %w", directory, err)
	}
	return nil
}

// Rename is a no-op for Ghostty v1 — tab name properties are read-only
// in Ghostty's scripting dictionary.
func (a *GhosttyAdapter) Rename(tabID string, name string) error {
	return nil
}
