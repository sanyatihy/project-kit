package main

// Every text the binary ships has a ceiling, because nothing else stops one growing.
// The rule the kit states is that a sentence earns its place by deciding, constraining
// or measuring something, and it rots silently: a paragraph is always easier to add than
// to justify, and no reviewer counts lines.
//
// These are ceilings on the kit's own text, not targets for anyone's docs. A doc too
// thin to act on costs the next session more than a long one, so nothing here has a
// lower bound — under-writing is caught by unfilled-template and by review, and a
// minimum-length check would reward padding.

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	assets "github.com/sanyatihy/project-kit"
)

const (
	// The protocol is written into every project and re-read at the start of every
	// session, so its length is a tax paid per session per project — the only text
	// here where growth compounds. It earns the tightest number.
	//
	// 0031 cut it from 155 to 115 without dropping a rule: the reasoning behind the rules
	// moved to docs/TECH.md, which is read once rather than every session, and what stayed
	// is the constraint and its consequence. A rule an agent must obey in a hurry has to
	// survive being read in isolation, which is what the sub-bullets are for.
	//
	// 0048 rewrote it as 26 numbered rules under four headings and it came out at 106,
	// so the ceiling comes down with it. A ceiling left above what the text needs is not
	// a ceiling: it is headroom nobody has to justify spending.
	// 111 since the feature rule was reframed: naming the value a feature ships, and the
	// three ways to tell it is not one, costs a line over asserting that it records a
	// decision. Decomposition is what the protocol is for.
	protocolBudget = 111

	// A command help is a usage reference someone reads mid-task, often after a
	// mistake. Anything approaching an essay belongs in a topic.
	commandHelpBudget = 45

	// Topics are read on demand and once. `go-checklist` is standing engineering
	// guidance whose substance is the point, so this only catches unbounded growth
	// rather than enforcing brevity.
	topicBudget = 220

	// The two documents this repo itself reads every session: the protocol above, and the
	// as-built doc before any change to the kit. 0031 cut TECH from 546 lines to 247 and
	// found two sections answering the same question and three facts stated twice — growth
	// nobody notices, because a paragraph is easier to add than to justify. Same escape
	// hatch as the ceilings above: raise the number here and say why.
	//
	// 0048 retook all three against what the cut left: TECH 287, AGENTS 159, README 279.
	// The headroom is deliberate and small — under the routing rule a shipped feature adds
	// one line each, so a jump means facts arrived with their history attached again.
	techBudget = 292
	// 168 since the round that repaired this file: it was spliced at the wrong marker and
	// lost 50 lines, including the only guard against an agent emptying a target directory,
	// so the old number was calibrated against a truncated file.
	agentsBudget = 168

	// README.md is the first thing a person reads and the only one of these with no
	// ceiling until now. It had drifted to 431 lines describing commands the binary
	// already documents itself, which is the growth this catches.
	readmeBudget = 285
)

func lines(text string) int { return len(strings.Split(strings.TrimRight(text, "\n"), "\n")) }

// overBudget is the failure message. It names the rule rather than only the number,
// because a bare count teaches "make it shorter", which is the wrong half.
func overBudget(t *testing.T, what, text string, budget int) {
	t.Helper()
	if n := lines(text); n > budget {
		t.Errorf("%s is %d lines, over its %d-line ceiling.\n"+
			"A sentence earns its place by deciding, constraining or measuring "+
			"something: cut what does none of the three, and cut the second clause "+
			"before the first. If every line earns its place, raise the ceiling in "+
			"this test and say why.",
			what, n, budget)
	}
}

func TestTheProtocolStaysUnderItsCeiling(t *testing.T) {
	t.Parallel()
	raw, err := fs.ReadFile(assets.FS, "templates/AGENTS-section.md")
	if err != nil {
		t.Fatalf("reading the protocol: %v", err)
	}
	overBudget(t, "templates/AGENTS-section.md", string(raw), protocolBudget)
}

// The kit's own docs, held to the same rule as the text it ships. They are not embedded, so
// they are read from disk rather than from assets.FS.
func TestTheKitsOwnDocsStayUnderTheirCeilings(t *testing.T) {
	t.Parallel()
	for _, doc := range []struct {
		path   string
		budget int
	}{
		{"../../docs/TECH.md", techBudget},
		{"../../AGENTS.md", agentsBudget},
		{"../../README.md", readmeBudget},
	} {
		raw, err := os.ReadFile(doc.path)
		if err != nil {
			t.Fatalf("reading %s: %v", doc.path, err)
		}
		overBudget(t, doc.path, string(raw), doc.budget)
	}
}

func TestEveryCommandHelpStaysUnderItsCeiling(t *testing.T) {
	t.Parallel()
	for name, text := range commandHelp {
		overBudget(t, "commandHelp["+name+"]", text, commandHelpBudget)
	}
	overBudget(t, "the overview", overview, commandHelpBudget+40)
}

func TestEveryTopicStaysUnderItsCeiling(t *testing.T) {
	t.Parallel()
	for name, path := range topics {
		raw, err := fs.ReadFile(assets.FS, path)
		if err != nil {
			t.Errorf("reading topic %q (%s): %v", name, path, err)
			continue
		}
		overBudget(t, "topic "+name+" ("+path+")", string(raw), topicBudget)
	}
}
