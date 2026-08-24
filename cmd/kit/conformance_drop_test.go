package main

// Conformance: `kit drop`. Feature docs live on the default branch at every status,
// so an unwanted one cannot be disposed of by deleting a branch — without this it
// sits on the board for ever and every agent reads it as work nobody has started.

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestDropRetiresAFeatureAndRecordsWhy(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	path := "docs/features/0001-shorten.md"
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")

	p.run("drop", "0001", "superseded by 0002's router").ok().has("dropped", path)

	got := p.read(path)
	if !strings.Contains(got, "status: Dropped") {
		t.Errorf("drop did not set the status:\n%s", got)
	}
	// The reason goes in the ## Log with every other durable decision. A dropped doc
	// with no reason reads the same as a forgotten one.
	if !strings.Contains(got, "superseded by 0002's router") {
		t.Errorf("drop did not record why in ## Log:\n%s", got)
	}

	// And it leaves the board: Dropped is terminal, so next stops offering it.
	p.run("next").ok().lacks("0001")
}

func TestDropCommitsTheStatusItWrites(t *testing.T) {
	t.Parallel()
	// `ship`'s reason: a status only the working tree carries is one session from being
	// read as debris. The doc's path is the whole commit — a retired plan is often edited
	// in the same breath, and nothing else in the tree is this command's to take.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")
	p.commitAll("the plan")
	p.write("probe-output.json", "{}\n")

	p.run("drop", "0001", "the metric it chased moved on its own").ok().has("Drop 0001")

	head := p.gitOut(nil, "show", "HEAD:docs/features/0001-shorten.md")
	if !strings.Contains(head, "status: Dropped") {
		t.Errorf("the committed doc is not the retired one:\n%s", head)
	}
	if left := p.gitOut(nil, "status", "--porcelain"); left != "?? probe-output.json" {
		t.Errorf("drop committed something that was not the doc, or lost it: %q", left)
	}
}

func TestDropSaysWhenNoReasonWasGiven(t *testing.T) {
	t.Parallel()
	// Not an error — refusing would only get an empty string typed at it. Saying so
	// is what makes the omission visible while the person is still here.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] Something")

	p.run("drop", "0001").ok().has("no reason recorded")
}

func TestDropRefusesAShippedDoc(t *testing.T) {
	t.Parallel()
	// Frozen history, and the work is in the repo. Retiring it would make the record
	// say the feature was never built.
	p := setUp(t)
	feature(p, "0001", "done", "Done", "Shipped", "- [x] Finished")

	p.run("drop", "0001", "changed my mind").fails(2).errHas("frozen history")
}

func TestDropRefusesADocThatDeclaresAKeyTwice(t *testing.T) {
	t.Parallel()
	// Same write, same ambiguity: retiring the first status: leaves the last one saying
	// the feature is still open, and the command reports success.
	p := setUp(t)
	p.write("docs/features/0001-twice.md",
		"---\nid: 0001\ntitle: Twice\nstatus: Draft\nstatus: Draft\n"+
			"created: 2026-01-01\n---\n\n## Problem\nx\n\n## Design\nx\n\n"+
			"## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	p.run("drop", "0001", "changed my mind").fails(2).errHas("status", "twice")
	if strings.Contains(p.read("docs/features/0001-twice.md"), "status: Dropped") {
		t.Error("drop wrote the status despite refusing")
	}
}

func TestDropRefusesAnIDThatNamesTwoDocs(t *testing.T) {
	t.Parallel()
	// Reproduced: drop retired the first of the two and printed nothing about the second,
	// so the board went on offering a feature whose id had just been declared dead.
	p := setUp(t)
	p.write("docs/features/0001-one.md", doc("0001", "One", "Draft", "", "- [ ] Something"))
	p.write("docs/features/0001-two.md", doc("0001", "Two", "Draft", "", "- [ ] Something"))

	p.run("drop", "0001", "changed my mind").fails(2).errHas("0001-one.md", "0001-two.md")
	for _, rel := range []string{"docs/features/0001-one.md", "docs/features/0001-two.md"} {
		if strings.Contains(p.read(rel), "status: Dropped") {
			t.Errorf("%s was retired despite the refusal", rel)
		}
	}
}

func TestDropReportsWhoWasWaitingOnIt(t *testing.T) {
	t.Parallel()
	// Dropping stranded every feature declaring the id in needs: without a word — `audit`
	// said needs-unknown afterwards and `next` went on promising a wait that can never
	// end. Reported and not refused: this is exactly when you want to know who was waiting.
	stranding := setUp(t)
	feature(stranding, "0001", "base", "Base", "Draft", "- [ ] Something")
	stranding.write("docs/features/0002-store.md",
		doc("0002", "Store", "Draft", "needs: 0001", "- [ ] Do it"))

	r := stranding.run("drop", "0001", "not worth it").ok()
	r.has("still needed by", "0002", "needs:")

	// And silence when nobody was waiting, or the line becomes noise nobody reads.
	alone := setUp(t)
	feature(alone, "0001", "base", "Base", "Draft", "- [ ] Something")
	alone.run("drop", "0001", "not worth it").ok().lacks("still needed by")
}

func TestDropCreatesTheLogWhenTheDocHasNone(t *testing.T) {
	t.Parallel()
	// Measured: on a doc with no ## Log the reason was discarded. AppendToSection reports
	// that the heading was absent and Drop ignored the answer, while the one warning it
	// prints fires only when *no* reason was given — so the operator who supplied one was
	// never told. Recording the reason is what the command is for.
	missing := setUp(t)
	missing.write("docs/features/0001-nolog.md",
		"---\nid: 0001\ntitle: No log\nstatus: Draft\ncreated: 2026-01-01\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [ ] Something\n")

	missing.run("drop", "0001", "superseded by a better idea").ok()

	got := missing.read("docs/features/0001-nolog.md")
	if !strings.Contains(got, "## Log") {
		t.Errorf("drop did not create the ## Log it had to write into:\n%s", got)
	}
	if !strings.Contains(got, "superseded by a better idea") {
		t.Errorf("the reason was discarded:\n%s", got)
	}

	// A doc that already has one gets the entry appended to it, not a second heading.
	present := setUp(t)
	feature(present, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] Something")
	present.run("drop", "0001", "not worth it").ok()
	if got := present.read("docs/features/0001-shorten.md"); strings.Count(got, "## Log") != 1 {
		t.Errorf("drop added a second ## Log:\n%s", got)
	}
}

func TestDropReportsALeftoverClaimRatherThanDeletingIt(t *testing.T) {
	t.Parallel()
	// The doc is this command's to retire and the branch is not. A branch named after
	// the doc still reads as a claim, so the board would show work nobody is doing.
	// setUp and not cleanProject: that fixture already carries a doc with id 0001, and
	// `drop` refuses an id naming two docs rather than retiring whichever sorted first.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] Something")
	p.commitAll("the plan")
	p.git("branch", "0001-shorten")

	r := p.run("drop", "0001", "not worth it").ok()
	r.has("still claimed", "0001-shorten")

	if !p.resolves("refs/heads/0001-shorten") {
		t.Error("drop deleted a branch, which is not its to delete")
	}
}

// reserve makes a reservation by hand, dated, so a test can put one on either side of the
// horizon. `kit new` makes one too, and can only ever date it now.
func (p *project) reserve(id, title string, daysAgo int) {
	p.t.Helper()
	// RFC 3339: git rejects "30 days ago" here, where it accepts it on the command line.
	when := time.Now().AddDate(0, 0, -daysAgo).Format(time.RFC3339)
	tree := p.gitOut(nil, "hash-object", "-t", "tree", "--stdin")
	sha := p.gitOut([]string{"GIT_AUTHOR_DATE=" + when, "GIT_COMMITTER_DATE=" + when},
		"commit-tree", tree, "-m", id+" "+title)
	p.git("push", "-q", "origin", sha+":refs/kit/ids/"+id)
}

// gitOut is git read for its answer rather than its effect, with extra environment for
// the one caller that has to control a commit's date.
func (p *project) gitOut(extra []string, args ...string) string {
	p.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = p.Root, gitEnv(extra...)
	out, err := cmd.Output()
	if err != nil {
		p.t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

func TestDropReleasesAReservationNoDocHolds(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	// The state four ids reached: a number handed out for a round that was then cut, held
	// by no doc anywhere, and reachable by no command — `drop` needs a doc to write into.
	a.reserve("0009", "Ninth", 30)

	a.run("drop", "0009", "the round was cut and pivoted into 0007").
		ok().has("released", "0009", "Ninth")

	if got := a.reservedIDs(); slices.Contains(got, "0009") {
		t.Errorf("the remote still reserves %v — the number is out of circulation with "+
			"nothing describing it", got)
	}
	// Free means back in circulation: 0008 is the next number on disk, and 0009 is the
	// one after it only because nothing reserves it any more.
	a.run("new", "Eighth").ok().has("0008-eighth.md")
	a.run("new", "Ninth").ok().has("0009-ninth.md")
}

func TestDropRefusesAReservationYoungerThanTheHorizon(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.reserve("0009", "Ninth", 1)

	// A reservation made a moment ago is an ordinary draft being written somewhere else,
	// and a release that fires on those hands out a number two people are using.
	a.run("drop", "0009", "looks unused").fails(2).errHas("draft in flight")

	if got := a.reservedIDs(); !slices.Contains(got, "0009") {
		t.Errorf("the remote holds %v — the reservation was released anyway", got)
	}
}

func TestDropRefusesToReleaseAnIDADraftStillHolds(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.reserve("0009", "Ninth", 30)

	// A round in flight: the doc is on its own branch and nowhere else, so from the trunk
	// the id looks exactly like one nobody used.
	trunk := a.headBranch()
	a.git("switch", "-q", "-c", "plan/round")
	feature(a, "0009", "ninth", "Ninth", "Draft", "- [ ] Build it")
	a.commitAll("the round")
	a.git("switch", "-q", trunk)

	a.run("drop", "0009", "looks unused").fails(2).errHas("plan/round")

	if got := a.reservedIDs(); !slices.Contains(got, "0009") {
		t.Errorf("the remote holds %v — the draft's number was released under it", got)
	}
}

func TestDropSaysWhenAnIDIsFreeAlready(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)

	// No doc and no reservation. The old answer was "no feature 0009", which reads as a
	// typo rather than as the number already being available.
	a.run("drop", "0009", "tidying").fails(2).errHas("free already")
}
