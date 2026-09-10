package kit

import (
	"testing"
	"time"
)

// A question lives in the doc it blocks, so these read `## Open questions` and nothing
// else. What blocks is an unticked box; what ages is the date on it.

func questionDoc(id, section string) *Feature {
	return &Feature{ID: id, Title: "Shorten URLs",
		sections: map[string]string{QuestionsHeading: section}}
}

func TestOpenQuestionsReadsBoxesAndTheirDates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		section string
		want    []string
		when    time.Time
	}{
		{"a dated box", "- [ ] 2026-08-11 · Which acquirer?",
			[]string{"Which acquirer?"}, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)},
		{"an em dash instead of the separator", "- [ ] 2026-08-11 — Which acquirer?",
			[]string{"Which acquirer?"}, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)},
		{"no separator at all", "- [ ] 2026-08-11 Which acquirer?",
			[]string{"Which acquirer?"}, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)},
		// It blocks either way — the safe direction. Only the ageing is lost, which is
		// what `question-informal` exists to report.
		{"an undated box still asks something", "- [ ] Which acquirer?",
			[]string{"Which acquirer?"}, time.Time{}},
		{"a ticked box is answered", "- [x] 2026-08-11 · Which acquirer?", nil, time.Time{}},
		// The prose bullet 71 frozen docs already carry. Reading these as open would
		// block every settled question ever written down.
		{"a plain bullet is not a question", "- Which acquirer?", nil, time.Time{}},
		{"an empty section", "", nil, time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := questionDoc("0001", tc.section).OpenQuestions()
			if len(got) != len(tc.want) {
				t.Fatalf("OpenQuestions() = %v, want %v", got, tc.want)
			}
			for i, want := range tc.want {
				if got[i].Text != want {
					t.Errorf("Text = %q, want %q", got[i].Text, want)
				}
				if !got[i].When.Equal(tc.when) {
					t.Errorf("When = %v, want %v", got[i].When, tc.when)
				}
			}
		})
	}
}

func TestBlockersForReadsOnlyTheFeaturesOwnDoc(t *testing.T) {
	t.Parallel()
	r := &Repo{Features: []*Feature{
		questionDoc("0001", "- [ ] 2026-08-11 · Which acquirer?"),
		questionDoc("0002", ""),
		// The id in the prose used to block by a whole-line match against the inbox.
		// Nothing scans prose for ids now, so this stops 0003 and only 0003.
		questionDoc("0003", "- [ ] 2026-08-11 · Does this need 0002 first?"),
	}}

	if got := r.BlockersFor("0001"); len(got) != 1 || got[0] != "Which acquirer?" {
		t.Errorf("BlockersFor(0001) = %v, want the open question", got)
	}
	if got := r.BlockersFor("0002"); len(got) != 0 {
		t.Errorf("BlockersFor(0002) = %v, want none — another doc's question is not its own", got)
	}
	// An id with no doc has no section to read, and `block` refuses to write one for it.
	if got := r.BlockersFor("0099"); len(got) != 0 {
		t.Errorf("BlockersFor(0099) = %v, want none", got)
	}
}

func TestInformalQuestionsNamesWhatTheKitCannotActOn(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		section string
		want    int
	}{
		{"a dated box is what block writes", "- [ ] 2026-08-11 · Which acquirer?", 0},
		{"a ticked box is answered", "- [x] Which acquirer?", 0},
		// Blocks, but nothing can age it into `question-stale`.
		{"a box with no date", "- [ ] Which acquirer?", 1},
		// Blocks nothing, so whoever wrote it reaches nobody.
		{"a plain bullet", "- Which acquirer?", 1},
		// A wrapped bullet is one finding, not one per line.
		{"a wrapped bullet", "- Which acquirer? Leaning: the one\n  we already use.", 1},
		// The section may explain itself without every paragraph being a question.
		{"prose is not a bullet", "Settled in review.", 0},
		// A documented example is not a question, the way it is not a task.
		{"a fenced example", "```\n- [ ] YYYY-MM-DD · <question>\n```", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := questionDoc("0001", tc.section).InformalQuestions(); len(got) != tc.want {
				t.Errorf("InformalQuestions() = %v, want %d", got, tc.want)
			}
		})
	}
}

func TestAppendToSectionIgnoresHeadingsTheParserCannotSee(t *testing.T) {
	t.Parallel()
	// The writer has to find the section where the reader will. A fenced example or a
	// template comment containing `## Open questions` is not a section, and appending
	// into one wrote a question nothing read: `block` reported the feature blocked and
	// `next` went on offering it as ready.
	for _, tc := range []struct{ name, body string }{
		{"a fenced example", "## Design\n\n```\n## Open questions\n\n## Log\n```\n\n" +
			"## Open questions\n\n## Log\n"},
		{"a template comment", "## Design\n\n<!--\n## Open questions\n\n## Log\n-->\n\n" +
			"## Open questions\n\n## Log\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := AppendToSection(tc.body, QuestionsHeading, "- [ ] 2026-08-11 · Which?")
			if !ok {
				t.Fatal("the real heading was not found at all")
			}
			f := &Feature{ID: "0001", sections: ParseSections(got)}
			if len(f.OpenQuestions()) != 1 {
				t.Errorf("the question landed where nothing reads it:\n%s", got)
			}
		})
	}
}
