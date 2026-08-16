package main

// Every text the binary ships has a ceiling, because nothing else stops one growing.
// The rule the kit states is "write as much as necessary and no more", and the half
// that rots silently is the second: a paragraph is always easier to add than to
// justify, and no reviewer counts lines.
//
// These are ceilings on the kit's own text, not targets for anyone's docs. A doc too
// thin to act on costs the next session more than a long one, so nothing here has a
// lower bound — under-writing is caught by unfilled-template and by review, and a
// minimum-length check would reward padding.

import (
	"io/fs"
	"strings"
	"testing"

	assets "github.com/sanyatihy/project-kit"
)

const (
	// The protocol is written into every project and re-read at the start of every
	// session, so its length is a tax paid per session per project — the only text
	// here where growth compounds. It earns the tightest number.
	protocolBudget = 145

	// A command help is a usage reference someone reads mid-task, often after a
	// mistake. Anything approaching an essay belongs in a topic.
	commandHelpBudget = 45

	// Topics are read on demand and once. `go-checklist` and `deployment` are standing
	// engineering guidance whose substance is the point, so this only catches
	// unbounded growth rather than enforcing brevity.
	topicBudget = 220
)

func lines(text string) int { return len(strings.Split(strings.TrimRight(text, "\n"), "\n")) }

// overBudget is the failure message. It names the rule rather than only the number,
// because a bare count teaches "make it shorter", which is the wrong half.
func overBudget(t *testing.T, what, text string, budget int) {
	t.Helper()
	if n := lines(text); n > budget {
		t.Errorf("%s is %d lines, over its %d-line ceiling.\n"+
			"Write as much as necessary and no more: cut restatement, hedging and "+
			"padding — not a decision, a constraint, or the evidence behind it. If "+
			"every line earns its place, raise the ceiling in this test and say why.",
			what, n, budget)
	}
}

func TestTheProtocolStaysUnderItsCeiling(t *testing.T) {
	raw, err := fs.ReadFile(assets.FS, "templates/AGENTS-section.md")
	if err != nil {
		t.Fatalf("reading the protocol: %v", err)
	}
	overBudget(t, "templates/AGENTS-section.md", string(raw), protocolBudget)
}

func TestEveryCommandHelpStaysUnderItsCeiling(t *testing.T) {
	for name, text := range commandHelp {
		overBudget(t, "commandHelp["+name+"]", text, commandHelpBudget)
	}
	overBudget(t, "the overview", overview, commandHelpBudget+40)
}

func TestEveryTopicStaysUnderItsCeiling(t *testing.T) {
	for name, path := range topics {
		raw, err := fs.ReadFile(assets.FS, path)
		if err != nil {
			t.Errorf("reading topic %q (%s): %v", name, path, err)
			continue
		}
		overBudget(t, "topic "+name+" ("+path+")", string(raw), topicBudget)
	}
}
