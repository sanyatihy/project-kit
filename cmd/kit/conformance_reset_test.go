package main

// Conformance: `kit reopen`. It exists because an unrecognised status wedged the doc that
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

func TestReopenClearsAStatusNobodyCanShipOrClaim(t *testing.T) {
	t.Parallel()
	// The wedge, end to end: `claim` says it is not open work and `submit` refuses while a
	// box is open, so before this the only way out was the hand-edit that caused it.
	p := setUp(t)
	wedged(p, "0007", "stuck")
	p.commitAll("a status the kit never wrote")

	p.run("claim", "0007").fails(1)
	p.run("reopen", "0007", "--repair", "In Progress was never a status the kit writes").
		ok().has("reopen", "was In Progress", "Reopen 0007")

	got := p.read("docs/features/0007-stuck.md")
	if !strings.Contains(got, "status: Draft") {
		t.Errorf("reopen did not set the status:\n%s", got)
	}
	// The reason is the commit's, not the plan's: a reopen changes no plan.
	if strings.Contains(got, "In Progress was never a status the kit writes") {
		t.Errorf("reopen wrote its reason into the doc:\n%s", got)
	}
	if subject := p.gitOut(nil, "log", "-1", "--format=%s"); subject != "Reopen 0007: In Progress was never a status the kit writes" {
		t.Errorf("reopen's commit subject does not carry the reason: %q", subject)
	}
	// The repair is durable, which is 0066's rule for every command that writes a status.
	if head := p.gitOut(nil, "show", "HEAD:docs/features/0007-stuck.md"); !strings.Contains(head, "status: Draft") {
		t.Errorf("the committed doc is not the repaired one:\n%s", head)
	}
	// And the board hands it out again, which is the whole point of clearing it.
	p.run("audit").ok().lacks("no-status")
	p.run("claim", "0007").ok()
}

func TestTheWayOutOfAWedgedStatusIsACommand(t *testing.T) {
	t.Parallel()
	// The finding a 27-session chain repaired by breaking the rule it was told to follow:
	// `use one of: Draft, Dropped, Submitted` is an instruction to hand-edit the field the
	// kit owns. Both messages an agent meets here now name the command instead.
	p := setUp(t)
	wedged(p, "0007", "stuck")
	p.commitAll("a status the kit never wrote")

	p.run("audit").ok().has("no-status", `kit reopen 0007 --repair "<why>"`)
	p.run("next").ok().has("kit reopen 0007")
}

func TestReopenWritesAStatusLineADocNeverHad(t *testing.T) {
	t.Parallel()
	// The other half of no-status, and the same wedge: `claim` refuses a doc whose status
	// is not Draft, and a doc with no status: line has none to be.
	p := setUp(t)
	p.write("docs/features/0007-old.md",
		"---\nid: 0007\ntitle: From an older kit\ncreated: 2026-01-01\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [ ] Build it\n\n## Log\n- 2026-01-01 — x.\n")
	p.commitAll("a doc from a kit that had no statuses")

	p.run("reopen", "0007", "--repair", "it was written before the field existed").ok()

	if got := p.read("docs/features/0007-old.md"); !strings.Contains(got, "status: Draft") {
		t.Errorf("reopen did not write the missing status line:\n%s", got)
	}
	p.run("audit").ok().lacks("no-status")
	p.run("claim", "0007").ok()
}

func TestReopenDoesNotReviveADroppedPlan(t *testing.T) {
	t.Parallel()
	// One rule for both: any status but Submitted is a plan, and a retired one somebody
	// wants back needs a command or it needs a hand-edit.
	p := setUp(t)
	feature(p, "0007", "cut", "Cut", "Draft", "- [ ] Build it")
	p.run("drop", "0007", "the constraint it fought went away").ok()

	p.run("reopen", "0007", "revived: the constraint came back").fails(2).errHas("new feature")
	if got := p.read("docs/features/0007-cut.md"); !strings.Contains(got, "status: Dropped") {
		t.Errorf("reopen changed the doc:\n%s", got)
	}
}

func TestReopenRefusesASubmittedDoc(t *testing.T) {
	t.Parallel()
	// Frozen history: the work is in the repo, and a status saying otherwise would make
	// the record say it was never done.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")
	trunk := p.headBranch()
	submitting(p, "0007").ok()
	p.git("checkout", trunk)
	p.git("merge", "0007-done")

	p.run("reopen", "0007", "I would rather it were open").fails(2).errHas("Submitted")
	if got := p.read("docs/features/0007-done.md"); !strings.Contains(got, "status: Submitted") {
		t.Errorf("reopen rewrote a Submitted doc despite refusing:\n%s", got)
	}
}

func TestReopenRefusesADocThatIsAlreadyDraft(t *testing.T) {
	t.Parallel()
	// Nothing to repair, and the commit it would make has nothing in it.
	p := setUp(t)
	feature(p, "0007", "open", "Open", "Draft", "- [ ] Build it")
	p.commitAll("the plan")

	p.run("reopen", "0007", "no reason at all").fails(2).errHas("already Draft")
}

func TestReopenRefusesAFeatureAnotherCheckoutClaims(t *testing.T) {
	t.Parallel()
	// `submit`'s guard, for `submit`'s reason: a doc another agent holds is work in flight,
	// and rewriting its status from here is the same interference.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok() // the branch is the claim; this checkout is not on it
	a.write("docs/features/0007-widget.md", doc("0007", "A widget", "In Progress", "", "- [ ] Build it"))

	a.run("reopen", "0007", "--repair", "clearing it from the wrong checkout").fails(2).
		errHas("work/0007", "kit release 0007")
	if strings.Contains(a.read("docs/features/0007-widget.md"), "status: Draft") {
		t.Error("reopen wrote the status despite refusing")
	}
}

func TestReopenSaysWhenNoReasonWasGiven(t *testing.T) {
	t.Parallel()
	// `drop`'s nudge: refusing an empty reason only gets an empty string typed at it.
	p := setUp(t)
	wedged(p, "0007", "stuck")
	p.commitAll("a status the kit never wrote")

	p.run("reopen", "0007", "--repair").ok().has("no reason recorded")
}
