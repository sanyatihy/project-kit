package main

import (
	"strings"
	"testing"
)

func TestStartRecordsIntentAndRetriesWithoutAnotherCommit(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"plan", "work"} {
		t.Run(mode, func(t *testing.T) {
			p := setUp(t)
			p.commitAll("setup")
			p.git("branch", "-m", "main")
			p.run("start", "--"+mode, "Requested outcome").ok().has("checkout:", "human review")
			head := p.gitOut(nil, "rev-parse", "HEAD")
			if msg := p.gitOut(nil, "show", "-s", "--format=%B"); !strings.Contains(msg, "Kit-Intent:") {
				t.Fatal("intent not recorded")
			}
			p.write("draft.txt", "keep in progress\n")
			p.run("start", "--"+mode, "Requested outcome").ok()
			if head != p.gitOut(nil, "rev-parse", "HEAD") {
				t.Fatal("retry duplicated start")
			}
		})
	}
}

func TestStartGuardsExistingWorkAndReusesClaim(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.run("start", "--work", "A widget").fails(2).errHas("already has a plan")
	a.write("dirty.txt", "valuable\n")
	a.run("start", "--plan", "later").fails(2).errHas("uncommitted")
	a.git("add", "dirty.txt")
	a.git("commit", "-m", "new trunk")
	a.git("push", "origin", "main")
	a.run("start", "0007").ok().has("work/0007")
	w := a.claimWorktree("work/0007")
	w.run("start", "--work", "another change").fails(2).errHas("already has work")
}
