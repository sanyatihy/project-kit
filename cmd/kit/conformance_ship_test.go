package main

// Conformance: `kit ship`. Its value is what it refuses — two of the audit's
// findings exist only because this is normally done by hand.

import (
	"strings"
	"testing"
	"time"
)

// shipping commits what the fixture wrote, then ships. A tree with nothing uncommitted
// in it is the state the ship gate describes — the last box ticked in the commit that
// implemented it — and ship refuses anything else.
func shipping(p *project, args ...string) *result {
	p.t.Helper()
	p.commitAll("the work")
	return p.run(append([]string{"ship"}, args...)...)
}

func TestShipRefusesWhileABoxIsOpen(t *testing.T) {
	t.Parallel()
	// unticked-shipped, prevented rather than reported.
	p := setUp(t)
	feature(p, "0007", "half", "Half done", "Draft", "- [x] One\n- [ ] Two")

	r := shipping(p, "0007").fails(2)
	r.errHas("unticked", "Two")
	if strings.Contains(p.read("docs/features/0007-half.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipRefusesDebrisInTheWorktree(t *testing.T) {
	t.Parallel()
	// The scratch a feature generates is the agent's to clean up. Shipping over it
	// either commits it under a status change or leaves it for whoever works here next.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One")
	p.commitAll("the work")
	p.write("probe-output.json", "{}\n")

	p.run("ship", "0007").fails(2).errHas("uncommitted or untracked", "probe-output.json")

	if strings.Contains(p.read("docs/features/0007-done.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipIgnoresWhatGitIgnores(t *testing.T) {
	t.Parallel()
	// Build output is not debris an agent has to explain, and a refusal that fired on
	// it would be one every project learns to work around.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One")
	p.write(".gitignore", "build/\n")
	p.commitAll("the work")
	p.write("build/kit", "not source")

	p.run("ship", "0007").ok()
}

func TestShipStampsTheDate(t *testing.T) {
	t.Parallel()
	// shipped-no-date, prevented rather than reported.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One\n- [x] Two")

	shipping(p, "0007").ok().has("shipped")

	doc := p.read("docs/features/0007-done.md")
	if !strings.Contains(doc, "status: Shipped") {
		t.Errorf("status was not set:\n%s", doc)
	}
	if want := "shipped: " + time.Now().UTC().Format("2006-01-02"); !strings.Contains(doc, want) {
		t.Errorf("date was not stamped, want %q:\n%s", want, doc)
	}
	p.run("audit").ok().lacks("shipped-no-date", "unticked-shipped")
}

func TestShipFillsAnExistingEmptyDateKey(t *testing.T) {
	t.Parallel()
	// The template ships a bare "shipped:" line, so this is the common path — and
	// it is a different branch from inserting one.
	p := setUp(t)
	p.write("docs/features/0007-templated.md",
		"---\nid: 0007\ntitle: Templated\nstatus: Draft\ncreated: 2026-01-01\nshipped:\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	shipping(p, "0007").ok()

	doc := p.read("docs/features/0007-templated.md")
	if want := "shipped: " + time.Now().UTC().Format("2006-01-02"); !strings.Contains(doc, want) {
		t.Errorf("the empty shipped: key was not filled, want %q:\n%s", want, doc)
	}
	if strings.Count(doc, "shipped:") != 1 {
		t.Errorf("ship produced a duplicate shipped: key:\n%s", doc)
	}
	p.run("audit").ok().lacks("shipped-no-date")
}

func TestShipAddsTheDateKeyWhenTheDocHasNone(t *testing.T) {
	t.Parallel()
	// An older doc may have no shipped: line at all. Stamping has to insert one
	// rather than silently do nothing.
	p := setUp(t)
	p.write("docs/features/0007-old.md",
		"---\nid: 0007\ntitle: Old\nstatus: Draft\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	shipping(p, "0007").ok()
	if !strings.Contains(p.read("docs/features/0007-old.md"), "shipped: ") {
		t.Error("no shipped: key was added")
	}
}

func TestShipRefusesAFeatureWithNoBoxesAtAll(t *testing.T) {
	t.Parallel()
	// Nothing to have finished. Shipping it would assert work that was never planned.
	p := setUp(t)
	feature(p, "0007", "empty", "Empty", "Draft", "")
	shipping(p, "0007").fails(2).errHas("no ## Tasks boxes")
}

func TestShipRefusesTwice(t *testing.T) {
	t.Parallel()
	// The second run would rewrite frozen history and restamp the date.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")
	shipping(p, "0007").ok()
	shipping(p, "0007").fails(2).errHas("already Shipped")
}

func TestShipRefusesWhileItsNeedsHaveNotShipped(t *testing.T) {
	t.Parallel()
	// shipped-early, prevented rather than reported: `ship` was minting a finding on a
	// doc the protocol then forbids anyone to edit. Exit 2 and not 1 — nobody else holds
	// this, the work is done and the plan is wrong, so the operator has to look.
	p := setUp(t)
	feature(p, "0001", "base", "Base", "Draft", "- [ ] Do it")
	p.write("docs/features/0002-store.md",
		doc("0002", "Store", "Draft", "needs: 0001", "- [x] Done"))

	shipping(p, "0002").fails(2).errHas("0001", "not shipped")
	if strings.Contains(p.read("docs/features/0002-store.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
	p.run("audit").ok().lacks("shipped-early")
}

func TestShipAllowsAFeatureWhoseNeedsHaveShipped(t *testing.T) {
	t.Parallel()
	// The quiet half: the foundation is there, so nothing is in the way.
	p := setUp(t)
	p.write("docs/features/0001-base.md",
		doc("0001", "Base", "Shipped", "shipped: 2026-01-02", "- [x] Do it"))
	p.write("docs/features/0002-store.md",
		doc("0002", "Store", "Draft", "needs: 0001", "- [x] Done"))

	shipping(p, "0002").ok().has("shipped")
}

func TestShipRefusesAFeatureAnotherCheckoutClaims(t *testing.T) {
	t.Parallel()
	// Measured from a checkout whose own board says the feature is taken: `ship` exited 0
	// and froze another agent's work mid-flight. `status: Shipped` is history the protocol
	// forbids anyone to edit, so the agent holding the branch could no longer touch its
	// own doc — and `audit` reported nothing at all.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok() // the branch is the claim; this checkout is not on it
	a.write("docs/features/0007-widget.md",
		doc("0007", "A widget", "Draft", "", "- [x] Build it"))

	r := shipping(a, "0007").fails(2)
	// Naming the branch and the way out: refusing without one is how an agent gets stuck.
	r.errHas("0007-widget", "rule 8")
	if strings.Contains(a.read("docs/features/0007-widget.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipFromTheCheckoutHoldingTheClaimIsFine(t *testing.T) {
	t.Parallel()
	// The quiet half, and the documented path: the worktree that holds the claim is
	// where the work was done and where it ships from.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	a.git("checkout", "-q", "0007-widget")
	a.write("docs/features/0007-widget.md",
		doc("0007", "A widget", "Draft", "", "- [x] Build it"))

	shipping(a, "0007").ok().has("shipped")
}

func TestShipRefusesAnUnknownFeature(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")
	shipping(p, "0099").fails(2).errHas("no feature")
}

func TestShipSaysTheWorkIsHandedOffNotFinished(t *testing.T) {
	t.Parallel()
	// The first real project to run the loop reached the end of ship, found no
	// instruction about what happens next, and merged its own work to main. The
	// protocol is read at the start of a session; this decision is made at the end
	// of one, so it has to be said here too.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")

	// No remote: there is nowhere to hand off to, which is the case that produced
	// the unreviewed merge — so it must say so rather than staying silent.
	shipping(p, "0007").ok().has("no remote", "merging is not yours")

	p.git("remote", "add", "origin", p.path(".git"))
	feature(p, "0008", "next", "Next", "Draft", "- [x] One")
	shipping(p, "0008").ok().has("push the branch and stop")
}

func TestShipPointsAtTheAsBuiltDoc(t *testing.T) {
	t.Parallel()
	// The protocol says move durable facts across; nothing enforced it, so the prompt
	// is the enforcement.
	// The creation case is TestShipCreatesTheAsBuiltDocOnce; this is the reminder
	// when the doc is already there.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")
	p.write("docs/TECH.md", "# As built\n")

	r := shipping(p, "0007").ok()
	r.has("move durable facts", "docs/TECH.md")
	r.lacks("created docs/TECH.md")
	// And the other two destinations, since this is where the routing is decided: a
	// rule stated only in the protocol is read at the start of a session, not here.
	r.has("only what changed the plan", "the PR")
}

func TestShipCreatesTheAsBuiltDocOnce(t *testing.T) {
	t.Parallel()
	// The protocol says move durable facts into the as-built doc. Until now nothing said
	// what that doc looks like, and shipping is exactly when it first has to exist.
	p := setUp(t)
	feature(p, "0007", "first", "First", "Draft", "- [x] Done")

	shipping(p, "0007").ok().has("created", "docs/TECH.md")
	if !p.exists("docs/TECH.md") {
		t.Fatal("shipping did not create the as-built doc")
	}
	// It answers no-tech, which is the finding it exists to resolve.
	p.run("audit").ok().lacks("no-tech")

	// A second ship must leave what has been written into it alone. Note this is
	// enforced by the caller finding the doc already present, not by the create —
	// once it exists, createTechDoc is never reached.
	p.write("docs/TECH.md", "# TECH\n\nReal content someone wrote.\n")
	feature(p, "0008", "second", "Second", "Draft", "- [x] Done")
	shipping(p, "0008").ok()
	if got := p.read("docs/TECH.md"); !strings.Contains(got, "Real content someone wrote.") {
		t.Errorf("shipping overwrote the as-built doc:\n%s", got)
	}
}
