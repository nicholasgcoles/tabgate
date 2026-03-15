package adapter

import (
	"os"
	"os/exec"
	"strings"
)

// ttyname returns the TTY device path for the current process's stdin,
// or an empty string if detection fails (e.g., not running in a terminal).
func ttyname() string {
	// On macOS, /dev/fd/0 is a symlink to the actual TTY device.
	target, err := os.Readlink("/dev/fd/0")
	if err == nil && strings.HasPrefix(target, "/dev/tty") {
		return target
	}

	// Fallback: run the `tty` command.
	out, err := exec.Command("tty").Output()
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(out))
	if strings.HasPrefix(s, "/dev/") {
		return s
	}
	return ""
}
