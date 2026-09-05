package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidPlansCannotBeClaimedOrShipped(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, extra, want string
	}{
		{"self", "needs: 0007", "dependency cycle"},
		{"malformed", "needs: tomorrow", "not an id list"},
		{"duplicate", "status: Draft", "twice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := setUp(t)
			p.write("docs/features/0007-widget.md", doc("0007", "Widget", "Draft", tc.extra, "- [ ] Build"))
			p.commitAll("plan")
			p.run("next").ok().has("blocked", tc.want)
			p.run("claim", "0007").fails(1).errHas(tc.want)
			p.write("docs/features/0007-widget.md", doc("0007", "Widget", "Draft", tc.extra, "- [x] Build"))
			shipping(p, "0007").fails(2).errHas(tc.want)
		})
	}
}

func TestDependencyCycleNamesTheProblemInsteadOfPromisingAWait(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.write("docs/features/0001-one.md", doc("0001", "One", "Draft", "needs: 0002", "- [ ] Build"))
	p.write("docs/features/0002-two.md", doc("0002", "Two", "Draft", "needs: 0001", "- [ ] Build"))
	p.commitAll("plan")
	p.run("audit").ok().has("needs-unknown", "dependency cycle")
	p.run("claim", "0001").fails(1).errHas("dependency cycle")
}

func TestShipRefusesBlockedUnwrittenAndUnknownPlans(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"blocked", "unwritten", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			p := setUp(t)
			feature(p, "0007", "widget", "Widget", "Draft", "- [x] Build")
			want := ""
			switch kind {
			case "blocked":
				p.run("block", "0007", "Which behavior is correct?").ok()
				want = "blocked"
			case "unwritten":
				p.write("docs/features/0007-widget.md", strings.Replace(p.read("docs/features/0007-widget.md"), "## Design", "## Other", 1))
				want = "no plan to submit"
			case "unknown":
				p.write("docs/features/0007-widget.md", strings.Replace(p.read("docs/features/0007-widget.md"), "status: Draft", "status: Surprise", 1))
				want = "unknown status"
			}
			shipping(p, "0007").fails(2).errHas(want)
		})
	}
}

func TestDropCannotRetireAnotherCheckoutsClaim(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	before := b.read("docs/features/0007-widget.md")
	b.run("drop", "0007", "stop").fails(2).errHas("not this checkout")
	if b.read("docs/features/0007-widget.md") != before {
		t.Fatal("refusal changed another checkout's plan")
	}
}

func TestReleasePreservesDirtyAndDocumentOnlyWork(t *testing.T) {
	t.Parallel()
	for _, committed := range []bool{false, true} {
		t.Run(map[bool]string{false: "dirty", true: "document commit"}[committed], func(t *testing.T) {
			a, _ := twoAgents(t)
			a.run("claim", "0007").ok()
			w := a.claimWorktree("0007-widget")
			w.write("docs/features/0007-widget.md", w.read("docs/features/0007-widget.md")+"\nReview decision.\n")
			if committed {
				w.commitAll("revise plan")
			}
			a.run("release", "0007").fails(2).errHas("work")
			if !hasRemoteBranch(t, a, "0007-widget") || !w.exists("docs/features/0007-widget.md") {
				t.Fatal("release surrendered or removed unfinished work")
			}
		})
	}
}

func TestReleaseCannotDeleteAPushThatArrivesAfterItsRead(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("fetch", "origin")
	b.git("switch", "-c", "0007-widget", "origin/0007-widget")
	b.write("work.txt", "concurrent progress")
	b.commitAll("progress")
	newTip := strings.TrimSpace(b.gitOut(nil, "rev-parse", "HEAD"))
	remote := strings.TrimSpace(a.gitOut(nil, "remote", "get-url", "origin"))
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	shim := t.TempDir()
	// Change the remote just before the delete push, after kit's ls-remote completed.
	script := "#!/bin/sh\nif [ \"$3\" = push ]; then\n" +
		shellQuote(realGit) + " --git-dir=" + shellQuote(remote) + " fetch " + shellQuote(b.Root) +
		" +refs/heads/0007-widget:refs/heads/0007-widget >/dev/null 2>&1 || exit 99\nfi\nexec " +
		shellQuote(realGit) + " \"$@\"\n"
	writeExecutable(t, filepath.Join(shim, "git"), script)
	cmd := exec.Command(kitBinary(t), "release", "0007")
	cmd.Dir, cmd.Env = a.Root, gitEnv("PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("release deleted a concurrent push: %s", out)
	}
	if got := a.gitOut(nil, "ls-remote", "--heads", "origin", "0007-widget"); !strings.Contains(got, newTip) {
		t.Fatalf("remote progress lost: %s", got)
	}
}

func TestReviewBoardReadsTheAdvertisedTip(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	w := a.claimWorktree("0007-widget")
	b.git("fetch", "origin")
	b.git("branch", "0007-widget", "origin/0007-widget")
	w.write("docs/features/0007-widget.md", doc("0007", "Widget", "Draft", "", "- [x] Build"))
	shipping(w, "0007").ok()
	w.git("push", "origin", "HEAD")
	b.git("fetch", "origin")
	// b's local branch remains Draft while its remote-tracking branch is Shipped.
	b.run("next").ok().has("awaiting review")
}

func TestStrictAuditFailsWhenTheRemoteCannotAnswer(t *testing.T) {
	t.Parallel()
	p := cleanProject(t)
	p.git("remote", "add", "origin", p.path("missing-remote"))
	p.run("audit", "--strict").fails(2).has("no findings in the checks that ran")
	p.run("audit", "--strict", "--offline").ok()
}

func TestReviewFeedbackCanReopenOnlyUnmergedWork(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	w := a.claimWorktree("0007-widget")
	w.write("docs/features/0007-widget.md", doc("0007", "Widget", "Draft", "", "- [x] Build"))
	shipping(w, "0007").ok()
	w.run("reset", "0007", "review requested another case").ok()
	if got := w.read("docs/features/0007-widget.md"); !strings.Contains(got, "status: Draft") || strings.Contains(got, "shipped: 20") {
		t.Fatalf("reset did not clear the submission: %s", got)
	}
	w.run("ship", "0007").ok()
	a.git("merge", "--no-ff", "-m", "reviewed", "0007-widget")
	a.git("push", "origin", "HEAD")
	w.run("reset", "0007", "try reopening history").fails(2).errHas("frozen history")
}

func TestInitPreservesDanglingUserSymlinks(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	if err := os.MkdirAll(p.path("docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-target.md", p.path("docs/VISION.md")); err != nil {
		t.Fatal(err)
	}
	p.run("init").ok()
	if p.exists("docs/missing-target.md") {
		t.Fatal("init wrote through the user's dangling symlink")
	}
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

func writeExecutable(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLintGatePropagatesFailures(t *testing.T) {
	t.Parallel()
	for _, path := range []string{makefilePath, "../../templates/stacks/go/Makefile"} {
		t.Run(path, func(t *testing.T) {
			makefile := readOrFail(t, path)
			p := newProject(t)
			p.write("Makefile", makefile)
			shim := t.TempDir()
			writeExecutable(t, filepath.Join(shim, "golangci-lint"), "#!/bin/sh\necho 'lint failed deliberately' >&2\nexit 9\n")
			cmd := exec.Command("make", "lint")
			cmd.Dir, cmd.Env = p.Root, append(os.Environ(), "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
			out, err := cmd.CombinedOutput()
			if err == nil || strings.Contains(string(out), "not installed") {
				t.Fatalf("gate swallowed lint failure: %v, %s", err, out)
			}
		})
	}
}

func TestFrozenDocumentEditsAndDeletionsFailTheGate(t *testing.T) {
	t.Parallel()
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "edit", true: "delete"}[remove], func(t *testing.T) {
			p := setUp(t)
			feature(p, "0007", "widget", "Widget", "Draft", "- [x] Done")
			shipping(p, "0007").ok()
			p.git("switch", "-c", "followup")
			if remove {
				if err := os.Remove(p.path("docs/features/0007-widget.md")); err != nil {
					t.Fatal(err)
				}
			} else {
				p.write("docs/features/0007-widget.md", p.read("docs/features/0007-widget.md")+"\nRevised history.\n")
			}
			p.commitAll("change frozen history")
			p.run("audit", "--strict").fails(1).has("frozen-doc")
		})
	}
}

func TestShipDoesNotTreatAFailedGitStatusAsClean(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	feature(p, "0007", "widget", "Widget", "Draft", "- [x] Done")
	p.commitAll("done")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	shim := t.TempDir()
	writeExecutable(t, filepath.Join(shim, "git"), "#!/bin/sh\nif [ \"$3\" = status ]; then echo 'status unreadable' >&2; exit 9; fi\nexec "+shellQuote(realGit)+" \"$@\"\n")
	cmd := exec.Command(kitBinary(t), "ship", "0007")
	cmd.Dir, cmd.Env = p.Root, gitEnv("PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "status unreadable") {
		t.Fatalf("unreadable state did not refuse shipping: %v, %s", err, out)
	}
	if strings.Contains(p.read("docs/features/0007-widget.md"), "status: Shipped") {
		t.Fatal("failed status read still submitted the feature")
	}
}

func TestReviewResetRefusesAnUnreadablePublishedTrunk(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	w := a.claimWorktree("0007-widget")
	w.write("docs/features/0007-widget.md", doc("0007", "Widget", "Draft", "", "- [x] Build"))
	shipping(w, "0007").ok()
	b.write("other.txt", "a new trunk commit")
	b.commitAll("move trunk")
	b.git("push", "origin", "HEAD")
	w.run("reset", "0007", "review").fails(2).errHas("run git fetch")
	w.git("fetch", "origin")
	w.run("reset", "0007", "review").ok()
}

func TestConcurrentBlockingQuestionsAreAllPreserved(t *testing.T) {
	t.Parallel()
	p := cleanProject(t)
	const count = 12
	before := strings.Count(p.read("docs/INBOX.md"), "BLOCKING")
	start := make(chan struct{})
	done := make(chan string, count)
	for i := 0; i < count; i++ {
		go func(i int) {
			<-start
			cmd := exec.Command(kitBinary(t), "block", "0001", strings.Repeat("question ", i+1))
			cmd.Dir, cmd.Env = p.Root, gitEnv()
			out, err := cmd.CombinedOutput()
			if err != nil {
				done <- string(out)
				return
			}
			done <- ""
		}(i)
	}
	close(start)
	for i := 0; i < count; i++ {
		if err := <-done; err != "" {
			t.Fatal(err)
		}
	}
	if got := strings.Count(p.read("docs/INBOX.md"), "BLOCKING"); got != before+count {
		t.Fatalf("inbox has %d new questions, want %d", got-before, count)
	}
}

func TestUnfetchedDraftCannotLoseItsReservation(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.reserve("0009", "Ninth", 30)
	b.git("switch", "-c", "plan/remote-round")
	feature(b, "0009", "ninth", "Ninth", "Draft", "- [ ] Build")
	b.commitAll("remote draft")
	b.git("push", "origin", "HEAD")
	a.run("drop", "0009", "looks unused").fails(2).errHas("git fetch")
	a.git("fetch", "origin")
	a.run("drop", "0009", "looks unused").fails(2).errHas("plan/remote-round")
	if !strings.Contains(a.gitOut(nil, "ls-remote", "origin", "refs/kit/ids/0009"), "refs/kit/ids/0009") {
		t.Fatal("live draft lost its reservation")
	}
}
