package applescript

import (
	"fmt"
	"os/exec"
	"strings"
)

// Run executes an AppleScript snippet via osascript and returns its stdout.
// Trailing whitespace/newlines are trimmed. If osascript exits with a non-zero
// status, the returned error includes stderr.
func Run(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("osascript: %s: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("osascript: %w", err)
	}

	return strings.TrimRight(string(out), " \t\r\n"), nil
}
