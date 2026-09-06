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
