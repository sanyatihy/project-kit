package main

// Conformance: `kit ship`. Its value is what it refuses — two of the audit's
// findings exist only because this is normally done by hand.

import (
	"strings"
	"testing"
	"time"
)

func TestShipRefusesWhileABoxIsOpen(t *testing.T) {
	// unticked-shipped, prevented rather than reported.
	p := setUp(t)
	feature(p, "0007", "half", "Half done", "In progress", "- [x] One\n- [ ] Two")

	r := p.run("ship", "0007").fails(2)
	r.errHas("unticked", "Two")
	if strings.Contains(p.read("docs/features/0007-half.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipStampsTheDate(t *testing.T) {
	// shipped-no-date, prevented rather than reported.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "In progress", "- [x] One\n- [x] Two")

	p.run("ship", "0007").ok().has("shipped")

	doc := p.read("docs/features/0007-done.md")
	if !strings.Contains(doc, "status: Shipped") {
		t.Errorf("status was not set:\n%s", doc)
	}
	if want := "shipped: " + time.Now().UTC().Format("2006-01-02"); !strings.Contains(doc, want) {
		t.Errorf("date was not stamped, want %q:\n%s", want, doc)
	}
	p.run("audit").ok().lacks("shipped-no-date", "unticked-shipped")
}

func TestShipAddsTheDateKeyWhenTheDocHasNone(t *testing.T) {
	// An older doc may have no shipped: line at all. Stamping has to insert one
	// rather than silently do nothing.
	p := setUp(t)
	p.write("docs/features/0007-old.md",
		"---\nid: 0007\ntitle: Old\nstatus: Accepted\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	p.run("ship", "0007").ok()
	if !strings.Contains(p.read("docs/features/0007-old.md"), "shipped: ") {
		t.Error("no shipped: key was added")
	}
}

func TestShipRefusesAFeatureWithNoBoxesAtAll(t *testing.T) {
	// Nothing to have finished. Shipping it would assert work that was never planned.
	p := setUp(t)
	feature(p, "0007", "empty", "Empty", "Accepted", "")
	p.run("ship", "0007").fails(2).errHas("no ## Tasks boxes")
}

func TestShipRefusesTwice(t *testing.T) {
	// The second run would rewrite frozen history and restamp the date.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Accepted", "- [x] One")
	p.run("ship", "0007").ok()
	p.run("ship", "0007").fails(2).errHas("already Shipped")
}

func TestShipRefusesAnUnknownFeature(t *testing.T) {
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Accepted", "- [x] One")
	p.run("ship", "0099").fails(2).errHas("no feature")
}

func TestShipPointsAtTheAsBuiltDoc(t *testing.T) {
	// Rule 10 says move durable facts across; nothing enforced it, so the prompt is
	// the enforcement.
	withTech := setUp(t)
	feature(withTech, "0007", "done", "Done", "Accepted", "- [x] One")
	withTech.write("docs/TECH.md", "# As built\n")
	withTech.run("ship", "0007").ok().has("docs/TECH.md")

	without := setUp(t)
	feature(without, "0007", "done", "Done", "Accepted", "- [x] One")
	without.run("ship", "0007").ok().has("add docs/TECH.md")
}
