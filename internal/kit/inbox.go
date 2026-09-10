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

// inboxQuestion is one unticked inbox line.
//
// Feature is the ID from the line's second field, empty when the line does not
// follow the documented format. Blocking decisions use it in preference to the
// whole line, so a question that merely mentions another feature in passing does
// not stop work on it.
type inboxQuestion struct {
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
	fields := strings.SplitN(text, questionSep, 4)
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

// inboxQuestions reads the inbox, skipping fenced examples.
func (r *Repo) inboxQuestions() []inboxQuestion {
	if r.Inbox == "" {
		return nil
	}
	raw, err := os.ReadFile(r.Inbox)
	if err != nil {
		return nil
	}
	var out []inboxQuestion
	for _, text := range Unticked(Normalise(string(raw))) {
		if text == "" {
			continue
		}
		q := inboxQuestion{When: ParseDate(text), Text: text}
		if id, fyi, ok := parseInboxLine(text); ok {
			q.Feature, q.FYI = id, fyi
		} else {
			q.FYI = isFYI(text)
		}
		out = append(out, q)
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
