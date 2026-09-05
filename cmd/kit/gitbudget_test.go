package main

// What a board read costs in git processes, with a ceiling on it.
//
// The regression this exists to catch has happened three times: an answer that cannot
// change inside one command, asked once per feature. It is invisible until a board read
// takes minutes, because nothing about the output changes — only how long it takes and
// how many processes it spawns.
//
// It was a script before, `make gitcount`, and a number nobody was required to look at.
// Which is how the third one landed: the script measured `next --offline`, the fan-out
// had moved to the checks that need the remote, and the offline count was identical
// either way. So the ceiling lives here now, where the gate fails on it, and both paths
// are measured. Same argument as the text budgets in budget_test.go: a call is always
// easier to add than to justify, and no reviewer counts subprocesses.
//
// Raising a number here is allowed and says so in the message — a genuinely new question
// costs a process, and that is not a regression. Raise it and say why. What must not
// happen is a number growing with the size of the board.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const budgetClaims = 10

func TestABoardReadKeepsGitWorkBoundedAsFeaturesGrow(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"next", "audit", "no trunk"} {
		t.Run(mode, func(t *testing.T) {
			counts := []int{}
			for _, size := range []int{30, 130} {
				var p *project
				args := []string{"next", "--offline"}
				if mode == "no trunk" {
					p = noTrunkFixture(t, size)
				} else {
					p = budgetFixture(t, size)
				}
				if mode == "audit" {
					shipEverything(t, p, size)
					args = []string{"audit"}
				}
				n, breakdown := countGit(t, p, args...)
				t.Logf("%s, %d features: %d git calls\n%s", mode, size, n, breakdown)
				counts = append(counts, n)
			}
			if counts[1] > counts[0] {
				t.Fatalf("100 unclaimed features added %d git calls; batch repository reads", counts[1]-counts[0])
			}
		})
	}
}

// budgetFixture is a project with a board to walk, claim branches to resolve, and a bare
// remote on disk to ask.
func budgetFixture(t *testing.T, size int) *project {
	t.Helper()
	base, origin := budgetRemote(t, "main")
	p := budgetBoard(t, base, size)
	p.git("remote", "add", "origin", origin)
	p.commitAll("a board")
	p.git("push", "-q", "origin", "HEAD:refs/heads/main")
	p.git("fetch", "-q", "origin")
	for i := 1; i <= budgetClaims; i++ {
		p.git("branch", fmt.Sprintf("%04d-feature-%d", i, i))
	}
	return p
}

// noTrunkFixture is the same board in the state where nothing resolves a trunk: a remote
// pushed to rather than cloned from, so nothing wrote `origin/HEAD`; a default branch
// called neither `main` nor `master`; and a checkout standing on a claim, which the last
// rung of the ladder refuses to call a trunk. Every claim worktree of a project that
// started life locally is in it.
func noTrunkFixture(t *testing.T, size int) *project {
	t.Helper()
	base, origin := budgetRemote(t, "develop")
	p := budgetBoard(t, base, size)
	p.git("branch", "-m", "develop")
	p.git("remote", "add", "origin", origin)
	p.commitAll("a board")
	p.git("push", "-q", "origin", "develop")
	for i := 1; i <= budgetClaims; i++ {
		p.git("branch", fmt.Sprintf("%04d-feature-%d", i, i))
	}
	p.git("checkout", "-q", "0001-feature-1")
	return p
}

// budgetRemote is a bare remote on disk, and the directory to put a project beside it.
func budgetRemote(t *testing.T, trunk string) (base, origin string) {
	t.Helper()
	base = t.TempDir()
	if resolved, err := filepath.EvalSymlinks(base); err == nil {
		base = resolved
	}
	origin = filepath.Join(base, "origin.git")
	gitIn(t, base, "init", "-q", "--bare", "-b", trunk, origin)
	return base, origin
}

// budgetBoard writes the features, uncommitted: what a caller does with git afterwards is
// what the two fixtures differ in.
func budgetBoard(t *testing.T, base string, size int) *project {
	t.Helper()
	p := projectAt(t, filepath.Join(base, "project"))
	p.run("init").ok()
	writeVision(p)
	for i := 1; i <= size; i++ {
		id := fmt.Sprintf("%04d", i)
		feature(p, id, fmt.Sprintf("feature-%d", i), fmt.Sprintf("Feature %d", i),
			"Draft", "- [ ] One\n- [ ] Two")
	}
	return p
}

// shipEverything puts the board in the state the claim checks work hardest in: every doc
// Submitted, so every one of them asks where its work is.
func shipEverything(t *testing.T, p *project, size int) {
	t.Helper()
	for i := 1; i <= size; i++ {
		rel := fmt.Sprintf("docs/features/%04d-feature-%d.md", i, i)
		doc := strings.NewReplacer(
			"status: Draft", "status: Submitted",
			"- [ ] One", "- [x] One",
			"- [ ] Two", "- [x] Two",
		).Replace(p.read(rel))
		p.write(rel, doc)
	}
	p.commitAll("every feature submitted")
	p.git("push", "-q", "origin", "HEAD:refs/heads/main")
	p.git("fetch", "-q", "origin")
}

// countGit runs the binary with a counting wrapper ahead of git on PATH and reports how
// many processes it spawned, with the commands behind the number for whoever has to fix
// it. The wrapper names the real git by absolute path: it is itself called `git`, and
// anything relative would find the wrapper again.
func countGit(t *testing.T, p *project, args ...string) (int, string) {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("finding git: %v", err)
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "git.log")
	shim := filepath.Join(dir, "bin")
	if err := os.MkdirAll(shim, 0o755); err != nil {
		t.Fatalf("preparing the wrapper: %v", err)
	}
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %q\nexec %q \"$@\"\n", log, realGit)
	if err := os.WriteFile(filepath.Join(shim, "git"), []byte(script), 0o755); err != nil {
		t.Fatalf("writing the wrapper: %v", err)
	}

	cmd := exec.Command(kitBinary(t), args...)
	cmd.Dir = p.Root
	cmd.Env = append(gitEnv(), "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("kit %s: %v\n%s", strings.Join(args, " "), err, out)
	}

	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("reading the count: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	counts := map[string]int{}
	for _, line := range lines {
		if fields := strings.Fields(line); len(fields) > 2 {
			counts[fields[2]]++ // past the `-C <root>` every call carries
		}
	}
	if counts["ls-remote"] > 1 || counts["fetch"] > 1 {
		t.Fatalf("single-remote survey repeated network reads: %v", counts)
	}
	var breakdown strings.Builder
	for verb, n := range counts {
		fmt.Fprintf(&breakdown, "  %3d %s\n", n, verb)
	}
	return len(lines), breakdown.String()
}
