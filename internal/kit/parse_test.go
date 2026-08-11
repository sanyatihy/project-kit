package kit

import (
	"testing"
	"time"
)

func TestParseFrontmatter(t *testing.T) {
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
	fields, body := ParseFrontmatter("no frontmatter here\n")
	if len(fields) != 0 {
		t.Errorf("fields = %v, want empty", fields)
	}
	if body != "no frontmatter here\n" {
		t.Errorf("body = %q, want the whole text", body)
	}
}

func TestParseSectionsIgnoresCommentGuidance(t *testing.T) {
	sections := ParseSections("## Tasks\n<!-- guidance only -->\n\n## Log\n- a note\n")
	if got := sections["Tasks"]; got != "" {
		t.Errorf("Tasks = %q, want empty — comment-only content is not filled in", got)
	}
	if got := sections["Log"]; got != "- a note" {
		t.Errorf("Log = %q, want %q", got, "- a note")
	}
}

func TestUntickedSkipsFencedExamples(t *testing.T) {
	got := Unticked("- [x] done\n- [ ] real work\n```\n- [ ] an example\n```\n")
	if len(got) != 1 || got[0] != "real work" {
		t.Errorf("Unticked = %q, want [real work]", got)
	}
}

func TestMentionsRequiresWholeNumber(t *testing.T) {
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
	if got := ParseDate("2026-08-11 · 0001 · BLOCKING"); !got.Equal(
		time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("ParseDate = %v, want 2026-08-11", got)
	}
	if got := ParseDate("YYYY-MM-DD"); !got.IsZero() {
		t.Errorf("ParseDate(placeholder) = %v, want zero", got)
	}
}

func TestNormalise(t *testing.T) {
	if got := Normalise("\uFEFFa\r\nb\rc"); got != "a\nb\nc" {
		t.Errorf("Normalise = %q, want %q", got, "a\nb\nc")
	}
}

func TestShowKeepsIdentifiersIntact(t *testing.T) {
	got := Show("Migrate `links(id, created_at)` **now**", 80)
	if want := "Migrate links(id, created_at) now"; got != want {
		t.Errorf("Show = %q, want %q", got, want)
	}
}
