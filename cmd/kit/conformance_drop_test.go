package main

// Conformance: `kit drop`. Feature docs live on the default branch at every status,
// so an unwanted one cannot be disposed of by deleting a branch — without this it
// sits on the board for ever and every agent reads it as work nobody has started.

import (
	"strings"
	"testing"
)

func TestDropRetiresAFeatureAndRecordsWhy(t *testing.T) {
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
	// Not an error — refusing would only get an empty string typed at it. Saying so
	// is what makes the omission visible while the person is still here.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] Something")

	p.run("drop", "0001").ok().has("no reason recorded")
}

func TestDropRefusesAShippedDoc(t *testing.T) {
	// Frozen history, and the work is in the repo. Retiring it would make the record
	// say the feature was never built.
	p := setUp(t)
	feature(p, "0001", "done", "Done", "Shipped", "- [x] Finished")

	p.run("drop", "0001", "changed my mind").fails(2).errHas("frozen history")
}

func TestDropReportsALeftoverClaimRatherThanDeletingIt(t *testing.T) {
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
