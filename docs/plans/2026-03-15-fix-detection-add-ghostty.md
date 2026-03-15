# Fix Terminal Detection + Add Ghostty Adapter

## Context

TabGate builds and tests pass, but `go run ./cmd/tabgate` fails with "No supported terminal emulators detected." Root causes:

1. **`pgrep -x Terminal` doesn't match Terminal.app** — the `-x` exact-match flag fails on macOS because the process name in the process table doesn't match the bare string "Terminal".
2. **Ghostty is running but has no adapter** — `pgrep -x ghostty` succeeds. The design spec planned a Ghostty adapter (deferred to Step 9), now needed.

## Changes

### 1. Fix `internal/adapter/detect.go`
- Terminal.app: change `pgrep -x Terminal` → `pgrep -f Terminal.app` (matches full path)
- Add Ghostty: `pgrep -x ghostty` → include `NewGhosttyAdapter()`

### 2. Create `internal/adapter/ghostty.go`

Ghostty's AppleScript API (verified on this machine):
- **Hierarchy**: application → windows → tabs → terminals
- **Window**: `id` (e.g. `tab-group-c26df26c0`), `name`, `selected tab`
- **Tab**: `id` (e.g. `tab-c29a5ca00`), `name`, `index`, `selected`, `focused terminal`
- **Terminal**: `id` (UUID), `name`, `working directory`
- **Commands**: `new tab`, `select tab`, `close tab`, `activate window`

Implementation:
- `GhosttyAdapter` struct (no self-exclusion needed — Ghostty doesn't expose TTY, so we can't match against own TTY; self-exclusion happens naturally since TabGate runs in Terminal.app or Ghostty won't list a tab without a terminal object)
- `ListTabs()`: AppleScript enumerating windows → tabs → focused terminal. Output format: `windowID|tabID|terminalID|workingDir|name` per line. Parse into Tab structs with `ID=tabID`, `Directory=workingDir`, `TerminalType="ghostty"`.
- `SwitchTo(tabID)`: iterate to find tab by ID, then `select tab` + `activate window`
- `Close(tabID)`: find tab by ID, then `close tab`
- `Create(directory)`: `new tab` with surface configuration setting `initial working directory`
- `Rename(tabID, name)`: Ghostty tab properties are read-only per the scripting dictionary — stub with error or use `perform action` if available. For v1, return nil (no-op).
- Exported `parseGhosttyListOutput(output string) []ghosttyRawTab` for testability

**Process info limitation**: Ghostty doesn't expose TTY paths via AppleScript, so `ResolveForTTY` will fail (error swallowed by enricher). Ghostty tabs will show git info but no running command. This is acceptable for v1.

### 3. Create `internal/adapter/ghostty_test.go`
- Table-driven tests for `parseGhosttyListOutput` (single tab, multiple windows/tabs, empty, malformed)

## Files Modified
- `internal/adapter/detect.go` — fix Terminal.app detection, add Ghostty
- `internal/adapter/ghostty.go` — new file
- `internal/adapter/ghostty_test.go` — new file

## Verification
1. `go build ./...` compiles
2. `go test ./...` passes
3. `go run ./cmd/tabgate` launches TUI showing real tabs from both Terminal.app and Ghostty
