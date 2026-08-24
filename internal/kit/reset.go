package kit

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ResetResult is what resetting changed, for reporting.
type ResetResult struct {
	Doc    string
	Was    string
	Commit string // the commit the status landed in, empty when there is no history to commit onto
}

// Reset returns a doc to Draft.
//
// It exists because an unrecognised status wedges the doc that carries it: `claim` says it
// is not open work, `ship` refuses while a box is open, and the only command that could
// write over it was `drop`, which retires the feature rather than repairing it. Every
// instruction left named an edit to `status:` — the thing that put the bad value there.
//
// So this is the third command that writes a status, and the field stays the kit's. Any
// status but Shipped is accepted, which covers the unrecognised value and reviving a
// retired plan under one rule. Shipped is refused for `drop`'s reason: the work is in the
// repo, and a status saying otherwise would make the record say it was never done.
func Reset(r *Repo, id, why string, today time.Time) (*ResetResult, error) {
	target, err := r.WriteTarget(id)
	if err != nil {
		return nil, err
	}
	switch target.Status {
	case StatusShipped:
		return nil, fmt.Errorf("%s is Shipped — it is frozen history, and reopening it "+
			"would make the record say the work was never done", target.Name())
	case StatusDraft:
		return nil, fmt.Errorf("%s is already Draft", target.Name())
	}

	// `ship`'s guard, for `ship`'s reason: a doc another checkout claims is work in
	// flight, and rewriting its status from here is the same interference.
	if claims := r.Claims(id); len(claims) > 0 && !r.Mine(id) {
		return nil, fmt.Errorf("%s is claimed by %s, which is not this checkout\n"+
			"reset it from that worktree. If the branch is a leftover of yours, release "+
			"the claim first: kit release "+target.ID, target.Name(), strings.Join(claims, " and "))
	}

	raw, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	updated, ok := SetStatus(Normalise(string(raw)), StatusDraft)
	if !ok {
		return nil, fmt.Errorf("%s has no status: line in its frontmatter to set", target.Name())
	}
	// `drop`'s argument: a status change nobody explained reads later as a mistake, and
	// the doc is where every other durable decision lives.
	if why != "" {
		updated = appendLog(updated,
			"- "+today.UTC().Format(DateLayout)+" — reset to Draft: "+why)
	}
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}

	res := &ResetResult{Doc: target.Rel, Was: target.Status}
	if r.HasHistory() {
		res.Commit = "Reset " + target.ID
		if err := r.commitPaths(res.Commit, target.Path); err != nil {
			return nil, fmt.Errorf("%s is back to Draft in %s, and committing it failed:\n%v\n"+
				"commit that file yourself — the status is repaired only once the commit exists",
				target.Name(), target.Rel, err)
		}
	}
	return res, nil
}
