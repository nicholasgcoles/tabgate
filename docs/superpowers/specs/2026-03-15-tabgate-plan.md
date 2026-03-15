# TabGate Implementation Plan

## Context

TabGate is a Go + Bubble Tea TUI app that acts as a companion/overview tool for managing terminal sessions across macOS terminal emulators. It groups sessions by git project (not terminal emulator), shows branch/worktree status and running commands, and supports switching, creating, closing, and renaming tabs. Design spec: `docs/superpowers/specs/2026-03-15-tabgate-design.md`.

## Project Structure

```
tabgate/
├── cmd/tabgate/main.go
├── internal/
│   ├── adapter/
│   │   ├── adapter.go           # Tab struct + TerminalAdapter interface
│   │   ├── terminal_app.go      # Terminal.app adapter (AppleScript + lsof)
│   │   ├── terminal_app_test.go
│   │   ├── detect.go            # Auto-detect running emulators
│   │   └── ghostty.go           # (deferred to Step 9)
│   ├── applescript/
│   │   ├── exec.go              # Shared osascript helper
│   │   └── exec_test.go
│   ├── enricher/
│   │   ├── enricher.go          # Orchestrates git + process enrichment
│   │   ├── enricher_test.go
│   │   ├── git.go               # Git info + mtime caching
│   │   ├── git_test.go
│   │   ├── process.go           # Foreground process via ps TPGID
│   │   └── process_test.go
│   ├── poller/
│   │   ├── poller.go            # Poll cycle: adapters -> enricher -> TUI msg
│   │   └── poller_test.go
│   └── tui/
│       ├── model.go             # Bubble Tea model
│       ├── model_test.go
│       ├── view.go              # Lip Gloss rendering
│       ├── keys.go              # Key bindings
│       └── grouping.go          # Group tabs by project
├── go.mod
└── go.sum
```

## Steps

### Step 1: Go Module and Skeleton
- `go mod init github.com/nic/tabgate`
- Create `cmd/tabgate/main.go` with startup message
- `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss github.com/charmbracelet/bubbles`
- **Verify:** `go run ./cmd/tabgate` prints output and exits

### Step 2: Adapter Interface + AppleScript Helper
- `internal/adapter/adapter.go`: `Tab` struct and `TerminalAdapter` interface (per spec)
- `internal/applescript/exec.go`: `Run(script string) (string, error)` wrapping `osascript -e`
- Unit test with trivial AppleScript `return "hello"`
- **Verify:** `go test ./internal/applescript/...`

### Step 3: Terminal.app Adapter (ListTabs)
- `internal/adapter/terminal_app.go`: implement `ListTabs()`
- AppleScript to iterate `every window` / `every tab`, output `windowID|tty` per line
- CWD via `lsof -a -p <pid> -d cwd -Fn` on shell process from `ps -t <tty>`
- Self-exclusion: detect own TTY at startup, filter it out
- Separate parsing from execution for testability
- **Verify:** `go test ./internal/adapter/...` + manual run printing real tabs

### Step 4: Tab Enricher
- `internal/enricher/git.go`: `git rev-parse --show-toplevel`, `git branch --show-current`, `git worktree list`. Mtime cache on `.git/HEAD`.
- `internal/enricher/process.go`: `ps -t <tty> -o tpgid=,pid=,comm=`, match PID == TPGID for foreground process. Idle shells show `zsh (idle)`.
- `internal/enricher/enricher.go`: `TabEnricher.Enrich(tabs) []Tab` calls git + process for each
- **Verify:** `go test ./internal/enricher/...`

### Step 5: Minimal TUI (Static Snapshot) — MVP
- `internal/tui/grouping.go`: `GroupByProject(tabs) []Project` — group by RepoRoot, sort alphabetically, "Other" at bottom
- `internal/tui/model.go`: Bubble Tea model with cursor, handles j/k/up/down/q
- `internal/tui/view.go`: Lip Gloss rendering matching spec mockup (header, project groups, footer keybinds)
- `internal/tui/keys.go`: key bindings via Bubbles `key.Binding`
- Wire in `main.go`: ListTabs → Enrich → GroupByProject → tea.NewProgram
- **Verify:** `go run ./cmd/tabgate` shows styled TUI with real tabs, navigation works, q quits

### Step 6: Poller (Live Refresh)
- `internal/poller/poller.go`: returns `tea.Cmd` that sleeps, runs adapter+enricher, returns `TabsUpdatedMsg`
- `internal/adapter/detect.go`: `DetectAdapters()` checks which emulators are running via `pgrep`
- Update model: `Init()` starts first poll, `Update()` handles `TabsUpdatedMsg`, preserves cursor by tab ID
- **Verify:** TUI live-updates when tabs are opened/closed externally

### Step 7: Actions (Switch, Create, Close, Rename)
- **Switch (enter):** AppleScript to activate window + set selected tab. Call via async `tea.Cmd`.
- **Create (n):** `do script "cd <dir>"` in Terminal.app. Immediate re-poll after.
- **Close (d):** AppleScript to close tab. Confirmation prompt in status bar (y/n).
- **Rename (r):** Bubbles `textinput` for new name, sets `custom title` via AppleScript.
- **Verify:** Each action works against real Terminal.app tabs

### Step 8: Polish and Edge Cases
- Empty state when no terminals detected
- Permission denied error with System Settings guidance
- Stale cursor handling (tab disappears between polls)
- Visual polish (colors, spacing, borders)
- Update README with usage

### Step 9 (Deferred): Ghostty Adapter
- `internal/adapter/ghostty.go` using Ghostty's AppleScript dictionary
- CWD via `working directory` property (no lsof)
- Tab IDs are Ghostty UUIDs
- Update `detect.go` to include Ghostty

## Key Design Decisions

- **Tab ID = TTY path** for Terminal.app (stable, usable for process lookups)
- **Separate parsing from execution** in all adapter methods for testability
- **Enricher is long-lived** — its git cache persists between poll cycles
- **Cursor preservation** — store selected tab ID, find it after re-grouping, clamp if gone
- **Actions are async** — return `tea.Cmd`, show brief status while running

## Verification

1. `go test ./...` — all unit tests pass
2. `go run ./cmd/tabgate` — TUI renders with real Terminal.app tabs
3. Open/close tabs externally → TUI updates within 2 seconds
4. Press enter → switches to selected tab
5. Press n → new tab created in project directory
6. Press d → tab closed (with confirmation)
7. Press r → rename prompt, title updates
