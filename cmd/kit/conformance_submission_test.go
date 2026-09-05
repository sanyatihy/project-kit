package main

import (
	"strings"
	"testing"
	"time"
)

func TestSubmissionRequiresItsFeatureBranchIncludingLegacySpelling(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"submit", "ship"} {
		t.Run(command, func(t *testing.T) {
			p := setUp(t)
			feature(p, "0007", "widget", "Widget", "Draft", "- [x] Done")
			p.commitAll("completed work")
			before := p.gitOut(nil, "rev-parse", "HEAD")
			p.run(command, "0007").fails(2).errHas("feature branch")
			p.git("checkout", "--detach")
			p.run(command, "0007").fails(2).errHas("feature branch")
			if got := p.gitOut(nil, "rev-parse", "HEAD"); got != before {
				t.Fatal("refusal changed HEAD")
			}
			if dirty := p.gitOut(nil, "status", "--porcelain"); dirty != "" {
				t.Fatal("refusal changed files:", dirty)
			}
			p.git("checkout", "-b", "0007-widget")
			p.run(command, "0007").ok()
			if got := p.read("docs/features/0007-widget.md"); !strings.Contains(got, "status: Submitted") || strings.Contains(got, "shipped:") {
				t.Fatal(got)
			}
		})
	}
}

func TestLegacySnapshotCanBeRevisedBeforeMergeAndIsFrozenAfterMerge(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	feature(p, "0007", "widget", "Widget", "Draft", "- [x] Done")
	p.commitAll("approved plan")
	trunk := p.headBranch()
	p.git("checkout", "-b", "0007-widget")
	path := "docs/features/0007-widget.md"
	legacy := strings.Replace(p.read(path), "status: Draft", "status: Shipped\nshipped: "+time.Now().UTC().Format("2006-01-02"), 1)
	p.write(path, legacy)
	p.commitAll("legacy submission")
	p.run("audit").ok().lacks("no-status", "submitted-no-date")
	p.run("reset", "0007", "review feedback").ok()
	p.run("ship", "0007").ok()
	p.git("checkout", trunk)
	p.git("merge", "0007-widget")
	p.git("checkout", "-b", "followup")
	p.run("reset", "0007", "rewrite history").fails(2).errHas("frozen")
	// Legacy documents on trunk have the same frozen-history contract.
	p.git("checkout", trunk)
	p.write(path, legacy)
	p.commitAll("legacy trunk fixture")
	p.git("checkout", "-b", "legacy-followup")
	p.run("reopen", "0007", "rewrite history").fails(2).errHas("frozen")
	p.write(path, legacy+"\nChanged history.\n")
	p.run("audit", "--strict").fails(1).has("frozen-doc")
}

func TestRepairMustBeExplicitAndCannotReviveDroppedPlans(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	wedged(p, "0007", "widget")
	p.commitAll("invalid status")
	before := p.read("docs/features/0007-widget.md")
	p.run("reopen", "0007", "repair").fails(2).errHas("--repair")
	if p.read("docs/features/0007-widget.md") != before {
		t.Fatal("implicit repair changed the doc")
	}
	p.run("reopen", "0007", "--repair", "repair").ok()
	p.run("drop", "0007", "retire").ok()
	p.run("reopen", "0007", "--repair", "revive").fails(2).errHas("new feature")
}

func TestDiscardPreservesForeignClaimsAndDirtyWork(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("fetch", "origin")
	b.run("release", "0007", "--discard").fails(2).errHas("never created")
	w := a.claimWorktree("0007-widget")
	w.write("valuable.txt", "uncommitted work\n")
	a.run("release", "0007", "--discard").fails(2).errHas("uncommitted work")
	if !hasRemoteBranch(t, a, "0007-widget") || !w.exists("valuable.txt") {
		t.Fatal("disposal bypassed preservation")
	}
}

func TestMissingFeatureDoesNotImplicitlyDeleteAReservation(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.reserve("0009", "unused", 30)
	a.run("drop", "0009", "cleanup").fails(2).errHas("--reservation")
	// The explicit operation succeeding proves the refusal preserved its reservation.
	a.run("drop", "0009", "--reservation", "cleanup").ok()
}

func TestCommandHelpWorksThroughBothEntryPoints(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	for _, command := range append(subcommands(t), "ship", "reset") {
		direct := p.run(command, "--help").ok()
		topic := p.run("help", command).ok()
		if !strings.HasPrefix(direct.Stdout, strings.TrimSpace(topic.Stdout)) || !strings.HasPrefix(topic.Stdout, "kit "+canonicalCommand(command)) {
			t.Errorf("help differs for %s", command)
		}
	}
}
