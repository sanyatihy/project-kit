package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// The reviewer is a separate clone acting as the human in this fixture. This exercises
// the git handoff; it does not certify permissions on a hosted remote.
func TestApprovedRoundSurvivesClaimsQuestionsReviewAndSquashIntegration(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	human := projectAt(t, filepath.Join(filepath.Dir(a.Root), "reviewer"))
	a.git("clone", "-q", filepath.Join(filepath.Dir(a.Root), "origin.git"), human.Root)
	a.git("checkout", "-b", "plan/next-round")
	a.run("new", "Independent improvement").ok()
	a.run("new", "Dependent improvement").ok()
	independent := "docs/features/0008-independent-improvement.md"
	dependent := "docs/features/0009-dependent-improvement.md"
	a.write(independent, doc("0008", "Independent improvement", "Draft", "needs:", "- [ ] Deliver independently"))
	a.write(dependent, doc("0009", "Dependent improvement", "Draft", "needs: 0007", "- [ ] Build on the widget"))
	a.commitAll("propose next round")
	a.git("push", "-u", "origin", "HEAD")
	a.run("claim", "0008").fails(2)

	human.git("fetch", "origin")
	human.git("merge", "--squash", "origin/plan/next-round")
	human.commitAll("approve next round")
	human.git("push", "origin", "main")
	for _, p := range []*project{a, b} {
		p.git("checkout", "main")
		p.git("pull", "--ff-only", "origin", "main")
		p.run("audit", "--strict").ok()
	}
	a.run("claim", "0009").fails(1)
	a.run("claim", "0007").ok()
	b.run("claim", "0008").ok()
	b.run("claim", "0007").fails(1)

	// A restart resumes the existing worktree. Surrendering an untouched claim
	// discards only its bookkeeping, and the same clone can acquire it again.
	a.run("claim", "0007").ok()
	a.run("release", "0007").ok()
	a.run("claim", "0007").ok()
	aw, bw := a.claimWorktree("work/0007"), b.claimWorktree("work/0008")
	// `block` commits the question itself, so there is nothing left in the tree to record.
	aw.run("block", "0007", "Which output should the widget provide? Leaning: compact.").ok()
	aw.git("push", "origin", "HEAD")
	aw.run("next").ok().has("blocked", "Which output")
	aw.run("submit", "0007").fails(2)

	// The agent records the human's answer in the paused checkout before resuming: the
	// decision goes in the plan, and ticking the box is what unblocks the feature.
	widget := "docs/features/0007-widget.md"
	var answered []string
	for _, line := range strings.Split(aw.read(widget), "\n") {
		if strings.Contains(line, "Which output should the widget provide?") {
			line = strings.Replace(line, "- [ ]", "- [x]", 1)
		}
		answered = append(answered, line)
	}
	aw.write(widget, strings.Join(answered, "\n"))
	aw.write(widget, strings.Replace(aw.read(widget), "## Design", "## Design\nThe human selected compact output.", 1))
	aw.commitAll("record the human decision and clear its blocker")
	aw.run("next").ok().has("Build it").lacks("Which output")
	aw.write("widget.txt", "compact\n")
	aw.write(widget, strings.Replace(aw.read(widget), "- [ ]", "- [x]", 1))
	aw.commitAll("provide compact output")
	aw.run("audit", "--strict").ok()
	aw.run("submit", "0007").ok()
	aw.git("push", "origin", "HEAD")
	a.run("claim", "0009").fails(1)

	aw.run("reopen", "0007", "human review requests a clearer output").ok()
	aw.write(widget, strings.Replace(aw.read(widget), "## Tasks", "## Tasks\n- [ ] Clarify the output", 1))
	aw.commitAll("plan the review correction")
	aw.write("widget.txt", "compact widget\n")
	aw.write(widget, strings.Replace(aw.read(widget), "- [ ] Clarify", "- [x] Clarify", 1))
	aw.commitAll("clarify the output")
	aw.run("audit", "--strict").ok()
	aw.run("submit", "0007").ok()
	aw.git("push", "origin", "HEAD")
	aw.run("next").ok().has("review")

	bw.write("independent.txt", "independent\n")
	bw.write(independent, strings.Replace(bw.read(independent), "- [ ]", "- [x]", 1))
	bw.commitAll("deliver the independent improvement")
	bw.run("audit", "--strict").ok()
	bw.run("submit", "0008").ok()
	bw.git("push", "origin", "HEAD")
	human.git("fetch", "origin")
	for _, branch := range []string{"work/0007", "work/0008"} {
		human.git("merge", "--squash", "origin/"+branch)
		human.commitAll("approve " + branch)
		human.git("push", "origin", "main")
	}
	a.git("pull", "--ff-only", "origin", "main")
	a.run("audit", "--strict").ok()
	aw.run("reopen", "0007", "must be new work now").fails(2)
	a.run("claim", "0009").ok()
	cw := a.claimWorktree("work/0009")
	if got := cw.read("widget.txt"); got != "compact widget\n" {
		t.Fatalf("unlocked dependency without the reviewed result: %q", got)
	}
	cw.run("next").ok().has("Build on the widget")
}

func TestInitialMVPReachesReviewAndIntegratesAcrossClones(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	p.run("init").ok()
	p.git("branch", "-m", "main")
	writeVision(p)
	p.write("prototype.txt", "useful exploration\n")
	p.run("harvest", "Useful prototype", "--box", "The prototype demonstrates its outcome").ok()
	featurePath := "docs/features/0001-useful-prototype.md"
	p.write(featurePath, doc("0001", "Useful prototype", "Draft", "needs:", "- [x] The prototype demonstrates its outcome"))
	p.commitAll("explain the explored result")
	base := strings.TrimSpace(p.gitOut(nil, "rev-list", "--max-parents=0", "HEAD"))
	if files := p.gitOut(nil, "ls-tree", "--name-only", base); strings.TrimSpace(files) != "" {
		t.Fatal("initial harvest put product files on the review base")
	}
	remote := filepath.Join(t.TempDir(), "origin.git")
	p.git("init", "--bare", "-b", "main", remote)
	p.git("remote", "add", "origin", remote)
	p.git("push", "origin", base+":refs/heads/main")
	p.run("audit", "--strict").ok()
	p.run("submit", "0001").ok()
	p.git("push", "-u", "origin", "HEAD")
	human := projectAt(t, filepath.Join(t.TempDir(), "reviewer"))
	p.git("clone", "-q", remote, human.Root)
	if reviewDoc := human.gitOut(nil, "show", "origin/0001-useful-prototype:"+featurePath); !strings.Contains(reviewDoc, "status: Submitted") {
		t.Fatal("the reviewer cannot read the submitted MVP")
	}
	human.git("merge", "--squash", "origin/0001-useful-prototype")
	human.commitAll("approve the initial MVP")
	human.git("push", "origin", "main")
	human.run("audit", "--strict").ok()
	p.git("fetch", "origin")
	p.run("reopen", "0001", "already integrated").fails(2)
}
