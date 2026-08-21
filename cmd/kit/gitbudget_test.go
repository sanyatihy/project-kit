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

const (
	// The fixture: big enough that per-feature growth is obvious against fixed costs,
	// small enough to build in a moment.
	budgetFeatures = 30
	budgetClaims   = 10

	// `kit next --offline`, where nothing may be asked of the remote. 116 before 0035
	// memoised the branch and the remotes, then 28. It is 29 here for a fixture reason
	// rather than a regression: this project has a remote, so resolving the trunk asks
	// `symbolic-ref refs/remotes/origin/HEAD`, and the harness this replaced measured a
	// project with no remote at all. One fixture serves both runs, which is worth a
	// process.
	offlineBudget = 29

	// `kit audit` against a local bare remote. The remote is on disk, so the round trip
	// is one cheap process and what grows with the board is the fan-out. 67 before 0039
	// read the trunk's statuses in one pass, 10 after.
	//
	// 11 since 0048: the plan checks gate on whether a doc is committed, which is a new
	// question and so costs a process — one `ls-tree` for the whole tree. It was 40 for a
	// day, one `cat-file` per feature, which is exactly the shape this ceiling exists to
	// catch, and it caught it.
	//
	// 12 since 0054: how far the newest tag is behind the trunk is a new question, and one
	// `describe --long` answers all of it. Fixed, not per feature — the distance is a fact
	// about the project and is asked once.
	onlineBudget = 12

	// The same board where no trunk resolves. `Trunk()` answered "" there, and "" was
	// also its "not asked yet" sentinel, so `findTrunk` ran the whole ladder again for
	// every caller: 747 processes on this fixture, 721 of them one `symbolic-ref`, and
	// growing with the board. The fixture above always resolves a trunk, which is the
	// only reason the ceiling never caught it. One `show` under the offline number
	// because this checkout stands on a claim, and its own boxes are read from the tree.
	noTrunkBudget = 28
)

func TestABoardReadStaysWithinItsGitBudget(t *testing.T) {
	t.Parallel()
	p := budgetFixture(t)

	offline, breakdown := countGit(t, p, "next", "--offline")
	// Logged whether or not it fails: `make gitcount` is how the numbers are read, and a
	// count that has quietly halved is worth knowing about too.
	t.Logf("%d features, %d claims: %d git processes offline (ceiling %d)\n%s",
		budgetFeatures, budgetClaims, offline, offlineBudget, breakdown)
	if offline > offlineBudget {
		t.Errorf("`kit next --offline` spawned %d git processes, over its %d ceiling.\n"+
			"An answer that cannot change inside one command is asked once. If this is a "+
			"genuinely new question rather than a repeat, raise the ceiling and say why.\n%s",
			offline, offlineBudget, breakdown)
	}

	shipEverything(t, p)

	online, breakdown := countGit(t, p, "audit")
	t.Logf("%d features, %d claims: %d git processes online (ceiling %d)\n%s",
		budgetFeatures, budgetClaims, online, onlineBudget, breakdown)
	if online > onlineBudget {
		t.Errorf("`kit audit` spawned %d git processes, over its %d ceiling.\n"+
			"The checks that need the remote are the half a `--offline` measurement cannot "+
			"see. If this is a new question rather than a repeat, raise the ceiling and say "+
			"why.\n%s", online, onlineBudget, breakdown)
	}
}

func TestABoardWithNoTrunkStaysWithinItsGitBudget(t *testing.T) {
	t.Parallel()
	p := noTrunkFixture(t)

	n, breakdown := countGit(t, p, "next", "--offline")
	t.Logf("%d features, %d claims, no trunk: %d git processes (ceiling %d)\n%s",
		budgetFeatures, budgetClaims, n, noTrunkBudget, breakdown)
	if n > noTrunkBudget {
		t.Errorf("`kit next --offline` spawned %d git processes on a board with no trunk, "+
			"over its %d ceiling.\nAn answer that cannot change inside one command is asked "+
			"once, and that includes the answer \"there is none\".\n%s",
			n, noTrunkBudget, breakdown)
	}
}

// budgetFixture is a project with a board to walk, claim branches to resolve, and a bare
// remote on disk to ask.
func budgetFixture(t *testing.T) *project {
	t.Helper()
	base, origin := budgetRemote(t, "main")
	p := budgetBoard(t, base)
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
func noTrunkFixture(t *testing.T) *project {
	t.Helper()
	base, origin := budgetRemote(t, "develop")
	p := budgetBoard(t, base)
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
func budgetBoard(t *testing.T, base string) *project {
	t.Helper()
	p := projectAt(t, filepath.Join(base, "project"))
	p.run("init").ok()
	writeVision(p)
	for i := 1; i <= budgetFeatures; i++ {
		id := fmt.Sprintf("%04d", i)
		feature(p, id, fmt.Sprintf("feature-%d", i), fmt.Sprintf("Feature %d", i),
			"Draft", "- [ ] One\n- [ ] Two")
	}
	return p
}

// shipEverything puts the board in the state the claim checks work hardest in: every doc
// Shipped, so every one of them asks where its work is.
func shipEverything(t *testing.T, p *project) {
	t.Helper()
	for i := 1; i <= budgetFeatures; i++ {
		rel := fmt.Sprintf("docs/features/%04d-feature-%d.md", i, i)
		doc := strings.NewReplacer(
			"status: Draft", "status: Shipped",
			"- [ ] One", "- [x] One",
			"- [ ] Two", "- [x] Two",
		).Replace(p.read(rel))
		p.write(rel, doc)
	}
	p.commitAll("every feature shipped")
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
	var breakdown strings.Builder
	for verb, n := range counts {
		fmt.Fprintf(&breakdown, "  %3d %s\n", n, verb)
	}
	return len(lines), breakdown.String()
}
