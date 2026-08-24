package main

// Conformance: `kit reset`. It exists because an unrecognised status wedged the doc that
// carried it, and every way out named an edit to the field the kit is meant to own.

import (
	"strings"
	"testing"
)

// wedged writes a doc carrying a status the kit does not recognise — the state a hand-edit
// produces, and the one no other command could clear.
func wedged(p *project, id, slug string) {
	p.t.Helper()
	p.write("docs/features/"+id+"-"+slug+".md",
		doc(id, "Wedged", "In Progress", "", "- [ ] Build it"))
}

func TestResetClearsAStatusNobodyCanShipOrClaim(t *testing.T) {
	t.Parallel()
	// The wedge, end to end: `claim` says it is not open work and `ship` refuses while a
	// box is open, so before this the only way out was the hand-edit that caused it.
	p := setUp(t)
	wedged(p, "0007", "stuck")
	p.commitAll("a status the kit never wrote")

	p.run("claim", "0007").fails(1)
	p.run("reset", "0007", "In Progress was never a status the kit writes").
		ok().has("reset", "was In Progress", "Reset 0007")

	got := p.read("docs/features/0007-stuck.md")
	if !strings.Contains(got, "status: Draft") {
		t.Errorf("reset did not set the status:\n%s", got)
	}
	if !strings.Contains(got, "In Progress was never a status the kit writes") {
		t.Errorf("reset did not record why in ## Log:\n%s", got)
	}
	// The repair is durable, which is 0066's rule for every command that writes a status.
	if head := p.gitOut(nil, "show", "HEAD:docs/features/0007-stuck.md"); !strings.Contains(head, "status: Draft") {
		t.Errorf("the committed doc is not the repaired one:\n%s", head)
	}
	// And the board hands it out again, which is the whole point of clearing it.
	p.run("audit").ok().lacks("no-status")
	p.run("claim", "0007").ok()
}

func TestResetRevivesADroppedPlan(t *testing.T) {
	t.Parallel()
	// One rule for both: any status but Shipped is a plan, and a retired one somebody
	// wants back needs a command or it needs a hand-edit.
	p := setUp(t)
	feature(p, "0007", "cut", "Cut", "Draft", "- [ ] Build it")
	p.run("drop", "0007", "the constraint it fought went away").ok()

	p.run("reset", "0007", "revived: the constraint came back").ok().has("was Dropped")
	if got := p.read("docs/features/0007-cut.md"); !strings.Contains(got, "status: Draft") {
		t.Errorf("reset did not revive the doc:\n%s", got)
	}
}

func TestResetRefusesAShippedDoc(t *testing.T) {
	t.Parallel()
	// Frozen history: the work is in the repo, and a status saying otherwise would make
	// the record say it was never done.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")
	shipping(p, "0007").ok()

	p.run("reset", "0007", "I would rather it were open").fails(2).errHas("Shipped")
	if got := p.read("docs/features/0007-done.md"); !strings.Contains(got, "status: Shipped") {
		t.Errorf("reset rewrote a Shipped doc despite refusing:\n%s", got)
	}
}

func TestResetRefusesADocThatIsAlreadyDraft(t *testing.T) {
	t.Parallel()
	// Nothing to repair, and the commit it would make has nothing in it.
	p := setUp(t)
	feature(p, "0007", "open", "Open", "Draft", "- [ ] Build it")
	p.commitAll("the plan")

	p.run("reset", "0007", "no reason at all").fails(2).errHas("already Draft")
}

func TestResetRefusesAFeatureAnotherCheckoutClaims(t *testing.T) {
	t.Parallel()
	// `ship`'s guard, for `ship`'s reason: a doc another agent holds is work in flight,
	// and rewriting its status from here is the same interference.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok() // the branch is the claim; this checkout is not on it
	a.write("docs/features/0007-widget.md", doc("0007", "A widget", "In Progress", "", "- [ ] Build it"))

	a.run("reset", "0007", "clearing it from the wrong checkout").fails(2).
		errHas("0007-widget", "kit release 0007")
	if strings.Contains(a.read("docs/features/0007-widget.md"), "status: Draft") {
		t.Error("reset wrote the status despite refusing")
	}
}

func TestResetSaysWhenNoReasonWasGiven(t *testing.T) {
	t.Parallel()
	// `drop`'s nudge: refusing an empty reason only gets an empty string typed at it.
	p := setUp(t)
	wedged(p, "0007", "stuck")
	p.commitAll("a status the kit never wrote")

	p.run("reset", "0007").ok().has("no reason recorded")
}
