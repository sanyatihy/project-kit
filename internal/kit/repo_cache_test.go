package kit

import (
	"os/exec"
	"testing"
)

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
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
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
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	if got := r.HeadBranch(); got != "main" {
		t.Errorf("HeadBranch() re-read git and returned %q; it should answer from the first call", got)
	}
	if got := r.remotes(); len(got) != 1 {
		t.Errorf("remotes() re-read git and returned %v; it should answer from the first call", got)
	}
}
