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
