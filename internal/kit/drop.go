package kit

import (
	"fmt"
	"os"
	"time"
)

// DropResult is what dropping changed, for reporting.
type DropResult struct {
	Doc    string
	Was    string
	Claims []string // branches still named after the doc, which nothing here deletes
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
		if logged, ok := AppendToSection(updated, "Log",
			"- "+today.UTC().Format(DateLayout)+" — dropped: "+why); ok {
			updated = logged
		}
	}
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}
	return &DropResult{Doc: target.Rel, Was: target.Status, Claims: r.Claims(id)}, nil
}
