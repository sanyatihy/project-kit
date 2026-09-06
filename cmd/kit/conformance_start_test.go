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

func TestStartPublishesOnlyAnEmptyBaseAndPreservesTheIndex(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	origin := p.path("../remote.git")
	gitIn(t, p.Root, "init", "--bare", origin)
	p.git("remote", "add", "origin", origin)
	p.write("product.txt", "staged\n")
	p.git("add", "product.txt")
	p.write("product.txt", "unstaged\n")
	p.run("start", "--work", "Build requested product").ok()
	if tree := p.gitOut(nil, "ls-tree", "-r", "--name-only", "main"); tree != "" {
		t.Fatalf("product leaked into main: %s", tree)
	}
	if p.read("product.txt") != "unstaged\n" || p.gitOut(nil, "show", ":product.txt") != "staged" {
		t.Fatal("start changed staged or unstaged content")
	}
	if !hasRemoteBranch(t, p, "main") {
		t.Fatal("empty base was not published")
	}
	head := p.gitOut(nil, "rev-parse", "HEAD")
	p.run("start", "--work", "Build requested product").ok()
	if head != p.gitOut(nil, "rev-parse", "HEAD") {
		t.Fatal("retry repeated setup")
	}
}

func TestStartResumesAfterEmptyBaseAndRefusesForeignRemoteHistory(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.git("commit", "--allow-empty", "-m", "Initialize review base")
	p.git("branch", "-m", "main")
	p.write("keep.txt", "uncommitted\n")
	p.run("start", "--plan", "first round").ok()
	if p.read("keep.txt") != "uncommitted\n" {
		t.Fatal("recovery lost work")
	}
	a, _ := twoAgents(t)
	q := setUp(t)
	q.git("remote", "add", "origin", a.gitOut(nil, "remote", "get-url", "origin"))
	q.run("start", "--plan", "fresh").fails(2).errHas("already has history")
	if q.resolves("HEAD") {
		t.Fatal("refusal created history")
	}
}
