package enricher

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nic/tabgate/internal/adapter"
)

func TestEnrichWithGitRepo(t *testing.T) {
	// Create a temp git repo.
	// Resolve symlinks to handle macOS /var -> /private/var.
	dir, _ := filepath.EvalSymlinks(t.TempDir())

	cmd := exec.Command("git", "init", dir)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	for _, args := range [][]string{
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("config failed: %v\n%s", err, out)
		}
	}

	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", dir, "add", ".").CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", dir, "commit", "-m", "init").CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}

	enricher := NewTabEnricher()

	tabs := []adapter.Tab{
		{
			ID:        "", // Empty ID to skip process resolution (no real TTY).
			Directory: dir,
		},
	}

	result := enricher.Enrich(tabs)

	if result[0].RepoRoot != dir {
		t.Errorf("expected RepoRoot %q, got %q", dir, result[0].RepoRoot)
	}
	if result[0].RepoName != filepath.Base(dir) {
		t.Errorf("expected RepoName %q, got %q", filepath.Base(dir), result[0].RepoName)
	}
	if result[0].Branch == "" {
		t.Error("expected non-empty Branch")
	}
}

func TestEnrichWithNoDirectory(t *testing.T) {
	enricher := NewTabEnricher()

	tabs := []adapter.Tab{
		{
			ID:        "",
			Directory: "",
		},
	}

	result := enricher.Enrich(tabs)

	if result[0].RepoRoot != "" {
		t.Errorf("expected empty RepoRoot, got %q", result[0].RepoRoot)
	}
}
