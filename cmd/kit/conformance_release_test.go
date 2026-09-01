package main

// Conformance: `kit release`. The command has one dangerous half — deleting a branch —
// so most of this drives a real remote and asks git what is left afterwards, rather
// than asserting on messages.

import (
	"os/exec"
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
// workOn puts a commit on the claim branch, in the worktree `kit claim` made: the trunk
// checkout cannot switch to a branch that worktree already holds.
func workOn(t *testing.T, p *project, branch string) *project {
	t.Helper()
	w := p.claimWorktree(branch)
	w.write("main.go", "package main\n")
	w.commitAll("0007: build it")
	return w
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
	w := a.claimWorktree("0007-widget")

	// Released from inside the worktree it was working in. Git will not remove the
	// checkout you are standing in, so the branch outlives the claim and release says so.
	w.run("release", "0007").ok().has("still here")

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

// The claim-theft chain, end to end. An agent that loses the race can see the winner's
// branch on the remote but never created it, so releasing it is refused. Without this,
// the losing agent could free the feature and claim it, and both checkouts were then
// told the feature was theirs.
func TestReleaseRefusesAClaimThisCloneNeverCreated(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("fetch", "-q", "origin")

	b.run("release", "0007").fails(2).errHas("never created", "--force")

	if !hasRemoteBranch(t, a, "0007-widget") {
		t.Fatal("the refusal deleted the winner's claim anyway")
	}
	// And the winner still holds it: the loser cannot get in by claiming again.
	b.run("claim", "0007").fails(1)
}

// The refusal `claim` prints must not name a command that takes somebody's branch.
// It named `git push --delete`, which has neither the work guard nor the ownership
// check, and a losing agent that ran it won the feature on the next attempt.
func TestLosingAClaimNeverPointsAtARawBranchDelete(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("fetch", "-q", "origin")

	lost := b.run("claim", "0007").fails(1)
	lost.errHas("kit release 0007")
	if strings.Contains(lost.Stderr, "--delete") {
		t.Errorf("the losing agent is told how to delete the winner's branch:\n%s", lost.Stderr)
	}
}

// `BranchBuildsOn` answers "carries no work" when no trunk resolves, and that is the
// answer that deletes. A project whose default branch is neither `main` nor `master`
// reached that state from any claim checkout, and released real commits at exit 0.
func TestReleaseRefusesWhenNoTrunkResolves(t *testing.T) {
	t.Parallel()
	a, _ := twoAgentsOn(t, "trunk")
	a.run("claim", "0007").ok()
	workOn(t, a, "0007-widget")
	a.git("push", "-q", "origin", "0007-widget")

	a.run("release", "0007").fails(2)

	if !hasRemoteBranch(t, a, "0007-widget") {
		t.Fatal("release deleted a branch carrying work on a repo with no resolvable trunk")
	}
}

func TestATagNamedLikeTheClaimDoesNotStrandIt(t *testing.T) {
	t.Parallel()
	// A tag and a branch may share a name, and git then writes the branch as
	// `heads/0007-widget` wherever it shortens a ref for a reader. Read as the claiming
	// branch, that name matches nothing: `release` said it was a branch this clone never
	// created and refused for as long as the tag existed, so only `--force` got out — and
	// `--force` is the flag for destroying an agent's live claim.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	trunk := a.headBranch()
	a.git("tag", "0007-widget", trunk)

	// The claim is still the claim-holder's, read from the checkout standing on it.
	w := a.claimWorktree("0007-widget")
	if got := boardGroup(w.plan("next", "--json"), "0007"); got != "yours" {
		t.Errorf("under a tag of the same name the board says 0007 is %s, want yours", got)
	}

	a.run("release", "0007").ok().has("released", "0007-widget")
	if hasRemoteBranch(t, a, "0007-widget") {
		t.Error("release exited 0 but the remote still has 0007-widget, so the claim stands")
	}
}

// The deleted-commit case, reproduced: one checkout's copy of the claim was left behind
// while another pushed real work to the same branch. `release` read the copy, called the
// branch empty, and deleted the ref carrying the commits — exit 0, no `--force`, and the
// work unreachable.
func TestReleaseRefusesWhenTheRemoteHasCommitsThisCheckoutHasNotSeen(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("fetch", "-q", "origin")
	b.git("checkout", "-q", "-b", "0007-widget", "origin/0007-widget")
	b.write("main.go", "package main\n")
	b.commitAll("0007: build it")
	b.git("push", "-q", "origin", "0007-widget")

	a.run("release", "0007").fails(2).errHas("git fetch")

	if !hasRemoteBranch(t, a, "0007-widget") {
		t.Fatal("release deleted a branch whose commits this checkout had never seen")
	}
}
