package main

import (
	"encoding/json"
	"path/filepath"
	"strconv"
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
	if !h.Published || h.Base != "main" || h.PRTitle != "Plan: next round" || h.Candidate != a.gitOut(nil, "rev-parse", "HEAD") {
		t.Fatalf("bad handoff: %+v", h)
	}
	if !strings.Contains(a.read("docs/features/0008-next.md"), "status: Draft") {
		t.Fatal("planning finish changed status")
	}
	head := h.Candidate
	a.run("finish").ok().has("pr_title: Plan: next round")
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

func TestReviewTitleComesFromTheFeatureAndPreservesTheClaimReceipt(t *testing.T) {
	t.Parallel()
	w := completedGuidedWork(t)
	receipt := strings.Split(w.gitOut(nil, "rev-list", "--reverse", "main..HEAD"), "\n")[0]
	if subject := w.gitOut(nil, "show", "-s", "--format=%s", receipt); !strings.HasPrefix(subject, "Kit claim 0007 ") {
		t.Fatalf("fixture must begin with a claim receipt: %s", subject)
	}
	const title = "Keep `claim` & \"review\" separate"
	feature(w, "0007", "widget", title, "Draft", "- [x] Build it")
	w.commitAll("Update review facts")
	result := w.run("finish", "--json", "--", "true").ok()
	var h kit.Handoff
	if err := json.Unmarshal([]byte(result.Stdout), &h); err != nil {
		t.Fatal(err)
	}
	if h.PRTitle != title || !strings.Contains(h.Next, "Set its title explicitly from pr_title") {
		t.Fatalf("missing explicit feature title: %+v", h)
	}
	w.run("finish", "--", "true").ok().has("pr_title: " + title)
	if first := strings.Split(w.gitOut(nil, "rev-list", "--reverse", "main..HEAD"), "\n")[0]; first != receipt {
		t.Fatal("handoff rewrote the claim receipt")
	}
	plan := w.plan("next", "--json")
	if plan.Action == nil || !strings.Contains(plan.Action.Instruction, "explicit PR title "+strconv.Quote(title)) {
		t.Fatalf("next lost the review title: %+v", plan.Action)
	}
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
	p.run("finish", "--", "true").ok().has("pr_title: Requested small change", "published: false", "PR review is pending")
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

func TestFinishSkipsTheGateForAProseOnlyRevisionAndNothingElse(t *testing.T) {
	t.Parallel()
	const doc = "docs/features/0007-widget.md"
	revise := func(w *project, from, to string) {
		w.t.Helper()
		w.write(doc, strings.Replace(w.read(doc), from, to, 1))
		w.commitAll("revise " + to)
	}
	w := completedGuidedWork(t)
	w.run("finish", "--", "true").ok()

	// The gate that would fail proves it never ran.
	w.run("reopen", "0007", "trim the prose").ok()
	if strings.Contains(w.read(doc), "trim the prose") {
		t.Fatal("reopen wrote its reason into the doc")
	}
	revise(w, "Something needs doing.", "Something short.")
	w.run("finish", "--", "false").ok().has("gate: skipped", "published: true")
	if !strings.Contains(w.read(doc), "status: Submitted") {
		t.Fatal("prose-only finish did not submit")
	}

	// A Tasks change is a plan change, so the gate runs.
	w.run("reopen", "0007", "another box").ok()
	revise(w, "- [x] Build it", "- [x] Build it\n- [x] Test it")
	w.run("finish", "--", "false").fails(2).errHas("gate failed")
	w.run("finish", "--", "true").ok().lacks("gate: skipped")

	// So does any file beside the doc.
	w.run("reopen", "0007", "review change").ok()
	w.write("review-change.txt", "changed after submission\n")
	revise(w, "Something short.", "Something shorter.")
	w.run("finish", "--", "false").fails(2).errHas("gate failed")
}

func TestFinishPublishesTheEmptyBaseWhenRemoteIsAddedLater(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.run("start", "--work", "Requested change").ok()
	p.write("product.txt", "kept\n")
	p.run("finish", "--", "true").fails(2).errHas("fill review facts")
	feature(p, "0001", "requested-change", "Requested change", "Draft", "- [x] Requested change")
	p.commitAll("facts")
	p.run("finish", "--", "true").ok().has("published: false")
	origin := p.path("../later-origin.git")
	gitIn(t, p.Root, "init", "--bare", origin)
	p.git("remote", "add", "origin", origin)
	p.run("finish", "--", "true").ok().has("published: true")
	if !hasRemoteBranch(t, p, "main") {
		t.Fatal("no first-PR base")
	}
}

func TestReviewHandoffDirectsContinuationWithoutInventingAPullRequest(t *testing.T) {
	t.Parallel()
	w := completedGuidedWork(t)
	w.run("finish", "--", "true").ok()
	plan := w.plan("next", "--json")
	if plan.Action == nil || plan.Action.Actor != "agent" || plan.Action.Kind != "review_handoff" || !strings.Contains(plan.Action.Instruction, "not inferred") || !strings.Contains(plan.Action.Instruction, "independent approved work") {
		t.Fatalf("missing handoff or continuation: %+v", plan.Action)
	}
	w.run("next").ok().has("actor: agent", "action: review_handoff", "independent approved work")
}

func TestFinishRecoversPartialHarvestWithoutAnotherID(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.run("start", "--work", "Small fix").ok()
	p.write("work.txt", "valuable\n")
	p.withGit("if [ \"$3\" = branch ] && [ \"$4\" = -m ]; then exit 9; fi", "finish", "--", "true").fails(2)
	p.run("finish", "--", "true").fails(2).errHas("fill review facts")
	if p.headBranch() != "0001-small-fix" {
		t.Fatal("did not resume the first harvest")
	}
	if p.read("work.txt") != "valuable\n" {
		t.Fatal("lost work")
	}
	if got := p.gitOut(nil, "ls-tree", "-r", "--name-only", "HEAD", "--", "docs/features"); strings.Contains(got, "0002-") {
		t.Fatal("allocated a duplicate harvest")
	}
}

func TestFinishRetriesAnAcceptedPushWithLostAcknowledgement(t *testing.T) {
	t.Parallel()
	w := completedGuidedWork(t)
	w.withGit("if [ \"$3\" = push ]; then \"$real_git\" \"$@\" || exit $?; exit 9; fi", "finish", "--", "true").fails(2).errHas("unconfirmed")
	head := w.gitOut(nil, "rev-parse", "HEAD")
	w.run("finish", "--", "true").ok().has("published: true")
	if head != w.gitOut(nil, "rev-parse", "HEAD") {
		t.Fatal("retry repeated submission")
	}
}

func TestFinishStopsForUnknownRemoteAndChangedPublicationCandidate(t *testing.T) {
	t.Parallel()
	w := completedGuidedWork(t)
	w.withGit("if [ \"$3\" = ls-remote ]; then exit 9; fi", "finish", "--", "true").fails(2).errHas("online audit is incomplete")
	if !strings.Contains(w.read("docs/features/0007-widget.md"), "status: Draft") {
		t.Fatal("unknown audit submitted")
	}
	w.withGit("if [ \"$3\" = push ]; then \"$real_git\" -C \"$2\" commit --allow-empty -m concurrent >/dev/null || exit 9; fi", "finish", "--", "true").fails(2).errHas("candidate changed")
}

func TestFinishRejectsCommittedChangesFromASubmissionHook(t *testing.T) {
	t.Parallel()
	w := completedGuidedWork(t)
	common := w.gitOut(nil, "rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(w.Root, common)
	}
	writeExecutable(t, filepath.Join(common, "hooks", "post-commit"), `#!/bin/sh
case "$(git log -1 --format=%s)" in
 'Submit 0007')
  echo generated > generated.txt
  git add generated.txt
  git -c core.hooksPath=/dev/null commit -m 'Generated after submission'
  ;;
esac
`)
	w.run("finish", "--", "true").fails(2).errHas("submission changed the validated candidate")
	remote := w.gitOut(nil, "show", "origin/work/0007:docs/features/0007-widget.md")
	if !strings.Contains(remote, "status: Draft") {
		t.Fatal("unvalidated hook changes were published")
	}
}

func TestFinishRejectsAHookThatAmendsTheSubmissionStatus(t *testing.T) {
	t.Parallel()
	w := completedGuidedWork(t)
	common := w.gitOut(nil, "rev-parse", "--git-common-dir")
	if !filepath.IsAbs(common) {
		common = filepath.Join(w.Root, common)
	}
	writeExecutable(t, filepath.Join(common, "hooks", "post-commit"), `#!/bin/sh
case "$(git log -1 --format=%s)" in
 'Submit 0007')
  sed 's/status: Submitted/status: Dropped/' docs/features/0007-widget.md > altered
  mv altered docs/features/0007-widget.md
  git add docs/features/0007-widget.md
  git -c core.hooksPath=/dev/null commit --amend --no-edit
  ;;
esac
`)
	w.run("finish", "--", "true").fails(2).errHas("submission changed the validated candidate")
}
