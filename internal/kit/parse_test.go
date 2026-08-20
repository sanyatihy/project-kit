package kit

import (
	"testing"
	"time"
)

func TestParseFrontmatter(t *testing.T) {
	t.Parallel()
	fields, body := ParseFrontmatter("---\nid: 0007\n" +
		"status: Accepted        # Draft | Accepted\n" +
		"# a comment line\nshipped:\n---\n\n## Problem\np\n")

	if got := fields["id"]; got != "0007" {
		t.Errorf("id = %q, want 0007", got)
	}
	if got := fields["status"]; got != "Accepted" {
		t.Errorf("status = %q, want Accepted (trailing comment must be dropped)", got)
	}
	if got, ok := fields["shipped"]; !ok || got != "" {
		t.Errorf("shipped = %q (%v), want empty and present", got, ok)
	}
	if _, ok := fields["# a comment line"]; ok {
		t.Error("comment line was parsed as a field")
	}
	if want := "\n\n## Problem\np\n"; body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestParseFrontmatterAbsent(t *testing.T) {
	t.Parallel()
	fields, body := ParseFrontmatter("no frontmatter here\n")
	if len(fields) != 0 {
		t.Errorf("fields = %v, want empty", fields)
	}
	if body != "no frontmatter here\n" {
		t.Errorf("body = %q, want the whole text", body)
	}
}

func TestParseSectionsIgnoresCommentGuidance(t *testing.T) {
	t.Parallel()
	sections := ParseSections("## Tasks\n<!-- guidance only -->\n\n## Log\n- a note\n")
	if got := sections["Tasks"]; got != "" {
		t.Errorf("Tasks = %q, want empty — comment-only content is not filled in", got)
	}
	if got := sections["Log"]; got != "- a note" {
		t.Errorf("Log = %q, want %q", got, "- a note")
	}
}

func TestUntickedSkipsFencedExamples(t *testing.T) {
	t.Parallel()
	got := Unticked("- [x] done\n- [ ] real work\n```\n- [ ] an example\n```\n")
	if len(got) != 1 || got[0] != "real work" {
		t.Errorf("Unticked = %q, want [real work]", got)
	}
}

func TestMentionsRequiresWholeNumber(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text, id string
		want     bool
	}{
		{"closes 0007", "0007", true},
		{"feat/0007-thing", "0007", true},
		{"0007", "0007", true},
		{"10007", "0007", false},
		{"00071", "0007", false},
		{"nothing here", "0007", false},
		{"anything", "", false},
	}
	for _, c := range cases {
		if got := Mentions(c.text, c.id); got != c.want {
			t.Errorf("Mentions(%q, %q) = %v, want %v", c.text, c.id, got, c.want)
		}
	}
}

func TestParseDate(t *testing.T) {
	t.Parallel()
	if got := ParseDate("2026-08-11 · 0001 · BLOCKING"); !got.Equal(
		time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("ParseDate = %v, want 2026-08-11", got)
	}
	if got := ParseDate("YYYY-MM-DD"); !got.IsZero() {
		t.Errorf("ParseDate(placeholder) = %v, want zero", got)
	}
}

func TestNormalise(t *testing.T) {
	t.Parallel()
	if got := Normalise("\uFEFFa\r\nb\rc"); got != "a\nb\nc" {
		t.Errorf("Normalise = %q, want %q", got, "a\nb\nc")
	}
}

func TestShowKeepsIdentifiersIntact(t *testing.T) {
	t.Parallel()
	got := Show("Migrate `links(id, created_at)` **now**", 80)
	if want := "Migrate links(id, created_at) now"; got != want {
		t.Errorf("Show = %q, want %q", got, want)
	}
}

// The feature ID lives in a fixed field of an inbox line. Matching the whole line
// instead blocks any feature the question happens to mention, which halts work for
// a reason nobody can see.
func TestParseInboxLine(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, line string
		wantID     string
		wantFYI    bool
		wantOK     bool
	}{
		{"blocking", "2026-08-11 · 0007 · BLOCKING · Which encoding?", "0007", false, true},
		{"fyi", "2026-08-11 · 0007 · FYI · Using base62.", "0007", true, true},
		{"lowercase marker", "2026-08-11 · 0007 · blocking · Which?", "0007", false, true},
		{"separator in the question", "2026-08-11 · 0007 · BLOCKING · A · B?", "0007", false, true},
		{"no date", "someday · 0007 · BLOCKING · Which?", "", false, false},
		{"unknown marker", "2026-08-11 · 0007 · MAYBE · Which?", "", false, false},
		{"too few fields", "2026-08-11 · 0007", "", false, false},
		{"free prose", "we should decide about 0007 at some point", "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id, fyi, ok := parseInboxLine(tc.line)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v — an unrecognised line must fall back, not parse wrongly", ok, tc.wantOK)
			}
			if id != tc.wantID || fyi != tc.wantFYI {
				t.Errorf("id = %q, fyi = %v; want %q, %v", id, fyi, tc.wantID, tc.wantFYI)
			}
		})
	}
}

func TestParseIDList(t *testing.T) {
	t.Parallel()
	// Written by hand as often as by a tool, so the shapes people actually type all
	// have to work, and junk has to be dropped rather than change the board.
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"0002", []string{"0002"}},
		{"0002, 0003", []string{"0002", "0003"}},
		{"0002 0003", []string{"0002", "0003"}},
		{"[0002, 0003]", []string{"0002", "0003"}},
		{"0002, 0002", []string{"0002"}},       // a repeat is one dependency
		{"none", nil},                          // prose is not an id
		{"see 0004 for why", []string{"0004"}}, // and an id inside prose still counts
		{"12345", nil},                         // not the shape of an id
	} {
		got := ParseIDList(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("ParseIDList(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("ParseIDList(%q) = %v, want %v", tc.in, got, tc.want)
				break
			}
		}
	}
}

func TestBloatCountsLinesAndIgnoresTrailingBlanks(t *testing.T) {
	t.Parallel()
	// Trailing blank lines are an editor artefact rather than something a reader pays
	// for, so a doc must not cross the limit by being saved.
	doc := "---\nid: 0001\n---\n\n## Tasks\n- [x] one box\n\n## Log\n- 2026-01-01 — one line.\n\n\n\n"
	if lines := Bloat(doc); lines != 9 {
		t.Errorf("Bloat counted %d lines, want 9", lines)
	}
}

func TestNarrationSkipsWhatADocAboutTheRuleQuotes(t *testing.T) {
	t.Parallel()
	// The whole risk in this check: a doc about the rule quotes the markers, and the
	// template states them inside comments. Reporting either makes the rule report itself.
	quoted := "## Design\n<!-- never \"this session\" -->\n\n```\n- this session went badly\n```\n"
	if m, _ := Narration(quoted); m != "" {
		t.Errorf("fired on %q quoted in a comment and a fence", m)
	}
	// And a comment span running over several lines, which is how the template writes them.
	span := "## Log\n<!-- guidance\n     about this session\n     ends here -->\n- 2026-01-01 — a decision.\n"
	if m, _ := Narration(span); m != "" {
		t.Errorf("fired on %q inside a multi-line comment span", m)
	}
	// The loud half, with the line a reader would open.
	if m, line := Narration("## Log\n- 2026-01-01 — this session found it.\n"); m != "this session" || line != 2 {
		t.Errorf("got %q at line %d, want \"this session\" at 2", m, line)
	}
}

func TestNarrationCatchesAMarkerSplitByTheWrap(t *testing.T) {
	t.Parallel()
	// Docs are hard-wrapped, and every marker is several words, so a line-at-a-time scan
	// misses whichever ones the wrap happens to split. "wrong as written" straddled it in
	// the doc the markers were taken from.
	split := "## Log\n- 2026-01-01 — the cap diagnosis was\n  wrong, and here is the real one.\n"
	if m, line := Narration(split); m != "was wrong" || line != 2 {
		t.Errorf("got %q at line %d, want \"was wrong\" at 2", m, line)
	}
	// The line reported is where the marker starts. A marker lying wholly in the next
	// line belongs to that line, not to the one the scan happened to join it to.
	next := "## Log\n- 2026-01-01 — a decision, stated plainly.\n- 2026-01-02 — that was wrong.\n"
	if m, line := Narration(next); m != "was wrong" || line != 3 {
		t.Errorf("got %q at line %d, want \"was wrong\" at 3", m, line)
	}
}

func TestLongSentenceCountsAcrossTheWrapAndNamesTheLine(t *testing.T) {
	t.Parallel()
	// Docs here are hard-wrapped around 95 characters, so anything worth reporting spans
	// several lines. A line-at-a-time count would see four short lines and report nothing.
	wrapped := "## Design\nA claim, stated plainly and at rather more length than it needs here.\n" +
		"The cache is per-process because a shared one\n" +
		"would need invalidation across every worker, which this deployment does not\n" +
		"have, and it also drops the dependency nobody has had time to remove.\n"
	if n, line := LongSentence(wrapped, 30); n != 32 || line != 3 {
		t.Errorf("got %d words at line %d, want 32 at 3", n, line)
	}
	// The line named is where the sentence starts, and the first one over is the one
	// reported — not the longest, and not the one that happens to sit last.
	if n, line := LongSentence(wrapped, 10); n != 13 || line != 2 {
		t.Errorf("got %d words at line %d, want 13 at 2", n, line)
	}
}

func TestLongSentenceSplitsWhereAClaimEnds(t *testing.T) {
	t.Parallel()
	// Two claims of twelve words are not one of twenty-four. The abbreviation and the
	// filename are the two ways a period lies about where a sentence ended.
	for _, text := range []string{
		"## Design\nThe cache is per-process and holds one entry per worker thread here.\n" +
			"A shared one would need cross-worker invalidation, which does not exist yet.\n",
		"## Design\nRoute the durable fact to docs/TECH.md. one line, chosen as you write it.\n",
		"## Design\nEvery backend, e.g. the local one, reports the same four counters here.\n",
	} {
		if n, line := LongSentence(text, 16); n != 0 {
			t.Errorf("joined two claims into %d words at line %d:\n%s", n, line, text)
		}
	}
	// A sentence can open on an identifier and still be a new claim.
	code := "## Design\nThe check reports one finding per doc, and never more than that.\n" +
		"`long-sentence` at LOW, with its threshold measured rather than picked.\n"
	if n, _ := LongSentence(code, 16); n != 0 {
		t.Errorf("joined a sentence opening on an identifier: %d words", n)
	}
}

func TestLongSentenceIgnoresWhatIsNotProse(t *testing.T) {
	t.Parallel()
	// A doc about writing quotes long examples, and the template's guidance is nothing
	// but comments — Narration's bargain. A task box is an outcome 0037 froze as written,
	// and a table row is a cell however wide it is.
	long := "The cache is per-process because a shared one would need invalidation across " +
		"every worker, which this deployment does not have, and it drops a dependency."
	for _, text := range []string{
		"## Design\n<!-- " + long + " -->\n",
		"## Design\n\n```\n" + long + "\n```\n",
		"## Design\n\n| what | why |\n| --- | --- |\n| a row | " + long + " |\n",
		"## Tasks\n- [ ] " + long + "\n",
	} {
		if n, line := LongSentence(text, 20); n != 0 {
			t.Errorf("counted %d words at line %d in:\n%s", n, line, text)
		}
	}
	// Two list items are two thoughts, however they wrap; joining them invents a sentence.
	items := "## Non-goals\n- A short reason, given once and no more than that\n" +
		"- Another short reason, given once and no more than that\n"
	if n, _ := LongSentence(items, 16); n != 0 {
		t.Errorf("joined two list items into %d words", n)
	}
}
