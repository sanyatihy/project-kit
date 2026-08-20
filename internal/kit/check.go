package kit

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// CheckResult is what recording an outcome changed, for reporting.
type CheckResult struct {
	Doc string
	Due time.Time // the date the doc named in check:
}

// RecordCheck settles a bet: it stamps `checked:` and writes the outcome into `## Log`.
//
// Those two writes are what `unchecked` asks for, and both land in a doc that is almost
// always Shipped — which rule 11 freezes. A command that owns them is what lets the rule
// stay absolute, the way `ship` owning `status:` did.
//
// A doc that names no `check:` date is refused rather than scheduled. The field says a
// feature is a bet worth revisiting and it is set while the doc is written; a revisit
// somebody wants after the fact is a BACKLOG.md line, which costs nothing and edits no
// frozen history.
func RecordCheck(r *Repo, id, outcome string, today time.Time) (*CheckResult, error) {
	var target *Feature
	for _, f := range r.Features {
		if f.ID == id {
			target = f
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("no feature %q", id)
	}
	if outcome == "" {
		return nil, fmt.Errorf("%s: say what happened — a stamped date with no outcome "+
			"records that somebody looked, not what they found", target.Name())
	}
	if target.Check.IsZero() {
		return nil, fmt.Errorf("%s names no check: date, so there is no bet to settle\n"+
			"that field is set while the doc is written; a revisit you want now is a "+
			"line in BACKLOG.md", target.Name())
	}
	if !target.Checked.IsZero() {
		return nil, fmt.Errorf("%s was already checked on %s",
			target.Name(), target.Checked.UTC().Format(DateLayout))
	}

	raw, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	stamped := today.UTC().Format(DateLayout)
	updated, ok := AddField(Normalise(string(raw)), "checked", stamped)
	if !ok {
		return nil, fmt.Errorf("%s has no frontmatter to stamp checked: into", target.Name())
	}
	updated = appendLog(updated, "- "+stamped+" — checked: "+outcome)
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}
	return &CheckResult{Doc: target.Rel, Due: target.Check}, nil
}

// appendLog puts one entry in a doc's ## Log, writing the section when there is none.
//
// The section is not optional in practice — `missing-section` reports a doc without one —
// but a caller that drops its entry on the floor is worse than one that adds a heading.
func appendLog(body, entry string) string {
	if logged, ok := AppendToSection(body, "Log", entry); ok {
		return logged
	}
	return strings.TrimRight(body, "\n") + "\n\n## Log\n" + entry + "\n"
}
