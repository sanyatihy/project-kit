package kit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ResolveID maps what someone typed to the canonical ID of an existing feature,
// so `kit block 7` and `kit block 0007` both reach 0007.
//
// It fails on an unknown ID rather than accepting it. A question filed against a
// feature that does not exist can never unblock anything, and nothing would ever
// notice.
func ResolveID(r *Repo, typed string) (string, error) {
	want := strings.TrimSpace(typed)
	if want == "" {
		return "", fmt.Errorf("a feature id is required")
	}
	var known []string
	for _, f := range r.Features {
		if f.ID == "" {
			continue
		}
		known = append(known, f.ID)
		if f.ID == want || strings.TrimLeft(f.ID, "0") == strings.TrimLeft(want, "0") {
			return f.ID, nil
		}
	}
	if len(known) == 0 {
		return "", fmt.Errorf("no feature docs yet — run: kit new \"<what it does>\"")
	}
	return "", fmt.Errorf("no feature %q; have %s", typed, strings.Join(known, ", "))
}

// AppendQuestion writes one question to the inbox in the documented format and
// returns the line it wrote, creating the inbox when this is the first question.
//
// Writing the format is the point: a hand-typed line that misses it falls back to
// matching by prose, which blocks whatever it happens to mention.
func AppendQuestion(templates fs.FS, r *Repo, id, question string, fyi bool, today time.Time) (string, error) {
	// One item per line is what makes the file parseable at all.
	question = strings.Join(strings.Fields(question), " ")
	if question == "" {
		return "", fmt.Errorf("a question is required")
	}
	if r.Inbox == "" {
		created, err := createInbox(templates, r)
		if err != nil {
			return "", err
		}
		r.Inbox = created
	}
	marker := "BLOCKING"
	if fyi {
		marker = "FYI"
	}
	line := fmt.Sprintf("- [ ] %s %s %s %s %s %s %s",
		today.UTC().Format(DateLayout), inboxSep, id, inboxSep, marker, inboxSep, question)

	raw, err := os.ReadFile(r.Inbox)
	if err != nil {
		return "", err
	}
	body := Normalise(string(raw))
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	if err := os.WriteFile(r.Inbox, []byte(body+line+"\n"), 0o644); err != nil {
		return "", err
	}
	return line, nil
}

// createInbox writes the inbox template the first time a question needs somewhere to go.
//
// The same model as the as-built doc, which `kit ship` creates: a file scaffolded before
// the project has anything to put in it is one the project is nagged about for describing
// nothing. O_EXCL covers two agents blocking at once — the loser writes into the file the
// winner made rather than over it.
func createInbox(templates fs.FS, r *Repo) (string, error) {
	dest := filepath.Join(r.Root, "docs", "INBOX.md")
	raw, err := fs.ReadFile(templates, "templates/INBOX.md")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return dest, nil
		}
		return "", err
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return dest, nil
}

// OpenQuestion is one unticked inbox line.
//
// Feature is the ID from the line's second field, empty when the line does not
// follow the documented format. Blocking decisions use it in preference to the
// whole line, so a question that merely mentions another feature in passing does
// not stop work on it.
type OpenQuestion struct {
	When    time.Time
	Text    string
	Feature string
	FYI     bool
}

// parseInboxLine pulls the feature ID and the BLOCKING/FYI marker out of a line
// in the documented format, reporting whether it was in that format at all.
//
// The question text is deliberately not validated: it is prose and may contain
// anything, including the separator.
func parseInboxLine(text string) (id string, fyi, ok bool) {
	fields := strings.SplitN(text, inboxSep, 4)
	if len(fields) < 3 {
		return "", false, false
	}
	if ParseDate(strings.TrimSpace(fields[0])).IsZero() {
		return "", false, false
	}
	marker := strings.TrimSpace(fields[2])
	switch {
	case strings.EqualFold(marker, "FYI"):
		fyi = true
	case strings.EqualFold(marker, "BLOCKING"):
		fyi = false
	default:
		return "", false, false
	}
	if id = strings.TrimSpace(fields[1]); id == "" {
		return "", false, false
	}
	return id, fyi, true
}

// OpenQuestions reads the inbox, skipping fenced examples.
func (r *Repo) OpenQuestions() []OpenQuestion {
	if r.Inbox == "" {
		return nil
	}
	raw, err := os.ReadFile(r.Inbox)
	if err != nil {
		return nil
	}
	var out []OpenQuestion
	for _, text := range Unticked(Normalise(string(raw))) {
		if text == "" {
			continue
		}
		q := OpenQuestion{When: ParseDate(text), Text: text}
		if id, fyi, ok := parseInboxLine(text); ok {
			q.Feature, q.FYI = id, fyi
		} else {
			q.FYI = isFYI(text)
		}
		out = append(out, q)
	}
	return out
}

// BlockersFor lists open questions that block this feature.
//
// A line marked FYI is a note, not a blocker — it is recorded so the human sees
// it, but work continues. Anything else blocks, because an unmarked question is
// more likely to be a real one than a courtesy note.
//
// A line in the documented format is matched on its feature-ID field alone.
// Anything else falls back to scanning the whole line, which over-blocks: a
// question that names the ID only in passing will stop work. That is the safe
// direction to be wrong in — over-blocking is visible and annoying, while missing
// a blocker sends an agent into work a human has not cleared. `inbox-unparsed`
// reports the lines that land here so they can be fixed rather than tolerated.
func (r *Repo) BlockersFor(id string) []string {
	var out []string
	for _, q := range r.OpenQuestions() {
		if q.FYI {
			continue
		}
		matched := Mentions(q.Text, id)
		if q.Feature != "" {
			matched = Mentions(q.Feature, id)
		}
		if matched {
			out = append(out, q.Text)
		}
	}
	return out
}

func isFYI(text string) bool {
	for _, field := range strings.Split(text, "·") {
		if strings.EqualFold(strings.TrimSpace(field), "FYI") {
			return true
		}
	}
	return false
}
