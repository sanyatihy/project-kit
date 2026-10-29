package main

// Conformance: `kit pause`. The pause rule's four clauses as one command, so the half that an
// agent could get wrong silently — the push — is verified rather than remembered.

import (
	"strings"
	"testing"
)

// A remote, because the push is the clause an agent forgets and cannot tell it forgot.
// twoAgents' own feature carries one box; a pause reports what is left, so it needs more.
func paused(t *testing.T) (*project, *project) {
	t.Helper()
	a, _ := twoAgents(t)
	feature(a, "0007", "widget", "A widget", "Draft",
		"- [x] The POST handler\n- [ ] The redirect\n- [ ] The expiry")
	a.commitAll("more boxes")
	a.git("push", "-q", "origin", "HEAD:refs/heads/main")
	a.run("claim", "0007").ok()
	return a, a.claimWorktree("work/0007")
}

func TestPauseCommitsPushesAndNamesWhatRemains(t *testing.T) {
	t.Parallel()
	p, w := paused(t)
	w.write("handler.go", "package main // half done\n")

	r := w.run("pause", "the expiry needs the operator's ceiling").ok()
	r.has("paused", "0007", "the expiry needs the operator's ceiling")
	// What remains is the unticked boxes, so the pull request can say it without
	// anybody rereading the doc.
	r.has("The redirect", "The expiry")
	r.lacks("The POST handler")

	// The tree is committed, not only the doc: half-finished work is the thing a pause
	// exists to preserve.
	if got := w.gitOut(nil, "status", "--porcelain"); got != "" {
		t.Errorf("pause left work in the tree:\n%s", got)
	}
	if got := w.read("docs/features/0007-widget.md"); !strings.Contains(got, "— paused: the expiry") {
		t.Errorf("the reason is not in the Log:\n%s", got)
	}
	// Pushed, which is the clause an agent forgets and cannot tell it forgot.
	local := w.gitOut(nil, "rev-parse", "HEAD")
	if remote := p.gitOut(nil, "rev-parse", "origin/work/0007"); remote != local {
		t.Errorf("origin/work/0007 = %s, want the paused commit %s", remote, local)
	}
}

func TestPauseKeepsTheClaimAndDoesNotSubmit(t *testing.T) {
	t.Parallel()
	p, w := paused(t)
	w.run("pause", "waiting on a decision").ok()

	// Not release: the feature is still this checkout's, so nobody else is offered it.
	if got := w.read("docs/features/0007-widget.md"); strings.Contains(got, "status: Submitted") {
		t.Error("pause submitted the feature, which tells a reviewer it is finished")
	}
	r := p.run("next").ok()
	r.lacks("start here")
	r.has("0007")
}

func TestPauseNeedsAReasonAndAClaim(t *testing.T) {
	t.Parallel()
	// A pause nobody can read is a branch nobody comes back to.
	_, w := paused(t)
	w.run("pause").fails(2)
	w.run("pause", "   ").fails(2)

	// The trunk claims nothing, so there is nothing there to set down.
	p := setUp(t)
	feature(p, "0007", "widget", "A widget", "Draft", "- [ ] The POST handler")
	p.commitAll("a plan")
	p.run("pause", "nothing is held here").fails(2).errHas("claims no feature")
}

func TestNextShowsAPausedClaimAsStillYoursWithItsReason(t *testing.T) {
	t.Parallel()
	// The agent returning to this checkout reads the board first, and a pause is not a
	// status, so nothing else on it would say why the work stopped.
	_, w := paused(t)
	w.run("pause", "the expiry needs the operator's ceiling").ok()

	r := w.run("next").ok()
	r.has("yours", "0007", "paused:", "the expiry needs the operator's ceiling")
	// Still the next box: a pause explains the stop, it does not remove the work.
	r.has("The redirect")
}

func TestNextStopsReportingAPauseOnceWorkResumes(t *testing.T) {
	t.Parallel()
	// The Log holds every pause the feature ever had, so the entry alone would read as
	// paused for ever. The branch is what answers it: `pause` commits the tree, so
	// anything committed on top of that ends the pause.
	_, w := paused(t)
	w.run("pause", "waiting on a decision").ok()
	w.run("next").ok().has("paused:")

	w.write("redirect.go", "package main\n")
	w.commitAll("build the redirect")

	r := w.run("next").ok()
	r.lacks("paused:")
	// Still this checkout's work, with a task — resuming changes nothing but the pause.
	r.has("yours", "0007", "The redirect")
}
