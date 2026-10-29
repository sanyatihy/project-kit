package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A repo is untrusted input: anything a doc, a branch or a remote says may reach a
// printed command, a git argument or a file write.

func TestAnIDThatIsNotDigitsIsNeverPrintedAsACommand(t *testing.T) {
	t.Parallel()
	p := cleanProject(t)
	const payload = "0002$(touch${IFS}pwned)"
	p.write("docs/features/evil.md", doc(payload, "Evil", "Draft", "", "- [ ] Do it"))
	p.commitAll("a hostile id")

	p.run("next", "--offline").ok().lacks("$(touch")
	p.run("next", "--offline", "--json").ok().lacks("$(touch")
	p.run("audit", "--offline").ok().has("id-mismatch", "not a feature id")
	p.run("audit", "--offline", "--strict").fails(1)
}

func TestAnIDPastFourDigitsIsClaimedAndCited(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	feature(a, "12345", "big", "Big", "Draft", "- [ ] Build it")
	a.write("docs/features/12346-after.md",
		doc("12346", "After", "Draft", "needs: 12345", "- [ ] Build it"))
	a.commitAll("ids past 9999")
	a.git("push", "-q", "origin", "HEAD:refs/heads/main")

	if got := boardGroup(a.plan("next", "--json"), "12346"); got != "blocked" {
		t.Errorf("12346 is %s; needs: 12345 did not parse as a dependency", got)
	}
	a.run("start", "12345").ok().has("work/12345")
	w := a.claimWorktree("work/12345")
	w.run("next").ok().has("12345")
}

func TestATitleIsOneLineAndATerminalNeverSeesItsEscapes(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.run("new", "Evil\nstatus: Submitted\nid: 0042").fails(2).errHas("one line")
	p.run("harvest", "Evil\nid: 0042", "--box", "x").fails(2).errHas("one line")

	p.write("docs/features/0002-red.md", doc("0002", "Red \x1b[31malert", "Draft", "", "- [ ] Do it"))
	p.commitAll("an escape in a title")
	p.run("next", "--offline").ok().has("Red [31malert").lacks("\x1b")
	p.run("audit", "--offline").ok().lacks("\x1b")
}

func TestABranchNamedLikeAnOptionNeverReachesGitAsOne(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.write("Makefile", "check:\n\ttrue\n")
	a.commitAll("a tracked file")
	a.git("push", "-q", "origin", "HEAD:refs/heads/main")
	// The remote names its default branch as an option. A clone checks it out and points
	// origin/HEAD at it, so both the trunk and the checkout's branch carry the name.
	a.git("push", "-q", "origin", "HEAD:refs/heads/--output=Makefile")
	origin := a.gitOut(nil, "remote", "get-url", "origin")
	gitIn(t, origin, "symbolic-ref", "HEAD", "refs/heads/--output=Makefile")
	c := projectAt(t, t.TempDir()+"/c")
	gitIn(t, c.Root+"/..", "clone", "-q", origin, c.Root)

	c.run("audit", "--offline").ok()
	c.run("next", "--offline").ok()
	if got := c.read("Makefile"); got != "check:\n\ttrue\n" {
		t.Fatalf("a branch name rewrote a tracked file: %q", got)
	}
}

func TestKitWritesNoFileThroughASymlink(t *testing.T) {
	t.Parallel()
	outside := t.TempDir() + "/rc"
	p := cleanProject(t)
	if err := os.WriteFile(outside, []byte("untouched\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// AGENTS.md committed as a link: init must not append the protocol to its target.
	agents := filepath.Join(p.Root, "AGENTS.md")
	if err := os.Remove(agents); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, agents); err != nil {
		t.Fatal(err)
	}
	p.run("init").fails(2).errHas("symlink")

	// A feature doc committed as a link: a status write must not land in its target.
	q := cleanProject(t)
	feature(q, "0002", "real", "Real", "Draft", "- [ ] Do it")
	if err := os.Symlink(outside, filepath.Join(q.Root, "docs/features/0003-link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte(doc("0003", "Link", "Draft", "", "- [ ] Do it")), 0o644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(outside)
	q.commitAll("a linked doc")
	q.run("drop", "0003", "not wanted").fails(2).errHas("symlink")
	if after, _ := os.ReadFile(outside); string(after) != string(before) {
		t.Fatalf("drop wrote through the link:\n%s", after)
	}
}

func TestAClaimWorktreeIsNeverMadeThroughALinkedWorktreesDir(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(a.Root, ".worktrees")); err != nil {
		t.Fatal(err)
	}
	a.run("claim", "0007").ok().has("not a directory")
	if _, err := os.Stat(filepath.Join(outside, "work")); err == nil {
		t.Fatal("the claim's worktree was made outside the project")
	}
}
