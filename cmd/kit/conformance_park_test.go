package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parkingPlan(t *testing.T) *project {
	t.Helper()
	p := setUp(t)
	feature(p, "0001", "later", "Later", "Draft", "- [ ] Build it\n- [ ] Verify it")
	p.commitAll("plan")
	p.git("branch", "-m", "main")
	p.git("switch", "-c", "plan/schedule")
	return p
}

func TestParkingPreservesPlanAndCommitsOnlyItsTarget(t *testing.T) {
	t.Parallel()
	p := parkingPlan(t)
	path := "docs/features/0001-later.md"
	before := p.read(path)
	p.write("unrelated.txt", "keep staged\n")
	p.git("add", "unrelated.txt")
	p.run("park", "0001", "operator prefers later").ok().has("Park 0001", "human review")
	parked := p.read(path)
	prefix := strings.Replace(before, "status: Draft", "status: Parked", 1)
	if !strings.HasPrefix(parked, prefix) || !strings.Contains(parked, "parked: operator prefers later") {
		t.Fatalf("lost plan content or reason:\n%s", parked)
	}
	if got := p.gitOut(nil, "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD"); got != path {
		t.Fatalf("committed other files: %q", got)
	}
	if got := p.gitOut(nil, "status", "--porcelain"); got != "A  unrelated.txt" {
		t.Fatalf("changed staging: %q", got)
	}
	head := p.gitOut(nil, "rev-parse", "HEAD")
	p.run("park", "0001", "operator prefers later").ok().has("already Parked")
	if got := p.gitOut(nil, "rev-parse", "HEAD"); got != head {
		t.Fatal("retry created another commit")
	}
	p.run("reopen", "0001", "--repair", "try repair").fails(2).errHas("kit activate")
	p.run("activate", "0001").ok().has("Activate 0001")
	if !strings.HasPrefix(p.read(path), before) {
		t.Fatal("activation lost original plan")
	}
	head = p.gitOut(nil, "rev-parse", "HEAD")
	p.run("activate", "0001").ok().has("already Draft")
	if got := p.gitOut(nil, "rev-parse", "HEAD"); got != head {
		t.Fatal("activation retry created another commit")
	}
}

func TestParkingRefusalsPreserveFilesAndHistory(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, want string
		prepare    func(*project)
	}{
		{"trunk", "plan/<round>", func(p *project) { p.git("switch", "main") }},
		{"claim branch", "plan/<round>", func(p *project) { p.git("switch", "-c", "plan/0001") }},
		{"dirty target", "uncommitted", func(p *project) {
			p.write("docs/features/0001-later.md", p.read("docs/features/0001-later.md")+"\nextra\n")
		}},
		{"incomplete", "complete plan", func(p *project) { unwritten(p, "0001", "later"); p.commitAll("incomplete") }},
		{"started", "unstarted", func(p *project) {
			feature(p, "0001", "later", "Later", "Draft", "- [x] Started\n- [ ] More")
			p.commitAll("started")
		}},
		{"submitted", "requires Draft", func(p *project) {
			feature(p, "0001", "later", "Later", "Submitted", "- [x] Done")
			p.commitAll("submitted")
		}},
		{"dropped", "requires Draft", func(p *project) {
			feature(p, "0001", "later", "Later", "Dropped", "- [ ] Done")
			p.commitAll("dropped")
		}},
		{"duplicate id", "declare id", func(p *project) {
			feature(p, "0001", "duplicate", "Other", "Draft", "- [ ] Work")
			p.commitAll("duplicate")
		}},
		{"duplicate field", "twice", func(p *project) {
			path := "docs/features/0001-later.md"
			p.write(path, strings.Replace(p.read(path), "status: Draft", "status: Draft\nstatus: Draft", 1))
			p.commitAll("duplicate")
		}},
		{"claim", "claimed", func(p *project) { p.git("branch", "work/0001") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := parkingPlan(t)
			tc.prepare(p)
			before := p.read("docs/features/0001-later.md")
			head := p.gitOut(nil, "rev-parse", "HEAD")
			p.run("park", "0001", "later").fails(2).errHas(tc.want)
			if p.read("docs/features/0001-later.md") != before || p.gitOut(nil, "rev-parse", "HEAD") != head {
				t.Fatal("refusal changed plan or history")
			}
		})
	}
}

func TestParkingRequiresKnownCurrentRemoteAndRefusesRemoteClaims(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("switch", "-c", "plan/later")
	b.run("park", "0007", "later").fails(2).errHas("claimed")
	b.git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	b.run("park", "0007", "later").fails(2).errHas("remote state is unknown")
}

func TestParkingCommitFailurePreservesDecisionForRecovery(t *testing.T) {
	t.Parallel()
	p := parkingPlan(t)
	hook := filepath.Join(p.Root, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	p.run("park", "0001", "later").fails(2).errHas("commit failed", "review and commit only")
	if !strings.Contains(p.read("docs/features/0001-later.md"), "status: Parked") {
		t.Fatal("lost failed status edit")
	}
	p.run("park", "0001", "later").fails(2).errHas("uncommitted")
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	p.git("commit", "-m", "Recover parking", "--", "docs/features/0001-later.md")
	p.run("park", "0001", "later").ok().has("already Parked")
}

func TestParkedAvailabilityChangesOnlyAfterReviewMerge(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.git("switch", "-c", "plan/parking")
	a.run("park", "0007", "operator chose later").ok()
	if plan := b.plan("next", "--json"); len(plan.Ready) != 1 {
		t.Fatal("unmerged parking changed shared availability")
	}
	a.git("switch", "main")
	a.git("merge", "--ff-only", "plan/parking")
	a.git("push", "origin", "main")
	// The stale clone has Draft in its files, even after fetching the new trunk.
	b.run("claim", "0007").fails(2)
	b.git("fetch", "origin")
	if plan := b.plan("next", "--json"); len(plan.Ready) != 0 {
		t.Fatal("stale clone offered parked plan")
	}
	b.run("claim", "0007").fails(2)
	b.git("pull", "--ff-only")
	plan := b.plan("next", "--json")
	if len(plan.Parked) != 1 || len(plan.Ready) != 0 || plan.Parked[0].Reason != "operator chose later" || plan.Parked[0].Task != "" || plan.Parked[0].Branch != "" {
		t.Fatalf("bad parked board: %+v", plan)
	}
	b.run("next").ok().has("parked", "operator chose later").lacks("kit claim 0007", "kit new")
	b.run("claim", "0007").fails(1).errHas("Parked")
	b.git("switch", "-c", "plan/activation")
	b.run("activate", "0007").ok()
	plan = b.plan("next", "--json")
	if len(plan.Ready) != 0 || len(plan.Blocked) != 1 || !strings.Contains(plan.Blocked[0].WaitingOn, "activation") {
		t.Fatalf("unreviewed activation offered work: %+v", plan)
	}
	b.run("claim", "0007").fails(2)
	if plan := a.plan("next", "--json"); len(plan.Parked) != 1 {
		t.Fatal("unmerged activation changed shared availability")
	}
	b.git("switch", "main")
	b.git("merge", "--ff-only", "plan/activation")
	b.git("push", "origin", "main")
	a.git("pull", "--ff-only", "origin", "main")
	if plan := a.plan("next", "--json"); len(plan.Ready) != 1 || len(plan.Parked) != 0 {
		t.Fatal("reviewed activation did not restore availability")
	}
	a.run("claim", "0007").ok()
}

func TestParkedDependenciesRemainUnmetAndDocumentsRemainValidated(t *testing.T) {
	t.Parallel()
	p := parkingPlan(t)
	p.write("docs/features/0002-child.md", doc("0002", "Child", "Draft", "needs: 0001", "- [ ] Use it"))
	p.commitAll("dependent plan")
	p.run("park", "0001", "after next quarter").ok()
	p.git("switch", "main")
	p.git("merge", "--ff-only", "plan/schedule")
	plan := p.plan("next", "--json")
	if len(plan.Parked) != 1 || len(plan.Blocked) != 1 || len(plan.Blocked[0].Needs) != 1 {
		t.Fatalf("parked dependency offered work: %+v", plan)
	}
	p.run("claim", "0002").fails(1).errHas("0001")
	// Scheduling itself creates no age-based work or question finding.
	p.run("audit", "--strict").ok().lacks("stalled", "inbox-stale", "no-status")
	path := "docs/features/0001-later.md"
	p.write(path, strings.Replace(p.read(path), "This way.", "", 1))
	p.commitAll("break design")
	p.run("audit", "--strict").fails(1).has("unfilled-template")
	p.git("switch", "-c", "plan/revalidate")
	p.run("activate", "0001").fails(2).errHas("complete plan")
}

func TestAuditReportsUncommittedParkingAndActivation(t *testing.T) {
	t.Parallel()
	p := parkingPlan(t)
	path := "docs/features/0001-later.md"
	p.write(path, strings.Replace(p.read(path), "status: Draft", "status: Parked", 1))
	p.run("audit", "--strict").fails(1).has("status-uncommitted")
	p.commitAll("park")
	p.write(path, strings.Replace(p.read(path), "status: Parked", "status: Draft", 1))
	p.run("audit", "--strict").fails(1).has("status-uncommitted")
}

func TestParkingRefusesStaleTrunkAndTerminalHistory(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.write("new.txt", "new trunk work\n")
	a.commitAll("advance trunk")
	a.git("push", "origin", "main")
	b.git("switch", "-c", "plan/stale")
	b.run("park", "0007", "later").fails(2).errHas("integrate the current trunk")
	b.git("fetch", "origin")
	b.git("merge", "--ff-only", "origin/main")
	b.run("park", "0007", "later").ok()
	// A manually rewritten local copy cannot retire already integrated history.
	a.run("drop", "0007", "retired").ok()
	a.git("push", "origin", "main")
	b.git("fetch", "origin")
	b.git("reset", "--hard", "origin/main")
	path := "docs/features/0007-widget.md"
	b.write(path, strings.Replace(b.read(path), "status: Dropped", "status: Draft", 1))
	b.commitAll("rewrite local status")
	b.run("park", "0007", "later").fails(2).errHas("terminal history")
}

func TestParkingReviewReportsAClaimAcquiredAfterTheDecision(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.git("switch", "-c", "plan/parking")
	a.run("park", "0007", "later").ok()
	b.run("claim", "0007").ok()
	a.run("audit", "--strict").fails(1).has("parked-claim", "work/0007")
	plan := a.plan("next", "--json")
	if len(plan.Parked) != 1 || plan.Parked[0].ClaimedBy != "work/0007" || plan.Parked[0].Task != "" {
		t.Fatalf("parking hid ownership: %+v", plan)
	}
	a.run("next").ok().has("claim observed", "work/0007").lacks("kit claim 0007")
	a.run("activate", "0007").fails(2).errHas("claimed")
}

func TestParkedBoardDoesNotGuessIntegrationWithAnUnansweredRemote(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.git("switch", "-c", "plan/parking")
	a.run("park", "0007", "later").ok()
	a.git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	plan := a.plan("next", "--json")
	if !plan.RemoteUnanswered || len(plan.Parked) != 1 || !strings.Contains(plan.Parked[0].WaitingOn, "unknown") {
		t.Fatalf("guessed integration: %+v", plan)
	}
}
