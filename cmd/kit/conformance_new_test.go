package main

// Conformance: `kit new`. IDs must be stable and unique, because branches and old
// commits cite them — a reused number makes history point at the wrong document.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// setUp returns a project that has already been initialised.
func setUp(t *testing.T) *project {
	t.Helper()
	p := newProject(t)
	p.run("init").ok()
	return p
}

func TestNewStartsAtOneAndCounts(t *testing.T) {
	p := setUp(t)
	p.run("new", "Shorten URLs").ok().has("0001-shorten-urls.md")
	p.run("new", "Rate limit").ok().has("0002-rate-limit.md")
}

func TestNewTakesTheHighestIDNotTheFirstGap(t *testing.T) {
	// Filling a gap would reuse the number of a dropped feature, and every branch
	// and commit citing it would then point at the wrong doc.
	p := setUp(t)
	p.write("docs/features/0001-first.md", "---\nid: 0001\nstatus: Shipped\n---\n")
	p.write("docs/features/0003-third.md", "---\nid: 0003\nstatus: Shipped\n---\n")

	p.run("new", "Fourth").ok().has("0004-fourth.md")
	if p.exists("docs/features/0002-fourth.md") {
		t.Error("new filled the 0002 gap; IDs must only ever go up")
	}
}

func TestNewSlugsTheTitle(t *testing.T) {
	p := setUp(t)
	for _, tc := range []struct{ title, want string }{
		{"Shorten URLs", "0001-shorten-urls.md"},
		{"Rate-limit POST by IP!", "0002-rate-limit-post-by-ip.md"},
		{"  Leading and trailing  ", "0003-leading-and-trailing.md"},
	} {
		p.run("new", tc.title).ok().has(tc.want)
		if !p.exists("docs/features/" + tc.want) {
			t.Errorf("new %q did not write %s", tc.title, tc.want)
		}
	}
}

func TestNewFillsTheFrontmatter(t *testing.T) {
	p := setUp(t)
	p.run("new", "Shorten URLs").ok()
	doc := p.read("docs/features/0001-shorten-urls.md")

	// The binary still reads the real clock — only the library takes a date — so
	// today is the honest expectation here. It is exact rather than "some date"
	// because NewFeature no longer invents one.
	for _, want := range []string{
		"id: 0001",
		"title: Shorten URLs",
		"created: " + time.Now().UTC().Format("2006-01-02"),
		"status: Draft",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("the new doc is missing %q:\n%s", want, doc)
		}
	}
	// A new doc must start as a Draft, or `kit next` would hand out a design
	// nobody has settled.
	if strings.Contains(doc, "status: Accepted") {
		t.Error("a new doc must not be Accepted")
	}
}

func TestANewDocIsEmptyToTheToolNotJustToTheReader(t *testing.T) {
	// The template's guidance lives in HTML comments, which every parser here strips.
	// When it lived in visible prose and two `- [ ] …` boxes, a doc nobody had written
	// read as a plan: `next` counted 0/2 and handed out `…` as the task to build, and
	// `audit` said the docs and the repo agreed.
	p := setUp(t)
	p.run("new", "Untouched").ok()
	p.write("docs/features/0001-untouched.md",
		strings.Replace(p.read("docs/features/0001-untouched.md"),
			"status: Draft", "status: Accepted", 1))

	// Nothing to hand out, and the audit says why rather than nothing.
	r := p.run("next").ok()
	r.lacks("…")
	p.run("audit").ok().has("unfilled-template")

	// One defect, one finding: an empty ## Tasks is unfilled-template's and not also
	// no-tasks', which covers the narrower case of prose with no boxes in it.
	if codes := p.auditCodes(); slices.Contains(codes, "no-tasks") {
		t.Errorf("an unwritten doc reported both unfilled-template and no-tasks: %v", codes)
	}
}

func TestNewNeverClobbersAnExistingFile(t *testing.T) {
	// Even a file that is not a valid feature doc counts toward the next ID, so
	// the name is skipped rather than the content replaced.
	p := setUp(t)
	p.write("docs/features/0001-shorten-urls.md", "not a feature doc at all\n")

	p.run("new", "Shorten URLs").ok()

	if got := p.read("docs/features/0001-shorten-urls.md"); got != "not a feature doc at all\n" {
		t.Errorf("new overwrote an existing file: %q", got)
	}
	if !p.exists("docs/features/0002-shorten-urls.md") {
		t.Error("new did not fall through to the next free ID")
	}
}

func TestNewUnderConcurrencyLosesNothing(t *testing.T) {
	// One agent per feature is the model, so several can run `new` at once. A
	// loser must report an error rather than silently overwrite the winner. The
	// interleaving is not deterministic, so assert the invariant instead: every
	// success left its own non-empty file behind.
	p := setUp(t)

	const racers = 4
	var wg sync.WaitGroup
	codes := make([]int, racers)
	for i := range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = p.run("new", "Race").Code
		}()
	}
	wg.Wait()

	succeeded := 0
	for _, c := range codes {
		if c == 0 {
			succeeded++
		}
	}
	if succeeded == 0 {
		t.Fatal("every concurrent new failed; at least one must win")
	}

	matches, err := filepath.Glob(p.path("docs/features/*-race.md"))
	if err != nil {
		t.Fatalf("globbing: %v", err)
	}
	if len(matches) != succeeded {
		t.Errorf("%d runs reported success but %d files exist — one overwrote another",
			succeeded, len(matches))
	}
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.Size() == 0 {
			t.Errorf("%s is empty; a doc was announced but not written", filepath.Base(m))
		}
	}
}

func TestNewRefusesATitleItCannotUse(t *testing.T) {
	p := setUp(t)
	// The empty cases are caught by the CLI before NewFeature sees them, so these
	// pin the behaviour a user gets, not which layer produces it — NewFeature's own
	// empty-title guard is unreachable through the binary and defends direct
	// library callers.
	p.run("new").fails(2).errHas("title")
	p.run("new", "   ").fails(2).errHas("title")
	// Punctuation alone slugs to nothing, which would produce "0001-.md". This one
	// is NewFeature's.
	p.run("new", "!!!").fails(2).errHas("no usable characters")
}

func TestNewOnAnUnsetUpProjectSaysHowToFixIt(t *testing.T) {
	p := newProject(t)
	r := p.run("new", "Anything").fails(2)
	r.errHas("kit init")
}
