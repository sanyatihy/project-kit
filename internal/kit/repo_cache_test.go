package kit

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// gitIn runs git in a throwaway repository with an identity of its own.
//
// Nothing may be inherited from the machine: a runner with no `user.name` cannot commit at
// all, which is how this failed in CI while passing on a laptop that has one. The config
// files are neutralised for the same reason — a developer's global settings must not decide
// whether a test passes.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_COUNT=0",
		"GIT_AUTHOR_NAME=kit tests", "GIT_AUTHOR_EMAIL=kit@example.invalid",
		"GIT_COMMITTER_NAME=kit tests", "GIT_COMMITTER_EMAIL=kit@example.invalid",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// The two answers that cannot change while one command runs, and were being asked once per
// feature. Rather than count subprocesses, each test changes the repository underneath a
// loaded Repo: if the second call still returns the first answer, no second git ran.
func TestTheBranchAndRemotesAreAskedOnce(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
		{"remote", "add", "origin", "https://example.invalid/one.git"},
	} {
		gitIn(t, root, args...)
	}
	r, err := Open(root)
	if err != nil {
		t.Fatalf("opening: %v", err)
	}

	if got := r.HeadBranch(); got != "main" {
		t.Fatalf("HeadBranch() = %q, want main", got)
	}
	if got := r.remotes(); len(got) != 1 || got[0] != "origin" {
		t.Fatalf("remotes() = %v, want [origin]", got)
	}

	// Move both underneath it. A cached answer is the whole point; a re-read would see
	// the new state and cost a process per feature, which is what this prevents.
	for _, args := range [][]string{
		{"checkout", "-q", "-b", "elsewhere"},
		{"remote", "add", "second", "https://example.invalid/two.git"},
	} {
		gitIn(t, root, args...)
	}

	if got := r.HeadBranch(); got != "main" {
		t.Errorf("HeadBranch() re-read git and returned %q; it should answer from the first call", got)
	}
	if got := r.remotes(); len(got) != 1 {
		t.Errorf("remotes() re-read git and returned %v; it should answer from the first call", got)
	}
}

func TestGitCommandsHooksRunInsideGetTheLongBound(t *testing.T) {
	t.Parallel()
	for args, want := range map[string]time.Duration{
		"commit --quiet -m x":                    hookTimeout,
		"push --porcelain origin x":              hookTimeout,
		"worktree add p b":                       hookTimeout,
		"-c maintenance.auto=false fetch origin": gitTimeout,
		"log -1":                                 gitTimeout,
	} {
		if got := timeoutFor(strings.Fields(args)); got != want {
			t.Errorf("timeoutFor(%q) = %s, want %s", args, got, want)
		}
	}
}
