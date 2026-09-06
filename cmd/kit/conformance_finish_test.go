package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sanyatihy/project-kit/internal/kit"
)

func TestFinishPlanningPublishesDraftsAndRefusesUnrelatedChanges(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.run("start", "--plan", "next round").ok()
	feature(a, "0008", "next", "Next capability", "Draft", "- [ ] Build next")
	a.write("unrelated.txt", "valuable\n")
	a.run("finish").fails(2).errHas("unrelated file")
	a.git("add", "unrelated.txt")
	a.git("commit", "-m", "unrelated")
	// Committing unrelated work does not make it part of an approved plan.
	a.run("finish").fails(2).errHas("unrelated file")
	a.git("rm", "unrelated.txt")
	a.git("commit", "-m", "remove unrelated change")
	result := a.run("finish", "--json").ok()
	var h kit.Handoff
	if err := json.Unmarshal([]byte(result.Stdout), &h); err != nil {
		t.Fatal(err)
	}
	if !h.Published || h.Base != "main" || h.Candidate != a.gitOut(nil, "rev-parse", "HEAD") {
		t.Fatalf("bad handoff: %+v", h)
	}
	if !strings.Contains(a.read("docs/features/0008-next.md"), "status: Draft") {
		t.Fatal("planning finish changed status")
	}
	head := h.Candidate
	a.run("finish").ok()
	if head != a.gitOut(nil, "rev-parse", "HEAD") {
		t.Fatal("planning retry duplicated commit")
	}
	a.run("claim", "0008").fails(2)
}

func completedGuidedWork(t *testing.T) *project {
	t.Helper()
	a, _ := twoAgents(t)
	a.run("start", "0007").ok()
	w := a.claimWorktree("work/0007")
	feature(w, "0007", "widget", "A widget", "Draft", "- [x] Build it")
	w.commitAll("0007 built")
	return w
}

func TestFinishRunsRealGateAndRetriesWithoutDuplicateSubmission(t *testing.T) {
	t.Parallel()
	w := completedGuidedWork(t)
	w.run("finish").fails(2).errHas("actual gate")
	w.run("finish", "--", "false").fails(2).errHas("gate failed")
	if !strings.Contains(w.read("docs/features/0007-widget.md"), "status: Draft") {
		t.Fatal("failed gate submitted")
	}
	w.run("finish", "--", "true").ok().has("published: true", "independent approved work")
	head := w.gitOut(nil, "rev-parse", "HEAD")
	w.run("finish", "--", "true").ok()
	if head != w.gitOut(nil, "rev-parse", "HEAD") {
		t.Fatal("retry duplicated submission")
	}
	w.run("finish", "--", "false").fails(2).errHas("gate failed")
}

func TestFinishRefusesCandidateChangesDuringGate(t *testing.T) {
	t.Parallel()
	for _, script := range []string{"echo changed > changed.txt", "git commit --allow-empty -m changed"} {
		t.Run(script, func(t *testing.T) {
			w := completedGuidedWork(t)
			w.run("finish", "--", "sh", "-c", script).fails(2).errHas("gate changed")
			if !strings.Contains(w.read("docs/features/0007-widget.md"), "status: Draft") {
				t.Fatal("changed candidate submitted")
			}
		})
	}
}

func TestFinishHarvestsOnceAndRequestsReviewFacts(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.run("start", "--work", "Requested small change").ok()
	p.write("product.txt", "work\n")
	p.run("finish", "--", "true").fails(2).errHas("fill review facts")
	path := "docs/features/0001-requested-small-change.md"
	if !strings.Contains(p.read(path), "- [x] Requested small change") {
		t.Fatal("missing harvested outcome")
	}
	feature(p, "0001", "requested-small-change", "Requested small change", "Draft", "- [x] Requested small change")
	p.commitAll("review facts")
	p.run("finish", "--", "true").ok().has("published: false", "PR review is pending")
	if p.resolves("refs/heads/0002-requested-small-change") {
		t.Fatal("duplicate harvest")
	}
}

func TestFinishRequiresReopenForChangesAfterSubmission(t *testing.T) {
	t.Parallel()
	w := completedGuidedWork(t)
	w.run("finish", "--", "true").ok()
	w.write("review-change.txt", "changed after submission\n")
	w.commitAll("review change")
	w.run("finish", "--", "true").fails(2).errHas("kit reopen 0007")
	w.run("reopen", "0007", "review requested change").ok()
	w.run("finish", "--", "true").ok()
}
