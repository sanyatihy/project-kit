package main

// Conformance: `kit drop`. Feature docs live on the default branch at every status,
// so an unwanted one cannot be disposed of by deleting a branch — without this it
// sits on the board for ever and every agent reads it as work nobody has started.

import (
	"strings"
	"testing"
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
	p := cleanProject(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] Something")
	p.commitAll("the plan")
	p.git("branch", "0001-shorten")

	r := p.run("drop", "0001", "not worth it").ok()
	r.has("still claimed", "0001-shorten")

	if !p.resolves("refs/heads/0001-shorten") {
		t.Error("drop deleted a branch, which is not its to delete")
	}
}
