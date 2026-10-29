package kit

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// AvailabilityResult describes a planning decision committed for human review.
type AvailabilityResult struct {
	Doc, Status, Commit string
}

// Park preserves an unstarted plan until the operator chooses to activate it.
func Park(r *Repo, id, why string, today time.Time) (*AvailabilityResult, error) {
	why = strings.TrimSpace(why)
	if why == "" || strings.ContainsAny(why, "\r\n") {
		return nil, fmt.Errorf("parking needs a nonempty, single-line reason")
	}
	return changeAvailability(r, id, StatusDraft, StatusParked, "Park", "parked: "+why, today)
}

// Activate returns a parked plan to Draft; shared eligibility changes after merge.
func Activate(r *Repo, id string, today time.Time) (*AvailabilityResult, error) {
	return changeAvailability(r, id, StatusParked, StatusDraft, "Activate", "activated: operator requested implementation availability", today)
}

func changeAvailability(r *Repo, id, from, to, verb, decision string, today time.Time) (*AvailabilityResult, error) {
	f, err := r.WriteTarget(id)
	if err != nil {
		return nil, err
	}
	branch := r.HeadBranch()
	if !strings.HasPrefix(branch, "plan/") || idListRe.MatchString(branch) || !r.HasHistory() {
		return nil, fmt.Errorf("use a plan/<round> branch with no feature id, based on an updated trunk; then retry kit %s %s", strings.ToLower(verb), id)
	}
	if f.Status != from && f.Status != to {
		return nil, fmt.Errorf("%s is %s — %s requires %s", f.Name(), f.Status, strings.ToLower(verb), from)
	}
	if err := r.PlanError(f); err != nil {
		return nil, err
	}
	if f.FilenamePrefix() != f.ID {
		return nil, fmt.Errorf("%s: filename and id must match", f.Rel)
	}
	if unwritten := f.Unwritten(); len(unwritten) > 0 {
		return nil, fmt.Errorf("%s has no complete plan: ## %s", f.Name(), strings.Join(unwritten, ", ## "))
	}
	if open, total := f.TaskCount(); open != total {
		return nil, fmt.Errorf("%s has completed tasks — parking and activation require unstarted work", f.Name())
	}
	for _, need := range f.Needs {
		if dep := r.FeatureByID(need); dep == nil || dep.Status == StatusDropped {
			return nil, fmt.Errorf("%s needs %s, which is missing or Dropped — repair the plan first", f.Name(), need)
		}
	}
	dirty, err := r.DirtyFiles()
	if err != nil {
		return nil, err
	}
	for _, path := range dirty {
		if path == f.Rel {
			return nil, fmt.Errorf("%s has uncommitted edits — review and commit that file before retrying", f.Rel)
		}
	}
	if !r.TrackedAtHEAD(f.Rel) {
		return nil, fmt.Errorf("commit %s on this planning branch before retrying", f.Rel)
	}
	remote, err := pickRemote(r)
	if err != nil {
		return nil, err
	}
	trunk := r.trunkRef()
	if remote != "" {
		var ok bool
		trunk, ok = r.remoteTip(remote, r.Trunk())
		if r.Offline || !ok {
			return nil, fmt.Errorf("remote state is unknown — reconnect and fetch before changing availability")
		}
	}
	statuses := r.statusesOn(trunk)
	if statuses == nil || !r.reaches(trunk, "HEAD") {
		return nil, fmt.Errorf("integrate the current trunk into this planning branch before changing availability")
	}
	if terminal(statuses[f.Rel]) {
		return nil, fmt.Errorf("%s is terminal history on the trunk — write a new plan", f.Name())
	}
	if claims := r.Claims(id); len(claims) > 0 {
		return nil, fmt.Errorf("%s is claimed by %s — preserve ongoing work; parking does not pause claims", f.Name(), strings.Join(claims, " and "))
	}
	res := &AvailabilityResult{Doc: f.Rel, Status: to}
	if f.Status == to {
		return res, nil
	}
	raw, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}
	updated, ok := SetStatus(Normalise(string(raw)), to)
	if !ok {
		return nil, fmt.Errorf("%s has no status: line to set", f.Rel)
	}
	updated = appendLog(updated, "- "+today.UTC().Format(DateLayout)+" — "+decision)
	if err := replaceFile(f.Path, updated); err != nil {
		return nil, err
	}
	res.Commit = verb + " " + id
	if err := r.commitPaths(res.Commit, f.Path); err != nil {
		return nil, fmt.Errorf("%s is stamped %s, but its commit failed: %w\nreview and commit only %s, then retry; availability still needs human review through merge", f.Rel, to, err, f.Rel)
	}
	return res, nil
}
