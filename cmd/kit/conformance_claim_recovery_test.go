package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func (p *project) withGit(script string, args ...string) *result {
	p.t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		p.t.Fatal(err)
	}
	shim := p.t.TempDir()
	writeExecutable(p.t, filepath.Join(shim, "git"), "#!/bin/sh\nreal_git="+shellQuote(realGit)+"\n"+script+"\nexec \"$real_git\" \"$@\"\n")
	cmd := exec.Command(kitBinary(p.t), args...)
	cmd.Dir, cmd.Env = p.Root, gitEnv("PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			p.t.Fatal(err)
		}
	}
	return &result{t: p.t, args: args, Code: code, Stdout: out.String(), Stderr: stderr.String()}
}

func TestClaimRecoversAnAcceptedPushWhoseAcknowledgementWasLost(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	marker := filepath.Join(t.TempDir(), "accepted")
	script := "if [ \"$3\" = push ]; then\n\"$real_git\" \"$@\" >/dev/null 2>&1 || exit $?\ntouch " + shellQuote(marker) + "\nexit 1\nfi\nif [ \"$3\" = ls-remote ] && [ -f " + shellQuote(marker) + " ]; then exit 1; fi\n"
	a.withGit(script, "claim", "0007").fails(2).errHas("preserved")
	receipt := a.gitOut(nil, "rev-parse", "refs/kit/claims/0007")
	if remote := a.gitOut(nil, "ls-remote", "--heads", "origin", "work/0007"); !strings.Contains(remote, receipt) {
		t.Fatal("accepted push was lost:", remote)
	}
	a.run("claim", "0007").ok().has("worktree")
	if got := a.gitOut(nil, "rev-parse", "refs/heads/work/0007"); got != receipt {
		t.Fatal("retry changed the claim")
	}
	a.run("release", "0007").ok()
	if a.resolves("refs/kit/claims/0007") || hasRemoteBranch(t, a, "work/0007") {
		t.Fatal("untouched receipt prevented release")
	}
}

func TestUnacceptedClaimRetryRecognizesACompetingWinnerAtTheSameFoundation(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.withGit("if [ \"$3\" = push ]; then exit 1; fi", "claim", "0007").fails(2)
	attempt := a.gitOut(nil, "rev-parse", "refs/kit/claims/0007")
	b.run("claim", "0007").ok()
	winner := b.gitOut(nil, "rev-parse", "refs/kit/claims/0007")
	if attempt == winner {
		t.Fatal("competing attempts have identical receipts")
	}
	a.run("claim", "0007").fails(1)
	if a.resolves("refs/kit/claims/0007") || a.resolves("refs/heads/work/0007") {
		t.Fatal("losing attempt retained its empty lock")
	}
	if !hasRemoteBranch(t, b, "work/0007") {
		t.Fatal("winner was removed")
	}
}

func TestUnacceptedClaimCanRetryAndWorktreeCreationCanResume(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.withGit("if [ \"$3\" = push ]; then exit 1; fi", "claim", "0007").fails(2)
	receipt := a.gitOut(nil, "rev-parse", "refs/kit/claims/0007")
	path := filepath.Join(a.Root, WorktreesDirName, "work/0007")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "valuable"), []byte("preserve"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.run("claim", "0007").ok().has("already exists")
	if got := a.gitOut(nil, "rev-parse", "refs/heads/work/0007"); got != receipt {
		t.Fatal("retry replaced its receipt")
	}
	if err := os.Rename(path, path+"-preserved"); err != nil {
		t.Fatal(err)
	}
	a.run("claim", "0007").ok().has("worktree")
	if !a.claimWorktree("work/0007").exists("docs/features/0007-widget.md") {
		t.Fatal("retry did not create the worktree")
	}
}

func TestARenameDuringClaimCannotCreateTwoLocksForOneID(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	b.git("mv", "docs/features/0007-widget.md", "docs/features/0007-renamed.md")
	b.commitAll("approved rename")
	script := "if [ \"$3\" = push ]; then\n\"$real_git\" -C " + shellQuote(b.Root) + " push origin main >/dev/null 2>&1 || exit 99\n(cd " + shellQuote(b.Root) + " && PATH=" + shellQuote(os.Getenv("PATH")) + " " + shellQuote(kitBinary(t)) + " claim 0007) >/dev/null 2>&1 || exit 98\nfi"
	a.withGit(script, "claim", "0007").fails(1)
	if !hasRemoteBranch(t, b, "work/0007") {
		t.Fatal("renamed feature has no claim")
	}
	if hasRemoteBranch(t, a, "0007-widget") || hasRemoteBranch(t, b, "0007-renamed") {
		t.Fatal("rename created a second claim namespace")
	}
}

func TestLegacyClaimsRemainOwnedAndPreventCanonicalAcquisition(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.git("checkout", "-b", "0007-widget")
	a.git("push", "-u", "origin", "HEAD")
	b.run("claim", "0007").fails(1).errHas("0007-widget")
	a.write("docs/features/0007-widget.md", doc("0007", "Widget", "Draft", "", "- [x] Build it"))
	a.commitAll("legacy work")
	a.run("submit", "0007").ok()
	a.run("reopen", "0007", "review feedback").ok()
	a.run("submit", "0007").ok()
	a.git("push", "origin", "HEAD")
	a.git("checkout", "main")
	a.git("merge", "0007-widget")
	a.run("release", "0007").ok()
}

func TestConfirmedUntouchedClaimCanBeRebasedAndReleased(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	w := a.claimWorktree("work/0007")
	b.write("unrelated.txt", "new trunk\n")
	b.commitAll("unrelated change")
	b.git("push", "origin", "main")
	a.git("pull", "--ff-only", "origin", "main")
	w.git("rebase", "--keep-empty", "main")
	w.git("push", "--force-with-lease", "origin", "HEAD")
	a.run("claim", "0007").ok()
	a.run("release", "0007").ok()
	if hasRemoteBranch(t, a, "work/0007") || a.resolves("refs/kit/owned/0007") {
		t.Fatal("rebased empty claim could not be surrendered")
	}
}
