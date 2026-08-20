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
	// 0041 raised it to 121 for two rules — what earns a feature doc, and write about the
	// system rather than the session. Both delete a class of writing from every doc after
	// them, which is the trade that buys protocol growth; a rule that only adds something
	// to remember does not.
	//
	// 0044 raised it to 123 for the routing rule — every fact has one home, chosen as the
	// sentence is written. It deletes the transcript of the work from every doc after it,
	// which "write about the system" alone did not: a run that taught something read as
	// durable and landed in the ## Log. The evidence rule folded into the bullet above it
	// rather than adding a third.
	//
	// 0045 added the sentence rule — a sentence earns its place by deciding, constraining or
	// measuring — and the ceiling held. It is the first rule here to arrive without a raise,
	// which is the test it was written to pass: the evidence clause moved out of "write about
	// the system" into it, and rule 4 now cites the routing rule rather than restating it. A
	// raise would have been the signal that the rule was written as an addition.
	protocolBudget = 123

	// A command help is a usage reference someone reads mid-task, often after a
	// mistake. Anything approaching an essay belongs in a topic.
	commandHelpBudget = 45

	// Topics are read on demand and once. `go-checklist` and `deployment` are standing
	// engineering guidance whose substance is the point, so this only catches
	// unbounded growth rather than enforcing brevity.
	topicBudget = 220

	// The two documents this repo itself reads every session: the protocol above, and the
	// as-built doc before any change to the kit. 0031 cut TECH from 546 lines to 247 and
	// found two sections answering the same question and three facts stated twice — growth
	// nobody notices, because a paragraph is easier to add than to justify. Same escape
	// hatch as the ceilings above: raise the number here and say why.
	//
	// 0044 retook it at 295 against a 289-line file, rewritten to state the rule and give
	// the evidence a trailing clause. The headroom is deliberate and small: under the
	// routing rule a shipped feature adds one line, so a jump means facts arrived with
	// their history attached again.
	techBudget   = 295
	agentsBudget = 210
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
