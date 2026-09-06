package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuidedPlanningPublishesAnEmptyBaseButCannotMergeThroughRestrictedRemote(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	gitIn(t, p.Root, "init", "--bare", "-b", "main", origin)
	// A server-side rule allows the initial empty base, then reserves trunk updates
	// for an external reviewer. This fixture does not certify hosted permissions.
	writeExecutable(t, filepath.Join(origin, "hooks", "pre-receive"), `#!/bin/sh
while read old new ref; do
 if [ "$ref" = refs/heads/main ]; then
  if [ "$old" != 0000000000000000000000000000000000000000 ]; then
   echo 'human review required for main' >&2; exit 1
  fi
  if [ -n "$(git ls-tree -r --name-only "$new")" ]; then
   echo 'first main must be empty' >&2; exit 1
  fi
 fi
done
`)
	p.git("remote", "add", "origin", origin)
	p.run("start", "--plan", "first round").ok()
	feature(p, "0001", "first", "First outcome", "Draft", "- [ ] Build it")
	p.run("finish").ok().has("published: true", "Human: approve and merge")
	cmd := exec.Command("git", "push", "origin", "HEAD:refs/heads/main")
	cmd.Dir, cmd.Env = p.Root, gitEnv()
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "human review required") {
		t.Fatalf("agent push could merge: %v\n%s", err, out)
	}
	p.run("start", "0001").fails(2)
	if files := p.gitOut(nil, "ls-tree", "-r", "--name-only", "main"); files != "" {
		t.Fatal("agent put setup onto initial trunk")
	}
}

func TestGuidedRoutesReduceManualGitWhileKeepingReviewStops(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"draft", "approved", "direct"} {
		t.Run(scenario, func(t *testing.T) {
			counts := map[string]int{}
			for _, guided := range []bool{false, true} {
				mode := "legacy"
				if guided {
					mode = "guided"
				}
				a, _ := twoAgents(t)
				manual := func(p *project, args ...string) { counts[mode]++; p.git(args...) }
				commit := func(p *project, message string) { manual(p, "add", "-A"); manual(p, "commit", "-m", message) }
				p := a
				switch scenario {
				case "draft":
					if guided {
						a.run("start", "--plan", "next round").ok()
					} else {
						manual(a, "switch", "-c", "plan/next-round")
					}
					a.run("new", "Next outcome").ok()
					feature(a, "0008", "next-outcome", "Next outcome", "Draft", "- [ ] Implement it")
					if guided {
						a.run("finish").ok()
					} else {
						commit(a, "plan")
						a.run("audit", "--strict").ok()
						manual(a, "push", "origin", "HEAD")
					}
					if !hasRemoteBranch(t, a, "plan/next-round") {
						t.Fatal("planning review candidate not published")
					}
					a.run("claim", "0008").fails(2)
				case "approved":
					command := "claim"
					if guided {
						command = "start"
					}
					a.run(command, "0007").ok()
					p = a.claimWorktree("work/0007")
					feature(p, "0007", "widget", "A widget", "Draft", "- [x] Build it")
					commit(p, "0007 built")
					if guided {
						p.run("finish", "--", "true").ok()
					} else {
						if err := exec.Command("true").Run(); err != nil {
							t.Fatal(err)
						}
						p.run("audit", "--strict").ok()
						p.run("submit", "0007").ok()
						manual(p, "push", "origin", "HEAD")
					}
					if !hasRemoteBranch(t, p, "work/0007") {
						t.Fatal("implementation candidate not published")
					}
				case "direct":
					if guided {
						a.run("start", "--work", "Small change").ok()
					} else {
						manual(a, "switch", "-c", "spike/small-change")
					}
					a.write("small.txt", "requested change\n")
					if guided {
						a.run("finish", "--", "true").fails(2).errHas("fill review facts")
					} else {
						a.run("harvest", "Small change", "--box", "Small change").ok()
					}
					feature(a, "0008", "small-change", "Small change", "Draft", "- [x] Small change")
					commit(a, "review facts")
					if guided {
						a.run("finish", "--", "true").ok()
					} else {
						if err := exec.Command("true").Run(); err != nil {
							t.Fatal(err)
						}
						a.run("audit", "--strict").ok()
						a.run("submit", "0008").ok()
						manual(a, "push", "origin", "HEAD")
					}
					if !hasRemoteBranch(t, a, "0008-small-change") {
						t.Fatal("direct-work review candidate not published")
					}
				}
				// Every route stops before human integration. The remote trunk still has no new outcome.
				main := p.gitOut(nil, "show", "origin/main:docs/features/0007-widget.md")
				if !strings.Contains(main, "status: Draft") {
					t.Fatal("route merged its own implementation")
				}
			}
			t.Logf("%s manual agent git calls: legacy=%d guided=%d (fixture setup and human review excluded)", scenario, counts["legacy"], counts["guided"])
			if counts["guided"] >= counts["legacy"] {
				t.Fatal("guided route did not reduce manual git")
			}
		})
	}
}
