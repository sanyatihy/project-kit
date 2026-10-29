package kit

import (
	"fmt"
	"os"
	"strings"
)

// ReopenResult is what reopening changed, for reporting.
type ReopenResult struct {
	Doc    string
	Was    string
	Commit string // the commit the status landed in, empty when there is no history to commit onto
}

// Reopen repairs a status or returns unmerged work to Draft for review changes.
// A Submitted document on the trunk is frozen history.
func Reopen(r *Repo, id, why string, repair bool) (*ReopenResult, error) {
	target, err := r.WriteTarget(id)
	if err != nil {
		return nil, err
	}
	switch target.Status {
	case StatusSubmitted, StatusLegacyShipped:
		if err := r.canReopen(target); err != nil {
			return nil, err
		}
	case StatusDropped:
		return nil, fmt.Errorf("%s is Dropped — draft a new feature for human approval instead of reviving a retired plan", target.Name())
	case StatusParked:
		return nil, fmt.Errorf("%s is Parked — use kit activate %s on a planning branch when the operator asks", target.Name(), target.ID)
	case StatusDraft:
		return nil, fmt.Errorf("%s is already Draft", target.Name())
	default:
		if !repair {
			return nil, fmt.Errorf("%s has invalid status %q — use kit reopen %s --repair \"<why>\" to repair it", target.Name(), target.Status, id)
		}
	}

	// `submit`'s guard, for `submit`'s reason: a doc another checkout claims is work in
	// flight, and rewriting its status from here is the same interference.
	if claims := r.Claims(id); len(claims) > 0 && !r.Mine(id) {
		return nil, fmt.Errorf("%s is claimed by %s, which is not this checkout\n"+
			"reopen it from that worktree. If the branch is a leftover of yours, release "+
			"the claim first: kit release "+target.ID, target.Name(), strings.Join(claims, " and "))
	}

	raw, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	// AddField and not SetStatus: a doc carrying no `status:` line at all is wedged the
	// same way — `claim` refuses it and `next` blocks it — and refusing here would leave
	// that one repairable only by the hand-edit this command exists to replace. A doc with
	// no frontmatter is where that stops: there is nowhere to write the field.
	updated, ok := AddField(Normalise(string(raw)), "status", StatusDraft)
	if !ok {
		return nil, fmt.Errorf("%s has no frontmatter to write a status into", target.Name())
	}
	updated = RemoveField(updated, "shipped")
	updated, _ = AddField(updated, "submitted", "")
	if err := replaceFile(target.Path, updated); err != nil {
		return nil, err
	}

	res := &ReopenResult{Doc: target.Rel, Was: target.Status}
	if r.HasHistory() {
		// The reason goes in the subject, not ## Log: a reopen changes no plan, and a task
		// revised afterwards gets its own Log entry (rule 12).
		res.Commit = "Reopen " + target.ID
		if why != "" {
			res.Commit += ": " + why
		}
		if err := r.commitPaths(res.Commit, target.Path); err != nil {
			return nil, fmt.Errorf("%s is back to Draft in %s, and committing it failed:\n%v\n"+
				"commit that file yourself — the status is repaired only once the commit exists",
				target.Name(), target.Rel, err)
		}
	}
	return res, nil
}

// canReopen reads the published trunk when there is one; a stale local copy cannot
// authorize rewriting history that another checkout already merged.
func (r *Repo) canReopen(f *Feature) error {
	trunk := r.trunkRef()
	if trunk == "" || r.HeadBranch() == "" || r.isTrunk(r.HeadBranch()) {
		return fmt.Errorf("%s is Submitted — reopening it needs its unmerged working branch", f.Name())
	}
	remote, err := pickRemote(r)
	if err != nil {
		return err
	}
	if remote != "" {
		tip, ok := r.remoteTip(remote, r.Trunk())
		if !ok {
			return fmt.Errorf("the remote trunk could not be read — fetch it before reopening %s", f.ID)
		}
		trunk = tip
	}
	statuses := r.statusesOn(trunk)
	if statuses == nil {
		return fmt.Errorf("the trunk's documents cannot be read — run git fetch before reopening %s", f.ID)
	}
	if IsSubmitted(statuses[f.Rel]) || r.reaches("HEAD", trunk) {
		return fmt.Errorf("%s is Submitted on the trunk — it is frozen history; write a new feature", f.Name())
	}
	return nil
}
