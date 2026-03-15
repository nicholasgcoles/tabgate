package enricher

import (
	"fmt"
	"os/exec"
	"strings"
)

// shells is the set of known shell process names (including login-shell variants).
var shells = map[string]bool{
	"bash":  true,
	"zsh":   true,
	"fish":  true,
	"sh":    true,
	"tcsh":  true,
	"ksh":   true,
	"-bash": true,
	"-zsh":  true,
	"-fish": true,
	"-sh":   true,
}

// ResolveForTTY returns the foreground process name for the given TTY.
// If the foreground process is a shell, it returns "<shell> (idle)".
func ResolveForTTY(tty string) (string, error) {
	out, err := exec.Command("ps", "-t", tty, "-o", "tpgid=,pid=,comm=").Output()
	if err != nil {
		return "", fmt.Errorf("ps command failed: %w", err)
	}
	return ParsePsOutput(string(out)), nil
}

// ParsePsOutput parses `ps -o tpgid=,pid=,comm=` output and returns the
// foreground process name. If the foreground process is a shell, returns
// "<shell> (idle)".
func ParsePsOutput(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		tpgid := fields[0]
		pid := fields[1]
		// comm may contain spaces, so rejoin everything after pid.
		comm := strings.Join(fields[2:], " ")

		if pid == tpgid {
			// This is the foreground process.
			base := comm
			// Use just the basename if it's a path.
			if idx := strings.LastIndex(comm, "/"); idx >= 0 {
				base = comm[idx+1:]
			}
			if shells[base] {
				return base + " (idle)"
			}
			return base
		}
	}
	return ""
}
