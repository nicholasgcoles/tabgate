# TabGate — Design Spec

## Overview

TabGate is a TUI (terminal UI) application that acts as a companion/overview tool for managing terminal sessions across macOS terminal emulators. Instead of organizing by terminal emulator, TabGate groups sessions by **git project**, showing branch/worktree status and running commands for each session.

Built with Go and Bubble Tea, it provides a lazygit-style interactive interface.

## Target Terminal Emulators

- **Terminal.app** — via AppleScript (`osascript`)
- **Ghostty** — via AppleScript (`osascript`) with Ghostty's scripting dictionary

Both adapters use AppleScript, though with different scripting dictionaries. A shared AppleScript execution helper handles the common `osascript` invocation logic.

The terminal emulator is an invisible implementation detail to the user.

## Architecture

### Adapter Interface

```go
type Tab struct {
    ID              string   // opaque, adapter-specific (Terminal.app uses TTY, Ghostty uses UUIDs)
    WindowID        string
    Directory       string
    RepoRoot        string   // git repo root, empty if not in a repo
    RepoName        string   // derived from repo root directory name
    Branch          string   // current git branch
    IsWorktree      bool
    RunningCommand  string   // foreground process name
    TerminalType    string   // "terminal.app" or "ghostty"
}

type TerminalAdapter interface {
    // ListTabs returns raw tab data: ID, WindowID, Directory, TerminalType.
    // Git and process enrichment is handled separately by the TabEnricher.
    ListTabs() ([]Tab, error)
    SwitchTo(tabID string) error
    Close(tabID string) error
    Create(directory string) error
    Rename(tabID string, name string) error
}
```

`ListTabs()` returns partially populated `Tab` structs (ID, WindowID, Directory, TerminalType). The `TabEnricher` fills in git and process fields.

### Components

1. **TUI Layer** — Bubble Tea model with project-grouped tab list, key bindings, status bar. Polls adapters every 1-2 seconds.
2. **Adapter Interface** — Go interface that each terminal emulator implements.
3. **Terminal.app Adapter** — Calls `osascript` to execute AppleScript for tab discovery and actions. Gets CWD via TTY → `lsof`.
4. **Ghostty Adapter** — Calls `osascript` with Ghostty's AppleScript dictionary. Gets CWD directly from the `working directory` property on Ghostty's `terminal` object (no `lsof` needed).
5. **Tab Enricher** — Takes raw tab data from adapters and enriches with:
   - **Git info:** repo root, name, branch, worktree status
   - **Process info:** foreground command detection
6. **AppleScript Helper** — Shared utility for executing `osascript` commands, used by both adapters.

### Data Flow

1. **Tab Discovery** — Poll each running adapter's `ListTabs()` to get raw tab data (ID, WindowID, Directory, TerminalType).
2. **Directory Detection** — Each adapter handles CWD retrieval differently:
   - Terminal.app: AppleScript TTY → `lsof -p <pid> -Fn` to get CWD from the shell process
   - Ghostty: reads `working directory` property directly via AppleScript
3. **Git Enrichment** — The TabEnricher runs `git rev-parse --show-toplevel` and `git branch --show-current` for each tab's directory. Uses mtime-based caching on `.git/HEAD` to skip unchanged repos.
4. **Worktree Detection** — `git worktree list` to determine if the directory is a worktree. Cached alongside git info.
5. **Process Detection** — Identify the foreground process for each tab's TTY using `ps -o tpgid,pid,comm` and matching the terminal's foreground process group ID. This correctly identifies the actual foreground process even when multiple child processes exist.
6. **Group & Render** — Group tabs by repo root, sort projects alphabetically, render in TUI.

Poll cycle: 1-2 seconds. Git/process caching keeps subprocess invocations manageable even with 10+ tabs.

### Self-Exclusion

TabGate detects its own TTY on startup and excludes it from the tab listing. The user does not see TabGate listed as a session.

## TUI Layout

### Project-Centric View

```
┌─────────────────────────────────────────────────────────┐
│ TabGate                        3 projects · 6 sessions  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ TabGate  ~/Projects/TabGate                             │
│   ❯ main                                  nvim main.go  │
│     feature/tui-layout  [worktree]        claude code   │
│     main                                  go test ./... │
│                                                         │
│ api-server  ~/Projects/api-server                       │
│     fix/auth-bug                          claude code   │
│     main                              docker compose up │
│                                                         │
│ Other                                                   │
│     ~/Downloads                           zsh (idle)    │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ ↑↓/jk navigate  enter switch  n new  r rename  d close  │
└─────────────────────────────────────────────────────────┘
```

### Key Bindings

| Key       | Action                          |
|-----------|---------------------------------|
| `↑`/`k`  | Move cursor up                  |
| `↓`/`j`  | Move cursor down                |
| `enter`  | Switch to selected session      |
| `n`      | Create new tab in selected project's directory |
| `r`      | Rename selected tab (sets terminal custom title) |
| `d`      | Close selected tab              |
| `q`      | Quit TabGate                    |

## Features (v1)

- List all terminal tabs grouped by git project
- Show current branch and worktree status for each tab
- Show the running foreground command for each tab
- Switch focus to any tab
- Create new tabs (opens in the selected project's directory)
- Rename tabs (sets the terminal's custom title property)
- Close tabs
- Auto-detect which terminal emulators are running
- "Other" group for tabs not in a git repo
- Exclude TabGate's own session from the listing

## Scope Exclusions (v1)

- **Split panes** — Not modeled. Only top-level tabs are tracked.
- **Persistence** — TabGate is stateless; all data is derived from live terminal state. Renames use the terminal's native custom title property.

## Error Handling

- **Terminal not running** — Skip that adapter. Empty state with message if no terminals found.
- **Tab closed externally** — Next poll removes it; cursor moves to nearest remaining tab.
- **Not in a git repo** — Tab goes into "Other" group, showing directory path instead of branch.
- **Permission denied** — Show clear message explaining macOS Automation permission setup in System Settings > Privacy & Security > Automation.
- **Stale process info** — Self-corrects on next poll cycle.

## Testing Strategy

- **Unit tests** — Test adapter parsing logic with mocked AppleScript output. Test TabEnricher with fixture data.
- **Integration tests** — Test TUI model: given a set of tabs, verify grouping, sorting, cursor navigation, self-exclusion.
- **Manual testing** — Launch with real Terminal.app tabs, verify all actions work end-to-end.

## Tech Stack

- **Language:** Go
- **TUI Framework:** Bubble Tea (charmbracelet)
- **Styling:** Lip Gloss (charmbracelet)
- **Terminal Communication:** AppleScript via `osascript` (both Terminal.app and Ghostty)
- **Git Info:** `git` CLI commands with mtime-based caching
- **Process Info:** `ps` with TPGID matching for foreground process detection
