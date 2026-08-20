package main

// Conformance: `kit accept`. The one transition that belongs to a human, and the only
// one that had no command — so it was done by hand, which is what `kit ship` exists to
// stop doing for the other end of the lifecycle.

import (
	"strings"
	"testing"
)

func TestAcceptSettlesADraftAndNextThenOffersIt(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.run("new", "Shorten URLs").ok()
	path := "docs/features/0001-shorten-urls.md"
	p.write(path, fullDoc("0001", "Shorten URLs", "Draft", "- [ ] Do the thing"))

	// A Draft is already offered — accepting promotes it, it does not unlock it.
	p.run("next").ok().has("start here", "0001")

	r := p.run("accept", "0001").ok()
	r.has("accepted", path)
	if !strings.Contains(p.read(path), "status: Accepted") {
		t.Errorf("accept did not set the status:\n%s", p.read(path))
	}
	// Where the edit lands is the ambiguity this closes: inside the draft's own review,
	// not a second one after it merges. Worded without naming a forge — the same repo
	// is a pull request on one host and a merge request on another.
	if !strings.Contains(r.out(), "same review") {
		t.Errorf("accept did not say where to commit it%s", r.context())
	}

	p.run("next").ok().has("start here", "0001")
}

func TestAcceptTakesAnIDWithoutLeadingZeros(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.write("docs/features/0003-thing.md", fullDoc("0003", "Thing", "Draft", "- [ ] Do it"))
	p.run("accept", "3").ok()
}

func TestAcceptRefusesWhatIsNotSettled(t *testing.T) {
	t.Parallel()
	// Each of these is a doc saying the design is not ready. Accepting over any of them
	// puts an agent to work on something nobody decided — the exact thing the Draft gate
	// exists to prevent, so the command must not be a way around it.
	for _, tc := range []struct {
		name  string
		setUp func(p *project)
		says  string
	}{
		{"a doc that is already Accepted", func(p *project) {
			p.write("docs/features/0002-done.md",
				fullDoc("0002", "Done", "Accepted", "- [ ] Do it"))
		}, "Accepted"},
		{"a doc that has shipped", func(p *project) {
			p.write("docs/features/0002-done.md",
				fullDoc("0002", "Done", "Shipped", "- [x] Did it"))
		}, "Shipped"},
		{"open questions", func(p *project) {
			p.write("docs/features/0002-open.md",
				fullDoc("0002", "Open", "Draft", "- [ ] Do it")+
					"\n## Open questions\n- Do codes expire?\n")
		}, "Open questions"},
		{"a blocking inbox line", func(p *project) {
			p.write("docs/features/0002-stuck.md",
				fullDoc("0002", "Stuck", "Draft", "- [ ] Do it"))
			p.run("block", "0002", "Which way round?").ok()
		}, "blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := setUp(t)
			tc.setUp(p)
			// Exit 2, not 1: none of these is another agent holding the feature, so
			// "take something else" would be the wrong instruction.
			p.run("accept", "0002").fails(2).errHas(tc.says)
		})
	}
}

func TestAcceptRefusesAnFYIItShouldNotBlockOn(t *testing.T) {
	t.Parallel()
	// The mirror of the case above. `--fyi` means work continues without an answer, so
	// accept must not treat one as a blocker — and it cannot decide that separately from
	// `next` without the two eventually disagreeing.
	p := setUp(t)
	p.write("docs/features/0002-fine.md", fullDoc("0002", "Fine", "Draft", "- [ ] Do it"))
	p.run("block", "0002", "Which host runs this?", "--fyi").ok()

	p.run("accept", "0002").ok()
}

func TestTheStatusRewriteStaysInsideTheFrontmatter(t *testing.T) {
	t.Parallel()
	// `(?m)^status:.*$` over the whole document rewrote every matching line, and a feature
	// doc for a tool whose subject *is* frontmatter quotes frontmatter constantly. Both
	// writers share one frontmatter-only helper now, so this pins accept and ship together.
	p := setUp(t)
	const example = "```yaml\nstatus: Draft\n```"
	body := fullDoc("0002", "Talks about frontmatter", "Draft", "- [x] Done") +
		"\n## Notes\n" + example + "\n"
	p.write("docs/features/0002-talks-about-frontmatter.md", body)

	for _, step := range []string{"accept", "ship"} {
		// ship refuses a tree with uncommitted work in it, and accept has just made some.
		p.commitAll("before " + step)
		p.run(step, "0002").ok()
		got := p.read("docs/features/0002-talks-about-frontmatter.md")
		if !strings.Contains(got, example) {
			t.Fatalf("kit %s rewrote a status: line inside a fenced example:\n%s", step, got)
		}
	}

	// And the frontmatter itself did change, or the test above would pass vacuously.
	if got := p.read("docs/features/0002-talks-about-frontmatter.md"); !strings.Contains(got,
		"status: Shipped") {
		t.Errorf("the frontmatter status was not updated:\n%s", got)
	}
}

func TestAcceptUsageErrorsAreExitTwo(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.run("accept").fails(2)
	p.run("accept", "0099").fails(2).errHas("no feature")
}

// fullDoc is a complete feature doc: every section a non-Draft needs, so these tests
// exercise `accept` and not `unfilled-template`.
func fullDoc(id, title, status, tasks string) string {
	return "---\nid: " + id + "\ntitle: " + title + "\nstatus: " + status +
		"\ncreated: 2026-01-01\nneeds:\n---\n\n" +
		"## Problem\nSomething needs doing.\n\n" +
		"## Design\nThis way.\n\n" +
		"## Tasks\n" + tasks + "\n\n" +
		"## Log\n- 2026-01-01 — proposed.\n"
}

func TestAcceptWritesTheReviewMarkOnlyWhenAsked(t *testing.T) {
	t.Parallel()
	// The field is the human's to set and this is the human's command, so it is the one
	// place it can be written without an agent deciding it. A doc written before the
	// field existed has to take it, which is every doc there is on the day this lands.
	p := cleanProject(t)
	p.write("docs/features/0002-marked.md",
		doc("0002", "Marked", "Draft", "", "- [ ] Something"))
	p.write("docs/features/0003-plain.md",
		doc("0003", "Plain", "Draft", "", "- [ ] Something"))

	p.run("accept", "0002", "--review").ok().has("review: human")
	marked := p.read("docs/features/0002-marked.md")
	if !strings.Contains(marked, "review: human") {
		t.Errorf("accept --review wrote no review: line:\n%s", marked)
	}
	if !strings.Contains(marked, "status: Accepted") {
		t.Errorf("accept --review did not accept the doc:\n%s", marked)
	}

	// Without the flag nothing is marked: the default is that a coordinator may merge,
	// and a mark nobody asked for would be the kit deciding what a human decides.
	p.run("accept", "0003").ok()
	if plain := p.read("docs/features/0003-plain.md"); strings.Contains(plain, "review:") {
		t.Errorf("accept wrote a review: line without --review:\n%s", plain)
	}
}
