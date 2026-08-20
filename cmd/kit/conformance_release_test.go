package main

// Conformance: `kit release`. The command has one dangerous half — deleting a branch —
// so most of this drives a real remote and asks git what is left afterwards, rather
// than asserting on messages.

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hasRemoteBranch asks the remote itself, which is where a claim lives.
func hasRemoteBranch(t *testing.T, p *project, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branch)
	cmd.Dir, cmd.Env = p.Root, gitEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("reading the remote: %v", err)
	}
	return strings.Contains(string(out), "refs/heads/"+branch)
}

// workOn puts a commit on the claim branch, which is what turns a claim nobody has
// built on into work that releasing would destroy.
func workOn(t *testing.T, p *project, branch string) {
	t.Helper()
	p.git("checkout", "-q", branch)
	p.write("main.go", "package main\n")
	p.commitAll("0007: build it")
}

func TestReleaseGivesTheFeatureBackToTheBoard(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()

	a.run("release", "0007").ok().has("released", "0007-widget")

	if hasRemoteBranch(t, a, "0007-widget") {
		t.Error("release exited 0 but the remote still has 0007-widget, so the claim stands")
	}
	for _, name := range a.branches() {
		if strings.HasPrefix(name, "0007") {
			t.Errorf("release left branch %q behind, which reads as a claim here", name)
		}
	}
	// The whole point: somebody else can now take it.
	b.run("claim", "0007").ok()
}

func TestReleaseRefusesWhileTheBranchCarriesWork(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	workOn(t, a, "0007-widget")

	a.run("release", "0007").fails(2).errHas("carries work", "pull request")

	if !hasRemoteBranch(t, a, "0007-widget") {
		t.Error("the refusal deleted the claim anyway, which is the work it exists to keep")
	}
}

func TestReleaseForceDeletesWorkOnPurpose(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	workOn(t, a, "0007-widget")
	a.git("checkout", "-q", a.headBranch())

	a.run("release", "0007", "--force").ok()

	if hasRemoteBranch(t, a, "0007-widget") {
		t.Error("--force exited 0 and left the claim on the remote")
	}
}

// The ordinary case for an agent that is stopping: it is standing in the worktree it
// was working in, so the branch it is on cannot be deleted. The claim is still
// released, and saying the branch is gone when it is not would hand the next agent a
// feature this checkout goes on reading as taken.
func TestReleaseSaysWhatIsLeftWhenAWorktreeHoldsTheBranch(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	a.git("worktree", "add", "-q", filepath.Join(t.TempDir(), "wt"), "0007-widget")

	a.run("release", "0007").ok().has("still here", "git branch -D 0007-widget")

	if hasRemoteBranch(t, a, "0007-widget") {
		t.Error("the claim on the remote survived a release that reported success")
	}
}

func TestReleaseRefusalsThatAreNotAboutWork(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)

	// Nothing claims it, so there is nothing to give back.
	a.run("release", "0007").fails(2).errHas("nothing claims")
	// A feature that does not exist at all.
	a.run("release", "0099").fails(2)
}
