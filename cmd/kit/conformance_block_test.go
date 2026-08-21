package main

// Conformance: `kit block`. Its whole job is to write a line the tool can parse,
// so the round-trip against `next` and `audit` is the real test.

import (
	"strings"
	"testing"
)

func blockable(t *testing.T) *project {
	t.Helper()
	p := setUp(t)
	feature(p, "0007", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")
	return p
}

func TestBlockWritesALineTheParserAccepts(t *testing.T) {
	t.Parallel()
	// The round-trip is the point: whatever `block` writes, `next` must read as a
	// blocker on that feature and `audit` must not call it unparsed.
	p := blockable(t)
	p.run("block", "0007", "Should codes expire? Leaning: no, keep forever.").ok()

	p.run("next").ok().has("blocked", "Should codes expire?")
	p.run("audit").ok().lacks("inbox-unparsed")
}

func TestBlockAcceptsAShortID(t *testing.T) {
	t.Parallel()
	// Nobody types leading zeros reliably, and refusing them would push people back
	// to editing the file by hand.
	p := blockable(t)
	p.run("block", "7", "Which encoding?").ok()

	if got := p.read("docs/INBOX.md"); !strings.Contains(got, "· 0007 ·") {
		t.Errorf("the short id was not normalised to the canonical one:\n%s", got)
	}
}

func TestBlockFYIRecordsWithoutBlocking(t *testing.T) {
	t.Parallel()
	p := blockable(t)
	p.run("block", "0007", "Using base62 for slugs.", "--fyi").ok()

	r := p.run("next").ok()
	r.has("start here", "The POST handler")
	r.lacks("blocked")
	if got := p.read("docs/INBOX.md"); !strings.Contains(got, "· FYI ·") {
		t.Errorf("--fyi did not write the FYI marker:\n%s", got)
	}
}

func TestBlockRefusesAFeatureThatDoesNotExist(t *testing.T) {
	t.Parallel()
	// A question against a nonexistent feature can never unblock anything, and
	// nothing downstream would notice it.
	p := blockable(t)
	r := p.run("block", "0099", "anything at all").fails(2)
	r.errHas("no feature")
	// The message has to say what does exist, or the user is guessing.
	r.errHas("0007")
}

func TestBlockRefusesADocThatDeclaresAKeyTwice(t *testing.T) {
	t.Parallel()
	// A question blocks by id, and a doc that declares its id twice does not have one the
	// kit and its author agree on — so the line would block a feature nobody is reading.
	p := setUp(t)
	p.write("docs/features/0007-twice.md",
		"---\nid: 0007\nid: 0008\ntitle: Twice\nstatus: Draft\n"+
			"created: 2026-01-01\n---\n\n## Problem\nx\n\n## Design\nx\n\n"+
			"## Tasks\n- [ ] Do it\n\n## Log\n- 2026-01-01 — started.\n")

	p.run("block", "0008", "Which one is it?").fails(2).errHas("id", "twice")
	if p.exists("docs/INBOX.md") {
		t.Errorf("block created the inbox despite refusing:\n%s", p.read("docs/INBOX.md"))
	}
}

func TestBlockRefusesAnIDThatNamesTwoDocs(t *testing.T) {
	t.Parallel()
	// One inbox line blocks by id, so against two docs it blocks both — and clearing it
	// clears both. That is the same ambiguity `claim` refuses.
	p := setUp(t)
	p.write("docs/features/0007-one.md", doc("0007", "One", "Draft", "", "- [ ] Something"))
	p.write("docs/features/0007-two.md", doc("0007", "Two", "Draft", "", "- [ ] Something"))

	p.run("block", "0007", "Which one is it?").fails(2).errHas("0007-one.md", "0007-two.md")
	if p.exists("docs/INBOX.md") {
		t.Errorf("block created the inbox despite refusing:\n%s", p.read("docs/INBOX.md"))
	}
}

func TestBlockNeedsAnIDAndAQuestion(t *testing.T) {
	t.Parallel()
	p := blockable(t)
	p.run("block").fails(2)
	p.run("block", "0007").fails(2)
	p.run("block", "0007", "   ").fails(2)
}

func TestBlockKeepsTheQuestionOnOneLine(t *testing.T) {
	t.Parallel()
	// One item per line is what makes the file parseable. A pasted multi-line
	// question would otherwise split into an unparseable tail.
	p := blockable(t)
	p.run("block", "0007", "First part\nsecond part\n\nthird part").ok()

	p.run("audit").ok().lacks("inbox-unparsed")
	for _, line := range strings.Split(p.read("docs/INBOX.md"), "\n") {
		if strings.Contains(line, "second part") && !strings.Contains(line, "- [ ]") {
			t.Errorf("the question spilled onto its own line: %q", line)
		}
	}
}

func TestBlockPreservesWhatIsAlreadyInTheInbox(t *testing.T) {
	t.Parallel()
	p := blockable(t)
	p.run("block", "0007", "The first question.").ok()
	before := p.read("docs/INBOX.md")
	p.run("block", "0007", "A question.").ok()

	after := p.read("docs/INBOX.md")
	if !strings.HasPrefix(after, strings.TrimRight(before, "\n")) {
		t.Error("block rewrote the inbox instead of appending to it")
	}
}

func TestBlockCreatesTheInboxWithTheFirstQuestion(t *testing.T) {
	t.Parallel()
	// The inbox is not scaffolded: a file written before anything can go in it is one
	// the project maintains for the tool. `kit ship` creates the as-built doc the same
	// way, and neither is ever reported missing.
	p := blockable(t)
	if p.exists("docs/INBOX.md") {
		t.Fatal("kit init scaffolded an inbox, so this proves nothing")
	}
	p.run("audit").ok().lacks("no-protocol")

	p.run("block", "0007", "A question.").ok().has("docs/INBOX.md", "created")

	// And what it wrote is the documented format, not a bare line: the file has to
	// explain itself to whoever drains it.
	got := p.read("docs/INBOX.md")
	if !strings.Contains(got, "BLOCKING") || !strings.Contains(got, "A question.") {
		t.Errorf("the inbox block created does not carry the format and the question:\n%s", got)
	}
	p.run("next").ok().has("blocked", "A question.")
	p.run("audit").ok().lacks("inbox-unparsed")
}

func TestBlockWritesIntoAnInboxTheProjectAlreadyHas(t *testing.T) {
	t.Parallel()
	// Creating it must never mean replacing it. A project that adopted the kit with an
	// inbox of its own keeps every line in it.
	p := blockable(t)
	p.write("docs/INBOX.md", "# Our inbox\n\n- [ ] 2026-01-02 · 0007 · FYI · ours\n")

	p.run("block", "0007", "A question.").ok().lacks("created")

	got := p.read("docs/INBOX.md")
	if !strings.Contains(got, "# Our inbox") || !strings.Contains(got, "· FYI · ours") {
		t.Errorf("block replaced the project's own inbox:\n%s", got)
	}
}
