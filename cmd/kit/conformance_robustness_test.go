package main

// Conformance: the inputs and invocations that break tools. Docs arrive with BOMs
// and CRLFs from Windows editors, half-written from an interrupted agent, and the
// commands get typed in whatever order feels natural.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sanyatihy/project-kit/internal/kit"
)

func TestParsingSurvivesBOMAndCRLF(t *testing.T) {
	t.Parallel()
	// A doc saved by a Windows editor must behave exactly like one that was not.
	p := setUp(t)
	body := "---\nid: 0001\ntitle: Windows\nstatus: Accepted\ncreated: 2026-01-01\n---\n\n" +
		"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [ ] The only task\n\n## Log\n- 2026-01-01 — x.\n"
	p.write("docs/features/0001-windows.md", "\uFEFF"+strings.ReplaceAll(body, "\n", "\r\n"))

	p.run("next").ok().has("start here", "The only task")
	p.run("audit").ok().has("docs and repo agree")
}

func TestADocWithNoFrontmatterIsReportedNotFatal(t *testing.T) {
	t.Parallel()
	// An interrupted agent leaves these behind. The tool has to keep working and
	// say what is wrong.
	p := setUp(t)
	p.write("docs/features/0001-partial.md", "# Just a heading\n\nNo frontmatter at all.\n")

	p.run("next").ok()
	p.run("audit").ok().has("no-status")
}

func TestAnUnparseableDateIsTreatedAsAbsent(t *testing.T) {
	t.Parallel()
	// Rather than crashing, or worse, being read as some other date.
	p := setUp(t)
	p.write("docs/features/0001-odd.md",
		"---\nid: 0001\ntitle: Odd\nstatus: Shipped\nshipped: sometime last spring\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	p.run("audit").ok().has("shipped-no-date")
}

func TestExamplesInFencedBlocksAreNotRealTasks(t *testing.T) {
	t.Parallel()
	// The template and the conventions doc both show `- [ ]` inside fences. Counting
	// those would have every project start life with imaginary work.
	// The example has to sit inside ## Tasks to be a real test: OpenTasks only ever
	// reads that section, so a fence anywhere else proves nothing.
	p := setUp(t)
	p.write("docs/features/0001-fenced.md",
		"---\nid: 0001\ntitle: Fenced\nstatus: Accepted\ncreated: 2026-01-01\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n"+
			"## Tasks\n```markdown\n- [ ] An illustrative example\n```\n- [ ] The real task\n\n"+
			"## Log\n- 2026-01-01 — x.\n")

	r := p.run("next").ok()
	r.has("The real task")
	r.lacks("An illustrative example")
}

func TestASectionHoldingOnlyACommentCountsAsEmpty(t *testing.T) {
	t.Parallel()
	// Template guidance lives in HTML comments. If those counted as content, a doc
	// nobody had written would look complete and nothing would ever fire.
	p := setUp(t)
	p.write("docs/features/0001-hollow.md",
		"---\nid: 0001\ntitle: Hollow\nstatus: Accepted\ncreated: 2026-01-01\n---\n\n"+
			"## Problem\nReal enough.\n\n"+
			"## Design\n<!-- describe how it works -->\n\n"+
			"## Tasks\n- [ ] Something\n\n## Log\n- 2026-01-01 — x.\n")

	p.run("audit").ok().has("unfilled-template", "Design")
}

func TestNewDocsAreDraftsAndNotOffered(t *testing.T) {
	t.Parallel()
	// Draft is where a human settles the design, so `next` never hands one out.
	p := setUp(t)
	p.run("new", "Untouched").ok()
	p.run("next").ok().lacks("start here")
}

func TestCommandsWorkFromASubdirectory(t *testing.T) {
	t.Parallel()
	// Documented behaviour: with no path they use the repo you are in, from
	// anywhere inside it.
	p := setUp(t)
	feature(p, "0001", "thing", "A thing", "Accepted", "- [ ] Do the thing")
	if err := os.MkdirAll(p.path("internal/deep/nested"), 0o755); err != nil {
		t.Fatalf("making a subdirectory: %v", err)
	}

	sub := &project{t: t, Root: p.path("internal/deep/nested")}
	sub.run("next").ok().has("Do the thing")
	sub.run("audit").ok().has("docs and repo agree")
}

func TestFlagsAreAcceptedAfterPaths(t *testing.T) {
	t.Parallel()
	// Go's flag package stops at the first positional, so `kit audit PATH --json`
	// would silently ignore the flag without parseArgs looping.
	p := setUp(t)
	feature(p, "0001", "thing", "A thing", "Accepted", "- [ ] Do the thing")

	after := p.run("audit", p.Root, "--json").ok()
	if !strings.HasPrefix(strings.TrimSpace(after.Stdout), "{") {
		t.Errorf("--json after a path was ignored:\n%s", after.Stdout)
	}
	p.run("next", p.Root, "--json").ok()
}

func TestJSONIsValidAndCarriesThePlan(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	feature(p, "0001", "thing", "A thing", "Accepted", "- [ ] Do the thing")

	var plans map[string]kit.Plan
	raw := p.run("next", "--json").ok().Stdout
	if err := json.Unmarshal([]byte(raw), &plans); err != nil {
		t.Fatalf("next --json is not valid JSON: %v\n%s", err, raw)
	}
	for _, plan := range plans {
		if len(plan.Ready) != 1 {
			t.Fatalf("expected one ready item, got %d", len(plan.Ready))
		}
		item := plan.Ready[0]
		if item.ID != "0001" || item.Task != "Do the thing" {
			t.Errorf("json item is wrong: %+v", item)
		}
		if item.Branch == "" || item.Doc == "" {
			t.Errorf("json item lacks the branch or doc an agent needs: %+v", item)
		}
	}
}

func TestJSONStaysCleanOnStdoutWhenThereIsNothingToReport(t *testing.T) {
	t.Parallel()
	// A consumer piping --json must get parseable output even in the empty case,
	// rather than a friendly sentence.
	p := setUp(t)
	var findings map[string]audited
	raw := p.run("audit", "--json").ok().Stdout
	if err := json.Unmarshal([]byte(raw), &findings); err != nil {
		t.Fatalf("audit --json is not valid JSON when clean: %v\n%s", err, raw)
	}
}

func TestNonGitDirectoriesStillWork(t *testing.T) {
	t.Parallel()
	// git powers claims and staleness, but its absence must degrade rather than
	// break — plenty of directories are not repositories yet.
	p := setUp(t)
	feature(p, "0001", "thing", "A thing", "Accepted", "- [ ] Do the thing")
	if err := os.RemoveAll(p.path(".git")); err != nil {
		t.Fatalf("removing .git: %v", err)
	}

	p.run("next").ok().has("Do the thing")
	p.run("audit").ok()
}

func TestExitCodesAreDistinct(t *testing.T) {
	t.Parallel()
	// An agent scripting the kit has to tell "you typed it wrong" from "the audit
	// found something".
	p := cleanProject(t)

	p.run("version").ok()
	p.run("help").ok()
	p.run("audit").ok()

	// 2 is usage: an unusable invocation, nothing to do with the docs.
	p.run("nonesuch").fails(2)
	p.run("audit", "/no/such/directory").fails(2)
	p.run("init", "--stack", "nonesuch").fails(2)

	// 1 is reserved for --strict finding something HIGH.
	p.write("docs/features/0002-early.md", doc("0002", "Early", "Shipped",
		"shipped: "+daysAgo(1), "- [ ] Not actually done"))
	p.run("audit", "--strict").fails(1)
}

func TestEmptyProjectDirectoriesAreHandled(t *testing.T) {
	t.Parallel()
	// An empty features directory is not the same as a broken project.
	p := setUp(t)
	entries, err := os.ReadDir(p.path("docs/features"))
	if err != nil {
		t.Fatalf("reading the features dir: %v", err)
	}
	for _, e := range entries {
		if err := os.Remove(p.path("docs/features/" + e.Name())); err != nil {
			t.Fatalf("emptying the features dir: %v", err)
		}
	}

	p.run("next").ok().has("nothing is planned yet")
	p.run("audit").ok()
}

func TestARemoteThatDoesNotAnswerIsSaidRatherThanShownAsNothing(t *testing.T) {
	t.Parallel()
	// The failure this feature is for: an unreachable remote and a project with nothing
	// waiting produced the same page. The remote here refuses the connection instead of
	// timing out, which reaches the same code by the same route and does not cost the
	// suite fifteen seconds.
	p := cleanProject(t)
	p.commitAll("the plan")
	p.git("remote", "add", "origin", "ssh://127.0.0.1:1/nowhere.git")

	p.run("next").ok().has("the remote did not answer")
	if !p.plan("next", "--json").RemoteUnanswered {
		t.Errorf("--json does not carry remote_unanswered, so an agent reads a board " +
			"that looks complete")
	}
	// The check that could not run at all is named, and the ones that ran on a stale
	// mirror are qualified. A clean bill here would be the worst of the three outputs.
	p.run("audit").ok().has("the remote did not answer", "shipped-not-pushed")

	// --offline is the reader's own instruction, not a failure, so it stays quiet.
	for _, args := range [][]string{{"next", "--offline"}, {"audit", "--offline"}} {
		r := p.run(args...).ok()
		if strings.Contains(r.Stdout, "did not answer") {
			t.Errorf("kit %s reported a failure the reader asked for%s",
				strings.Join(args, " "), r.context())
		}
	}
}
