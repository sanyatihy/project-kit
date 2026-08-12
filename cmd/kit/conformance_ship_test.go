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

func TestShipFillsAnExistingEmptyDateKey(t *testing.T) {
	// The template ships a bare "shipped:" line, so this is the common path — and
	// it is a different branch from inserting one.
	p := setUp(t)
	p.write("docs/features/0007-templated.md",
		"---\nid: 0007\ntitle: Templated\nstatus: Accepted\ncreated: 2026-01-01\nshipped:\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	p.run("ship", "0007").ok()

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
	// The creation case is TestShipCreatesTheAsBuiltDocOnce; this is the reminder
	// when the doc is already there.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Accepted", "- [x] One")
	p.write("docs/TECH.md", "# As built\n")

	r := p.run("ship", "0007").ok()
	r.has("move durable facts", "docs/TECH.md")
	r.lacks("created docs/TECH.md")
}

func TestShipCreatesTheAsBuiltDocOnce(t *testing.T) {
	// Rule 10 says move durable facts into the as-built doc. Until now nothing said
	// what that doc looks like, and shipping is exactly when it first has to exist.
	p := setUp(t)
	feature(p, "0007", "first", "First", "Accepted", "- [x] Done")

	p.run("ship", "0007").ok().has("created", "docs/TECH.md")
	if !p.exists("docs/TECH.md") {
		t.Fatal("shipping did not create the as-built doc")
	}
	// It answers no-tech, which is the finding it exists to resolve.
	p.run("audit").ok().lacks("no-tech")

	// A second ship must not overwrite whatever has been written into it.
	p.write("docs/TECH.md", "# TECH\n\nReal content someone wrote.\n")
	feature(p, "0008", "second", "Second", "Accepted", "- [x] Done")
	p.run("ship", "0008").ok()
	if got := p.read("docs/TECH.md"); !strings.Contains(got, "Real content someone wrote.") {
		t.Errorf("shipping overwrote the as-built doc:\n%s", got)
	}
}

func TestShipRespectsAnExistingAsBuiltDocUnderAnotherName(t *testing.T) {
	// ARCHITECTURE.md is an accepted name. Creating docs/TECH.md alongside it would
	// give the project two as-built docs and no way to tell which is true.
	p := setUp(t)
	feature(p, "0007", "first", "First", "Accepted", "- [x] Done")
	p.write("docs/ARCHITECTURE.md", "# Architecture\n\nAlready here.\n")

	p.run("ship", "0007").ok().has("docs/ARCHITECTURE.md")
	if p.exists("docs/TECH.md") {
		t.Error("shipping created a second as-built doc beside the existing one")
	}
}
