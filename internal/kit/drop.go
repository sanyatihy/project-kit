package kit

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// DropResult is what dropping changed, for reporting.
type DropResult struct {
	Doc        string
	Was        string
	Claims     []string // branches still named after the doc, which nothing here deletes
	Dependants []string // features declaring this id in needs:, which nothing here edits
}

// Drop retires a feature that will not be built.
//
// It exists because drafts live on the default branch now. Under the old flow an
// unwanted proposal was a branch you deleted and it was gone; a doc on the trunk
// cannot be removed that way, and left alone it sits on the board for ever — `next`
// offers it, and every agent reads it as work nobody has started.
//
// Same reasoning as `accept` and `ship`: the edit is one frontmatter field, and doing
// it by hand is where `unticked-shipped` and `shipped-no-date` came from. A command
// that cannot get it wrong beats a check that reports it afterwards.
//
// Shipped is refused rather than dropped. A shipped doc is frozen history and the work
// is in the repo; retiring it would make the record say the feature was never built.
// today is passed in rather than read here, the same way Ship takes it, so a test can
// assert the date the log entry gets instead of guessing at the one the clock produced.
func Drop(r *Repo, id, why string, today time.Time) (*DropResult, error) {
	target := r.FeatureByID(id)
	if target == nil {
		return nil, fmt.Errorf("no feature %q", id)
	}
	switch target.Status {
	case StatusShipped:
		return nil, fmt.Errorf("%s is Shipped — it is frozen history, not a plan to retire",
			target.Name())
	case "Dropped":
		return nil, fmt.Errorf("%s is already Dropped", target.Name())
	}

	raw, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	updated, ok := SetStatus(Normalise(string(raw)), "Dropped")
	if !ok {
		return nil, fmt.Errorf("%s has no status: line in its frontmatter to set", target.Name())
	}
	// The reason goes in the ## Log, which is where every other durable decision lives.
	// A dropped feature with no reason is the same defect as a shipped one with no log:
	// the next person reads a dead doc and cannot tell whether it was wrong or forgotten.
	if why != "" {
		entry := "- " + today.UTC().Format(DateLayout) + " — dropped: " + why
		// A doc with no ## Log gets one. This used to drop the line on the floor, while
		// the only warning printed fires when *no* reason was given — so the operator
		// who supplied one was never told it had been thrown away.
		updated = appendLog(updated, entry)
	}
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}
	return &DropResult{
		Doc:        target.Rel,
		Was:        target.Status,
		Claims:     r.Claims(id),
		Dependants: dependants(r, id),
	}, nil
}

// dependants names the features declaring this id in needs:.
//
// Reported rather than refused: a feature that will not be built is exactly when you want
// to know who was waiting on it, and refusing would leave the dropped doc stuck open.
// Nothing here edits them — what replaces a dropped plan is a judgement the kit does not
// have — but leaving it unsaid is how they got stranded: `audit` reported `needs-unknown`
// afterwards and `next` went on promising a wait that can never end.
func dependants(r *Repo, id string) []string {
	var out []string
	for _, f := range r.Features {
		if f.ID == id {
			continue
		}
		for _, need := range f.Needs {
			if need == id {
				out = append(out, f.Name())
				break
			}
		}
	}
	return out
}

// appendLog puts an entry in `## Log`, creating the section when a doc predates it.
// `drop` is the one command that writes a Log entry, because retiring a feature is a
// change to the plan that nothing else records.
func appendLog(body, entry string) string {
	if logged, ok := AppendToSection(body, "Log", entry); ok {
		return logged
	}
	return strings.TrimRight(body, "\n") + "\n\n## Log\n" + entry + "\n"
}
