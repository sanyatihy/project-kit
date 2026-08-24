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
	// 113 since 0054: nothing named the step after a merge, and an install resolves to the
	// newest tag, so every round after the last one reached no project that installed it.
	protocolBudget = 113

	// A command help is a usage reference someone reads mid-task, often after a
	// mistake. Anything approaching an essay belongs in a topic.
	commandHelpBudget = 45

	// Topics are read on demand and once. `go-checklist` is standing engineering
	// guidance whose substance is the point, so this only catches unbounded growth
	// rather than enforcing brevity.
	// 193 since 0060: the inbox and feature-doc topics lost the lines describing checks
	// that no longer exist, and the ceiling is retaken against the longest topic left.
	// Headroom nobody has to justify spending is not a ceiling.
	topicBudget = 193

	// The two documents this repo itself reads every session: the protocol above, and the
	// as-built doc before any change to the kit. 0031 cut TECH from 546 lines to 247 and
	// found two sections answering the same question and three facts stated twice — growth
	// nobody notices, because a paragraph is easier to add than to justify. Same escape
	// hatch as the ceilings above: raise the number here and say why.
	//
	// 0048 retook all three against what the cut left: TECH 287, AGENTS 159, README 279.
	// The headroom is deliberate and small — under the routing rule a shipped feature adds
	// one line each, so a jump means facts arrived with their history attached again.
	// 306 since 0052. Under the routing rule a feature adds one line here, and this one
	// closed seven states where `audit` went quiet rather than reporting — four rules,
	// each of which the next reader has to know before touching a check.
	// 309 on merging 0053, whose three lines state the CI contract this repo now holds
	// itself to: one list of steps, a pinned linter, and no coverage number.
	// 317 on merging 0051: the frontmatter format now carries three constraints an agent
	// can reverse in one line each — where the block closes, where a write may land, and
	// what marks a comment — and a write needs one unambiguous doc to land on.
	// 327 on merging 0056, which states where a claim is read from. Ten lines rather than
	// its own seven: 0050 landed three over the ceiling nobody raised, because a merge
	// result is the one state neither pull request's own gate ever runs against.
	// 339 since 0055. Three guarantees rather than the routing rule's one line: what the
	// binary may name, that a refusal list is whole, and the shape of `--json`. Each is a
	// contract a later change can break silently, so each states what holds and what
	// broke — and the twelfth line is the `flag` gotcha growing a second half.
	// 344 on merging 0054: five lines for the half of shipping nothing here described —
	// what a release is, what refuses one, and the command that cuts it, which belongs
	// beside the other two a developer runs.
	// 352 since 0058. Eight rather than the routing rule's one: an id is now taken over the
	// network, and a reader who does not know where the ref lives, that only a created one
	// counts, what it points at, or what is lost with no remote cannot tell a correct run
	// from a degraded one.
	// 349 on merging 0060, the first round to take lines out: thirteen checks left the
	// vocabulary, so the catalogue of what was deleted and why compresses into the rule it
	// was evidence for, and the `linkRe` gotcha went with the parser it was about. The
	// number falls under 0058's rather than past it, because that round's eight lines stand.
	// 360 on merging 0059: the board and the commands answering one question from one
	// place, what a claim carries out of the checkout it is made from, and the Go gotcha
	// under a bound that could not be enforced. Eleven lines for four states an agent was
	// sent into, each of which a later change can restore without any output looking wrong.
	// 368 on 0062. Eight lines for two refusals nothing else records: one checkout taking
	// one claim, and a draft never taking another. Each names the state it closes — two
	// claims pushed from one checkout at exit 0, a plan branch holding a feature while
	// `audit` agreed — because both read as correct output until you know what to look for.
	// Still 368 on merging 0061: that round rewrote four facts and added one — that there
	// is one kind of work — inside the lines the rewrites gave back, so the file did not
	// shorten. A deletion round does not always shorten this one.
	// 381 on 0061. Thirteen lines for what a new project receives and where a file comes
	// from: one command writes the layout, a `.gitignore` is a refusal in file form, and a
	// test holds the protocol's own list against it. The reversal is the fact worth the
	// room — splitting creation across three commands was tried, and what it bought was a
	// check gate.
	// 387 on 0063. Six lines for where a claim's worktree goes and who writes the exclude:
	// the location is a trade the next reader would otherwise reverse, and the exclude is in
	// `claim` rather than `init` for a reason — `git clone` carries none — that nothing else
	// records.
	// 395 on 0064. Eight lines for what a reservation now is and what closes it: the ref
	// carries a record, a number no doc holds is released rather than left, and the
	// question is asked of every branch's tree rather than of the log — the reversal is
	// the fact worth the room, because the log answer refuses every id this frees.
	// 399 on 0065. Four lines for why the frontmatter carries no prose and what catches
	// the prose that gets there anyway: two spaces is the whole margin between a comment
	// and a value, which is a constraint a later edit can spend without noticing.
	// 405 on 0066. Six lines for what makes a status durable and what notices when it is
	// not: the commit belongs to the command that wrote the status, and the check behind it
	// is the only one that reads HEAD, which is the fact a reader cannot recover from a
	// registry entry.
	techBudget = 405
	// 168 since the round that repaired this file: it was spliced at the wrong marker and
	// lost 50 lines, including the only guard against an agent emptying a target directory,
	// so the old number was calibrated against a truncated file.
	// 169 on 0054: the protocol's tagging rule arrives here with it, and the line above it
	// names the command that cuts one — the generated section cannot, because the Go stack
	// it scaffolds already ships a `make release` that builds a binary.
	agentsBudget = 169

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
