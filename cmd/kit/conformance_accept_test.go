package main

// Conformance: `kit accept`. The one transition that belongs to a human, and the only
// one that had no command — so it was done by hand, which is what `kit ship` exists to
// stop doing for the other end of the lifecycle.

import (
	"strings"
	"testing"
)

func TestAcceptSettlesADraftAndNextThenOffersIt(t *testing.T) {
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
	p := setUp(t)
	p.write("docs/features/0003-thing.md", fullDoc("0003", "Thing", "Draft", "- [ ] Do it"))
	p.run("accept", "3").ok()
}

func TestAcceptRefusesWhatIsNotSettled(t *testing.T) {
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
	// The mirror of the case above. `--fyi` means work continues without an answer, so
	// accept must not treat one as a blocker — and it cannot decide that separately from
	// `next` without the two eventually disagreeing.
	p := setUp(t)
	p.write("docs/features/0002-fine.md", fullDoc("0002", "Fine", "Draft", "- [ ] Do it"))
	p.run("block", "0002", "Which host runs this?", "--fyi").ok()

	p.run("accept", "0002").ok()
}

func TestTheStatusRewriteStaysInsideTheFrontmatter(t *testing.T) {
	// `(?m)^status:.*$` over the whole document rewrote every matching line, and a feature
	// doc for a tool whose subject *is* frontmatter quotes frontmatter constantly. Both
	// writers share one frontmatter-only helper now, so this pins accept and ship together.
	p := setUp(t)
	const example = "```yaml\nstatus: Draft\n```"
	body := fullDoc("0002", "Talks about frontmatter", "Draft", "- [x] Done") +
		"\n## Notes\n" + example + "\n"
	p.write("docs/features/0002-talks-about-frontmatter.md", body)

	for _, step := range []string{"accept", "ship"} {
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
