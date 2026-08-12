package kit

import (
	"fmt"
	"os"
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
// returns the line it wrote.
//
// Writing the format is the point: a hand-typed line that misses it falls back to
// matching by prose, which blocks whatever it happens to mention.
func AppendQuestion(r *Repo, id, question string, fyi bool, today time.Time) (string, error) {
	if r.Inbox == "" {
		return "", fmt.Errorf("no docs/INBOX.md — run: kit init %s", r.Root)
	}
	// One item per line is what makes the file parseable at all.
	question = strings.Join(strings.Fields(question), " ")
	if question == "" {
		return "", fmt.Errorf("a question is required")
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
