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
