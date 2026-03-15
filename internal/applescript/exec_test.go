package applescript_test

import (
	"testing"

	"github.com/nic/tabgate/internal/applescript"
)

func TestRun_Hello(t *testing.T) {
	out, err := applescript.Run(`return "hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello" {
		t.Fatalf("expected %q, got %q", "hello", out)
	}
}

func TestRun_InvalidScript(t *testing.T) {
	_, err := applescript.Run(`this is not valid applescript at all`)
	if err == nil {
		t.Fatal("expected error for invalid script, got nil")
	}
}
