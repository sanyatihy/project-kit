package kit

import (
	"testing"
	"time"
)

func TestParseFrontmatter(t *testing.T) {
	t.Parallel()
	fields, body := ParseFrontmatter("---\nid: 0007\n" +
		"status: Draft        # Draft | Submitted | Dropped\n" +
		"# a comment line\nsubmitted:\n---\n\n## Problem\np\n")

	if got := fields["id"]; got != "0007" {
		t.Errorf("id = %q, want 0007", got)
	}
	if got := fields["status"]; got != "Draft" {
		t.Errorf("status = %q, want Draft (trailing comment must be dropped)", got)
	}
	if got, ok := fields["submitted"]; !ok || got != "" {
		t.Errorf("submitted = %q (%v), want empty and present", got, ok)
	}
	if _, ok := fields["# a comment line"]; ok {
		t.Error("comment line was parsed as a field")
	}
	if want := "\n\n## Problem\np\n"; body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestParseFrontmatterKeepsAHashInsideAValue(t *testing.T) {
	t.Parallel()
	// The comment marker is the template's run of spaces before the #. A single space is
	// part of the value: an issue number in a title used to truncate the title to the
	// word before it, and every message naming the feature carried the stub.
	fields, _ := ParseFrontmatter("---\ntitle: Fix #123 crash\n" +
		"status: Draft        # Draft | Submitted | Dropped\n---\n")

	if got := fields["title"]; got != "Fix #123 crash" {
		t.Errorf("title = %q, want the whole line", got)
	}
	if got := fields["status"]; got != "Draft" {
		t.Errorf("status = %q, want Draft — the template's comment is still dropped", got)
	}
}

func TestParseFrontmatterClosesOnAWholeLine(t *testing.T) {
	t.Parallel()
	// A run of more than three dashes is a setext underline, not a fence. The close was a
	// substring search for the first "\n---", which matched it — so the block ended
	// early, every field below it was lost, and audit reported no-status on a doc that
	// declares one.
	fields, body := ParseFrontmatter("---\nid: 0007\ntitle: T\n---------\n" +
		"status: Draft\n---\n\n## Problem\np\n")

	if got := fields["status"]; got != "Draft" {
		t.Errorf("status = %q, want Draft — the block ended at the underline", got)
	}
	if want := "\n\n## Problem\np\n"; body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestDuplicateKeys(t *testing.T) {
	t.Parallel()
	// The state `submit` used to produce: it rewrote the first status: and the parser reads
	// the last, so the doc carries two and the kit and its author disagree about which.
	text := "---\nid: 0007\nstatus: Submitted\nstatus: Draft\nsubmitted:\n---\n\n" +
		"## Design\nstatus: Draft is a line in the body, not a second key\n"

	if got := DuplicateKeys(text); len(got) != 1 || got[0] != "status" {
		t.Errorf("DuplicateKeys = %q, want [status]", got)
	}
	if got := DuplicateKeys("---\nid: 0007\nstatus: Draft\n---\n"); got != nil {
		t.Errorf("DuplicateKeys = %q on a doc that repeats nothing, want none", got)
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
		{"12345", []string{"12345"}},           // ids grow past four digits
		{"123", nil},                           // and never shrink below them
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

func TestFencedExamplesCannotReplaceTheRealTaskSection(t *testing.T) {
	for _, fence := range []string{"```", "~~~~", "````"} {
		text := "## Tasks\n- [ ] Actual work\n\n" + fence + "markdown\n## Tasks\n- [x] Example\n" + fence + "\n"
		sections := ParseSections(text)
		open, total := CountTasks(sections["Tasks"])
		if open != 1 || total != 1 {
			t.Errorf("%q: got %d/%d, want one open real task", fence, open, total)
		}
	}
}
