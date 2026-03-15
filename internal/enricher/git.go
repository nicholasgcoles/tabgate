package enricher

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type gitCacheEntry struct {
	repoRoot   string
	repoName   string
	branch     string
	isWorktree bool
	headMtime  time.Time
}

// GitResolver resolves git metadata for directories, with mtime-based caching.
type GitResolver struct {
	mu    sync.Mutex
	cache map[string]*gitCacheEntry
}

// NewGitResolver creates a new GitResolver with an empty cache.
func NewGitResolver() *GitResolver {
	return &GitResolver{
		cache: make(map[string]*gitCacheEntry),
	}
}

// Resolve returns git metadata for the given directory.
// Results are cached and invalidated when .git/HEAD modtime changes.
func (g *GitResolver) Resolve(dir string) (repoRoot, repoName, branch string, isWorktree bool, err error) {
	headPath, headMtime, err := g.getHeadInfo(dir)
	if err != nil {
		return "", "", "", false, err
	}

	g.mu.Lock()
	if entry, ok := g.cache[dir]; ok && entry.headMtime.Equal(headMtime) {
		g.mu.Unlock()
		return entry.repoRoot, entry.repoName, entry.branch, entry.isWorktree, nil
	}
	g.mu.Unlock()

	// Resolve repo root.
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", "", false, err
	}
	repoRoot = strings.TrimSpace(string(out))
	repoName = path.Base(repoRoot)

	// Resolve current branch.
	out, err = exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	if err != nil {
		// Not fatal — might be detached HEAD.
		branch = ""
	} else {
		branch = strings.TrimSpace(string(out))
	}

	// Determine if this is a worktree.
	out, err = exec.Command("git", "-C", dir, "worktree", "list").Output()
	if err != nil {
		isWorktree = false
	} else {
		worktrees := ParseWorktreeList(string(out))
		if len(worktrees) > 0 {
			// The first entry is always the main worktree.
			// If dir's repo root differs from the main worktree, it's a linked worktree.
			mainWorktree := worktrees[0]
			isWorktree = repoRoot != mainWorktree
		}
	}

	entry := &gitCacheEntry{
		repoRoot:   repoRoot,
		repoName:   repoName,
		branch:     branch,
		isWorktree: isWorktree,
		headMtime:  headMtime,
	}
	g.mu.Lock()
	g.cache[dir] = entry
	g.mu.Unlock()

	_ = headPath // used for stat only
	return repoRoot, repoName, branch, isWorktree, nil
}

// getHeadInfo finds the .git/HEAD path and its modtime.
// Handles both normal repos (.git is a directory) and worktrees (.git is a file).
func (g *GitResolver) getHeadInfo(dir string) (string, time.Time, error) {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return "", time.Time{}, err
	}

	var headPath string
	if info.IsDir() {
		// Normal repo.
		headPath = filepath.Join(gitPath, "HEAD")
	} else {
		// Worktree: .git is a file containing "gitdir: <path>".
		data, err := os.ReadFile(gitPath)
		if err != nil {
			return "", time.Time{}, err
		}
		content := strings.TrimSpace(string(data))
		gitdir := strings.TrimPrefix(content, "gitdir: ")
		if !filepath.IsAbs(gitdir) {
			gitdir = filepath.Join(dir, gitdir)
		}
		headPath = filepath.Join(gitdir, "HEAD")
	}

	headInfo, err := os.Stat(headPath)
	if err != nil {
		return "", time.Time{}, err
	}
	return headPath, headInfo.ModTime(), nil
}

// ParseWorktreeList parses the output of `git worktree list` and returns
// the list of worktree paths.
func ParseWorktreeList(output string) []string {
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Each line looks like: /path/to/worktree  <hash> [branch]
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			paths = append(paths, fields[0])
		}
	}
	return paths
}
