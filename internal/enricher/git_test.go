package enricher

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name: "single worktree",
			input: `/Users/nic/project  abc1234 [main]
`,
			expect: []string{"/Users/nic/project"},
		},
		{
			name: "multiple worktrees",
			input: `/Users/nic/project       abc1234 [main]
/Users/nic/project-wt   def5678 [feature]
`,
			expect: []string{"/Users/nic/project", "/Users/nic/project-wt"},
		},
		{
			name:   "empty output",
			input:  "",
			expect: nil,
		},
		{
			name: "bare repo entry",
			input: `/Users/nic/project.git  (bare)
/Users/nic/project-wt  abc1234 [main]
`,
			expect: []string{"/Users/nic/project.git", "/Users/nic/project-wt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseWorktreeList(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("expected %d paths, got %d: %v", len(tt.expect), len(got), got)
			}
			for i := range got {
				if got[i] != tt.expect[i] {
					t.Errorf("path[%d]: expected %q, got %q", i, tt.expect[i], got[i])
				}
			}
		})
	}
}

func TestGitResolverCache(t *testing.T) {
	// Create a temp directory with a git repo.
	// Resolve symlinks to handle macOS /var -> /private/var.
	dir, _ := filepath.EvalSymlinks(t.TempDir())

	cmd := exec.Command("git", "init", dir)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	// Configure user for the test repo.
	for _, args := range [][]string{
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("config failed: %v\n%s", err, out)
		}
	}

	// Create an initial commit so branch exists.
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

	resolver := NewGitResolver()

	// First call — populates cache.
	repoRoot, repoName, branch, _, err := resolver.Resolve(dir)
	if err != nil {
		t.Fatalf("first Resolve failed: %v", err)
	}
	if repoRoot != dir {
		t.Errorf("expected repoRoot %q, got %q", dir, repoRoot)
	}
	if repoName != filepath.Base(dir) {
		t.Errorf("expected repoName %q, got %q", filepath.Base(dir), repoName)
	}
	if branch == "" {
		t.Error("expected non-empty branch")
	}

	// Second call — should use cache. We verify by checking the cache entry exists.
	resolver.mu.Lock()
	_, cached := resolver.cache[dir]
	resolver.mu.Unlock()
	if !cached {
		t.Fatal("expected cache entry to exist after first Resolve")
	}

	repoRoot2, _, _, _, err := resolver.Resolve(dir)
	if err != nil {
		t.Fatalf("second Resolve failed: %v", err)
	}
	if repoRoot2 != repoRoot {
		t.Errorf("cache miss: expected %q, got %q", repoRoot, repoRoot2)
	}
}
