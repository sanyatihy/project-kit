package main

// Conformance: `kit block`. Its whole job is to write a box the tool can parse into the
// doc it blocks, so the round-trip against `next` and `audit` is the real test.

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

func TestBlockWritesABoxTheParserAccepts(t *testing.T) {
	t.Parallel()
	// The round-trip is the point: whatever `block` writes, `next` must read as a
	// blocker on that feature and `audit` must not call it informal.
	p := blockable(t)
	p.run("block", "0007", "Should codes expire? Leaning: no, keep forever.").ok()

	p.run("next").ok().has("blocked", "Should codes expire?")
	p.run("audit").ok().lacks("question-informal")
}

func TestBlockWritesIntoTheFeatureDocAndNowhereElse(t *testing.T) {
	t.Parallel()
	// The doc is the whole record. A question filed anywhere else is one more place to
	// keep in step with it.
	p := blockable(t)
	p.run("block", "7", "Which encoding?").ok()

	got := p.read("docs/features/0007-shorten.md")
	if !strings.Contains(got, "## Open questions") || !strings.Contains(got, "Which encoding?") {
		t.Errorf("the question is not in the doc it blocks:\n%s", got)
	}
	// Nobody types leading zeros reliably, and refusing them would push people back to
	// editing the doc by hand.
	if !strings.Contains(got, "- [ ] ") {
		t.Errorf("the short id did not reach the canonical doc:\n%s", got)
	}
}

func TestBlockOpensTheSectionAboveTheLog(t *testing.T) {
	t.Parallel()
	// The template says to delete the heading when it is empty, so a doc without one is
	// the normal case. Opening it in the template's position keeps every doc reading in
	// the same order.
	p := blockable(t)
	before := p.read("docs/features/0007-shorten.md")
	if strings.Contains(before, "## Open questions") {
		t.Fatalf("the fixture already has the section, so this proves nothing:\n%s", before)
	}
	p.run("block", "0007", "Which encoding?").ok()

	got := p.read("docs/features/0007-shorten.md")
	questions, log := strings.Index(got, "## Open questions"), strings.Index(got, "## Log")
	if questions == -1 || log == -1 || questions > log {
		t.Errorf("the section did not open above ## Log:\n%s", got)
	}
}

func TestBlockCommitsTheQuestion(t *testing.T) {
	t.Parallel()
	// A question left in the working tree dirties the checkout, and `kit start` refuses
	// a dirty tree — so recording one would cost the agent its next claim.
	p := blockable(t)
	p.commitAll("the feature")
	p.run("block", "0007", "Which encoding?").ok()

	if got := p.gitOut(nil, "status", "--porcelain"); got != "" {
		t.Errorf("block left the question in the tree:\n%s", got)
	}
	if got := p.gitOut(nil, "log", "-1", "--pretty=%s"); !strings.Contains(got, "0007") {
		t.Errorf("the commit subject does not name the feature: %q", got)
	}
}

func TestBlockKeepsTheRestOfTheDoc(t *testing.T) {
	t.Parallel()
	p := blockable(t)
	p.run("block", "0007", "A question.").ok()
	p.run("block", "0007", "Another question.").ok()

	got := p.read("docs/features/0007-shorten.md")
	for _, want := range []string{"The POST handler", "## Design", "A question.", "Another question."} {
		if !strings.Contains(got, want) {
			t.Errorf("block rewrote the doc instead of appending to it — %q is gone:\n%s", want, got)
		}
	}
}

func TestBlockNoLongerTakesTheFYIFlag(t *testing.T) {
	t.Parallel()
	// Every open question blocks now. A flag that once meant "do not block" must refuse
	// rather than be accepted and ignored, or it reads as still meaning it.
	p := blockable(t)
	p.run("block", "0007", "Using base62 for slugs.", "--fyi").fails(2)
}

func TestBlockRefusesAFeatureThatDoesNotExist(t *testing.T) {
	t.Parallel()
	// There is no doc to write the question into, and nothing downstream would notice it.
	p := blockable(t)
	r := p.run("block", "0099", "anything at all").fails(2)
	r.errHas("no feature")
	// The message has to say what does exist, or the user is guessing.
	r.errHas("0007")
}

func TestBlockRefusesADocThatDeclaresAKeyTwice(t *testing.T) {
	t.Parallel()
	// The question is written into the doc, so this is the refusal `submit` and `drop`
	// make: a doc whose frontmatter says two things about itself is not one to write to.
	p := setUp(t)
	p.write("docs/features/0007-twice.md",
		"---\nid: 0007\nid: 0008\ntitle: Twice\nstatus: Draft\n"+
			"created: 2026-01-01\n---\n\n## Problem\nx\n\n## Design\nx\n\n"+
			"## Tasks\n- [ ] Do it\n\n## Log\n- 2026-01-01 — started.\n")

	p.run("block", "0008", "Which one is it?").fails(2).errHas("id", "twice")
	if got := p.read("docs/features/0007-twice.md"); strings.Contains(got, "Which one is it?") {
		t.Errorf("block wrote the question despite refusing:\n%s", got)
	}
}

func TestBlockRefusesAnIDThatNamesTwoDocs(t *testing.T) {
	t.Parallel()
	// Which doc the question belongs in is the author's to say, not the kit's to guess.
	// That is the same ambiguity `claim` refuses.
	p := setUp(t)
	p.write("docs/features/0007-one.md", doc("0007", "One", "Draft", "", "- [ ] Something"))
	p.write("docs/features/0007-two.md", doc("0007", "Two", "Draft", "", "- [ ] Something"))

	p.run("block", "0007", "Which one is it?").fails(2).errHas("0007-one.md", "0007-two.md")
	for _, rel := range []string{"docs/features/0007-one.md", "docs/features/0007-two.md"} {
		if got := p.read(rel); strings.Contains(got, "Which one is it?") {
			t.Errorf("block wrote the question despite refusing, in %s:\n%s", rel, got)
		}
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
	// One box per line is what makes the section parseable. A pasted multi-line question
	// would otherwise split into a tail that blocks nothing.
	p := blockable(t)
	p.run("block", "0007", "First part\nsecond part\n\nthird part").ok()

	p.run("audit").ok().lacks("question-informal")
	for _, line := range strings.Split(p.read("docs/features/0007-shorten.md"), "\n") {
		if strings.Contains(line, "second part") && !strings.Contains(line, "- [ ]") {
			t.Errorf("the question spilled onto its own line: %q", line)
		}
	}
}
