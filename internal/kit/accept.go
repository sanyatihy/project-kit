package kit

import (
	"fmt"
	"os"
)

// AcceptResult is what accepting changed, for reporting.
type AcceptResult struct {
	Doc        string
	Branch     string // the branch the doc's own claim would use, for the closing instruction
	NeedsHuman bool   // review: human was written, so a person merges this one
}

// Accept settles a Draft, which is the one transition in this model that belongs to a
// human.
//
// It exists for the same reason `Ship` does. Hand-editing frontmatter is what
// `unticked-shipped` and `shipped-no-date` are the wreckage of, and accepting by hand has
// the same failure modes available to it: a doc accepted with its `## Open questions`
// still open, or one a BLOCKING inbox line is waiting on, puts an agent to work on a
// design nobody settled.
//
// Deliberately no date stamped. `created:` already records when the feature was proposed,
// and a second date nobody reads is a field that goes wrong.
//
// needsHuman writes `review: human`, which says a person merges this one whatever the
// trunk allows. It is set here because this is the human's command and this is the moment
// they have the whole round in front of them — before anyone has sunk work into the
// outcome, and through the one command an agent never runs.
func Accept(r *Repo, id string, needsHuman bool) (*AcceptResult, error) {
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
	if target.Status != StatusDraft {
		return nil, fmt.Errorf("%s is %s, not a Draft — there is nothing to accept",
			target.Name(), orNoStatus(target.Status))
	}
	// Rule 3 of bootstrapping: unresolved questions are what Draft is for. Accepting over
	// them is how a design nobody settled reaches an agent.
	if open := target.Section("Open questions"); open != "" {
		return nil, fmt.Errorf("%s still has ## Open questions:\n  %s\n"+
			"settle them and delete the section, or move what only you can answer to %s",
			target.Name(), clip(open, 70), r.Rel(r.Inbox))
	}
	// BlockersFor rather than a second reading of the inbox, so `accept` and `next` can
	// never disagree about what is blocked.
	if blockers := r.BlockersFor(id); len(blockers) > 0 {
		return nil, fmt.Errorf("%s is blocked by a question in %s:\n  %s\n"+
			"answer it and delete the line first — kit next would refuse this feature anyway",
			target.Name(), r.Rel(r.Inbox), clip(blockers[0], 70))
	}

	raw, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	updated, ok := SetStatus(Normalise(string(raw)), StatusAccepted)
	if !ok {
		return nil, fmt.Errorf("%s has no status: line in its frontmatter to set",
			target.Name())
	}
	if needsHuman {
		// AddField, not SetField: the docs written before the field existed do not carry
		// it, and those are exactly the ones somebody is reading when they decide.
		if updated, ok = AddField(updated, "review", "human"); !ok {
			return nil, fmt.Errorf("%s has no frontmatter to write review: into",
				target.Name())
		}
	}
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}
	return &AcceptResult{Doc: target.Rel, Branch: target.Branch(), NeedsHuman: needsHuman}, nil
}

// orNoStatus describes a status for a message, including the case of not having one.
func orNoStatus(status string) string {
	if status == "" {
		return "missing a status"
	}
	return status
}
