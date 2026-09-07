package kit

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// Handoff reports git evidence, never inferred PR creation or human approval.
type Handoff struct {
	PRTitle    string `json:"pr_title"`
	Base       string `json:"base"`
	BaseCommit string `json:"base_commit"`
	Candidate  string `json:"candidate"`
	Branch     string `json:"branch"`
	Checkout   string `json:"checkout"`
	Published  bool   `json:"published"`
	Next       string `json:"next"`
}

// Finish binds gates and review publication to the candidate actually checked.
// The caller executes argv directly; failed or mutating gates stop publication.
func Finish(templates fs.FS, r *Repo, gate []string, runGate func([]string) error, protocol string, today time.Time) (*Handoff, error) {
	branch := r.HeadBranch()
	if strings.HasPrefix(branch, "plan/") {
		if len(gate) > 0 {
			return nil, fmt.Errorf("planning finish takes no implementation gate")
		}
		return finishPlan(r, protocol, today)
	}
	if len(gate) == 0 {
		return nil, fmt.Errorf("work finish requires an actual gate: kit finish -- <gate argv>")
	}
	id := r.HeldClaim()
	if id == "" {
		intent, err := r.WorkIntent()
		if err != nil {
			return nil, err
		}
		if intent == nil || intent.Mode != "work" || branch != "spike/"+Slug(intent.Outcome) {
			return nil, fmt.Errorf("no recorded direct-work intent; use kit start --work before implementing, or harvest an existing spike")
		}
		id, err = finishHarvest(r, intent, today)
		if err != nil {
			return nil, err
		}
		r, err = Open(r.Root)
		if err != nil {
			return nil, err
		}
	}
	f, err := r.WriteTarget(id)
	if err != nil {
		return nil, err
	}
	if !r.Mine(id) {
		return nil, fmt.Errorf("finish %s from its owning feature checkout", id)
	}
	if IsSubmitted(r.integratedStatuses()[f.Rel]) {
		return nil, fmt.Errorf("%s is integrated; return to an updated trunk checkout and run kit next", id)
	}
	if f.Status != StatusDraft && !IsSubmitted(f.Status) {
		return nil, fmt.Errorf("%s is %s; it is not work to finish", id, f.Status)
	}
	if err := r.PlanError(f); err != nil {
		return nil, err
	}
	if sections := f.Unwritten(); len(sections) > 0 {
		return nil, fmt.Errorf("fill review facts in %s: ## %s; commit them, then retry kit finish with the gate", f.Rel, strings.Join(sections, ", ## "))
	}
	if open, _ := f.TaskCount(); open > 0 {
		return nil, fmt.Errorf("%s still has unfinished tasks: %s", id, f.NextTask())
	}
	if blocked := r.BlockersFor(id); len(blocked) > 0 {
		return nil, fmt.Errorf("%s is blocked: %s", id, blocked[0])
	}
	if needs := r.UnmetNeeds(f); len(needs) > 0 {
		return nil, fmt.Errorf("%s still needs integrated prerequisites: %s", id, strings.Join(needs, ", "))
	}
	if err := cleanCandidate(r); err != nil {
		return nil, err
	}
	if IsSubmitted(f.Status) {
		stamp, err := r.gitRun("log", "-1", "--format=%H", "-G", "^status: (Submitted|Shipped)", "--", f.Rel)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(stamp) != strings.TrimSpace(r.git("rev-parse", "HEAD")) {
			return nil, fmt.Errorf("changes follow submission; run kit reopen %s before revising and finishing again", id)
		}
	}
	candidate := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	branch = r.HeadBranch()
	if err := runGate(gate); err != nil {
		return nil, fmt.Errorf("gate failed; no submission or push: %w", err)
	}
	r, err = Open(r.Root)
	if err != nil {
		return nil, err
	}
	if r.HeadBranch() != branch || strings.TrimSpace(r.git("rev-parse", "HEAD")) != candidate {
		return nil, fmt.Errorf("gate changed the candidate or branch; review and commit the changes, then rerun kit finish")
	}
	if err := cleanCandidate(r); err != nil {
		return nil, fmt.Errorf("gate changed the working tree: %w", err)
	}
	if err := finishAudit(r, protocol, today, f.Rel); err != nil {
		return nil, err
	}
	fresh, err := Open(r.Root)
	if err != nil {
		return nil, err
	}
	if fresh.HeadBranch() != branch || strings.TrimSpace(fresh.git("rev-parse", "HEAD")) != candidate {
		return nil, fmt.Errorf("candidate changed during audit; rerun finish before submission")
	}
	if err := cleanCandidate(fresh); err != nil {
		return nil, err
	}
	r = fresh
	if !IsSubmitted(f.Status) {
		if _, err := Submit(templates, r, id, today); err != nil {
			return nil, err
		}
		r, err = Open(r.Root)
		if err != nil {
			return nil, err
		}
		stamped := r.FeatureByID(id)
		if stamped == nil || stamped.Status != StatusSubmitted || stamped.Submitted.UTC().Format(DateLayout) != today.UTC().Format(DateLayout) || !r.onlySubmissionStamp(candidate, f.Rel) {
			return nil, fmt.Errorf("submission changed the validated candidate; preserve its commits, run kit reopen %s, then rerun finish", id)
		}
	}
	h, err := publishHandoff(r, f.Title)
	if err != nil {
		return nil, err
	}
	if h.Published {
		fresh, err := Open(r.Root)
		if err != nil {
			return nil, err
		}
		if err := finishAudit(fresh, protocol, today, ""); err != nil {
			return nil, fmt.Errorf("candidate pushed but final audit is incomplete: %w", err)
		}
	}
	return h, nil
}

func cleanCandidate(r *Repo) error {
	files, err := r.DirtyFiles()
	if err != nil {
		return err
	}
	if len(files) > 0 {
		return fmt.Errorf("uncommitted files remain: %s; commit them before finishing", strings.Join(files, ", "))
	}
	return nil
}

func finishAudit(r *Repo, protocol string, today time.Time, pendingPublication string) error {
	findings := Audit(r, today, protocol)
	if r.RemoteUnanswered() {
		return fmt.Errorf("online audit is incomplete; reconnect and retry kit finish")
	}
	for _, f := range findings {
		if f.Severity == High {
			// Publication is the next stage; this finding is discharged only by a verified push.
			if f.Code == checkSubmittedNotPushed.Code && f.Where == pendingPublication {
				continue
			}
			return fmt.Errorf("audit %s: %s; %s", f.Code, f.Problem, f.Fix)
		}
	}
	return nil
}

func finishPlan(r *Repo, protocol string, today time.Time) (*Handoff, error) {
	intent, err := r.WorkIntent()
	if err != nil {
		return nil, err
	}
	if intent == nil || intent.Mode != "plan" || r.HeadBranch() != "plan/"+Slug(intent.Outcome) {
		return nil, fmt.Errorf("planning branch has no matching start intent; start a reviewed round with kit start --plan")
	}
	changed, err := r.gitRun("diff", "--name-only", "-z", intent.Base, "--")
	if err != nil {
		return nil, err
	}
	dirty, err := r.DirtyFiles()
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for _, p := range append(gitPaths(changed), dirty...) {
		paths[p] = true
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("planning round has no changes; write its feature docs first")
	}
	initial := r.isReviewBase(intent.Base)
	for path := range paths {
		if !planningPath(path, initial, r.Root) {
			return nil, fmt.Errorf("planning finish refuses unrelated file %s; preserve it on its own work branch", path)
		}
	}
	plans := 0
	for _, f := range r.Features {
		if paths[f.Rel] {
			if f.Status != StatusDraft && f.Status != StatusParked {
				return nil, fmt.Errorf("planning finish requires Draft or Parked plans; %s is %s", f.ID, f.Status)
			}
			if err := r.PlanError(f); err != nil {
				return nil, err
			}
			if sections := f.Unwritten(); len(sections) > 0 {
				return nil, fmt.Errorf("write ## %s in %s before finishing", strings.Join(sections, ", ## "), f.Rel)
			}
			if open, total := f.TaskCount(); open != total {
				return nil, fmt.Errorf("%s already has completed tasks; planning approval must precede implementation", f.ID)
			}
			plans++
		}
	}
	if plans == 0 {
		return nil, fmt.Errorf("planning finish needs at least one complete feature plan")
	}
	if len(dirty) > 0 {
		absolute := make([]string, 0, len(dirty))
		for _, path := range dirty {
			absolute = append(absolute, filepath.Join(r.Root, path))
		}
		if err := r.commitPaths("Plan "+intent.Outcome, absolute...); err != nil {
			return nil, fmt.Errorf("planning commit failed; files are preserved: %w", err)
		}
		r, err = Open(r.Root)
		if err != nil {
			return nil, err
		}
	}
	if err := finishAudit(r, protocol, today, ""); err != nil {
		return nil, err
	}
	return publishHandoff(r, "Plan: "+intent.Outcome)
}

func planningPath(path string, initial bool, root string) bool {
	if strings.HasPrefix(path, "docs/") && strings.HasSuffix(path, ".md") {
		return true
	}
	switch path {
	case "AGENTS.md", "CLAUDE.md", "README.md", ".gitignore":
		return true
	}
	if !initial {
		return false
	}
	values := map[string]string{"name": filepath.Base(root), "pkg": PackageName(filepath.Base(root))}
	for _, stack := range stackLayout {
		for _, pair := range stack.files {
			if path == fill(pair[1], values) {
				return true
			}
		}
		if path == stack.keep+"/.gitkeep" {
			return true
		}
	}
	for _, forges := range ciLayout {
		for _, pair := range forges {
			if path == pair[1] {
				return true
			}
		}
	}
	return false
}

func publishHandoff(r *Repo, title string) (*Handoff, error) {
	if err := cleanCandidate(r); err != nil {
		return nil, err
	}
	remote, err := pickRemote(r)
	if err != nil {
		return nil, err
	}
	branch := r.HeadBranch()
	base := r.Trunk()
	candidate := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	h := &Handoff{PRTitle: title, Base: base, BaseCommit: strings.TrimSpace(r.git("rev-parse", r.trunkRef())), Candidate: candidate, Branch: branch, Checkout: r.Root}
	if remote == "" {
		h.Next = "agent: add the intended remote and retry finish; PR review is pending, not complete"
		return h, nil
	}
	if _, ok := r.remoteHeads(); !ok {
		return nil, fmt.Errorf("publication is unknown; reconnect and retry kit finish")
	}
	if _, exists := r.remoteTip(remote, base); !exists {
		ref := r.trunkRef()
		if !r.isReviewBase(ref) {
			return nil, fmt.Errorf("remote review base is missing; publish the reviewed trunk before finishing")
		}
		baseTip := strings.TrimSpace(r.git("rev-parse", ref))
		remoteRef := "refs/heads/" + base
		if _, err := r.gitRun("push", "--porcelain", "--force-with-lease="+remoteRef+":", remote, baseTip+":"+remoteRef); err != nil {
			return nil, fmt.Errorf("review base publication is unconfirmed; retry finish: %w", err)
		}
	}
	if tip, ok := r.remoteTip(remote, branch); ok && tip != candidate && !r.reaches(tip, candidate) {
		return nil, fmt.Errorf("remote candidate differs; fetch and reconcile its work before retrying finish")
	}
	if _, err := r.gitRun("push", "--porcelain", remote, candidate+":refs/heads/"+branch); err != nil {
		return nil, fmt.Errorf("candidate %s preserved; push is unconfirmed — retry finish: %w", candidate, err)
	}
	fresh, err := Open(r.Root)
	if err != nil {
		return nil, err
	}
	if tip, ok := fresh.remoteTip(remote, branch); !ok || tip != candidate {
		return nil, fmt.Errorf("cannot confirm publication of candidate %s; retry finish", candidate)
	}
	if fresh.HeadBranch() != branch || strings.TrimSpace(fresh.git("rev-parse", "HEAD")) != candidate {
		return nil, fmt.Errorf("local candidate changed during publication; run finish again before review")
	}
	baseTip, ok := fresh.remoteTip(remote, base)
	if !ok {
		return nil, fmt.Errorf("review base is unknown after publication; reconnect and retry finish")
	}
	h.BaseCommit = baseTip
	h.Published = true
	h.Next = "agent: open or update the PR for this base and candidate using the available forge interface. Set its title explicitly from pr_title, not commit autofill. Human: approve and merge. Then agent: return to an updated trunk checkout, run kit next, and continue independent approved work."
	return h, nil
}

// finishHarvest reuses a doc written before an interrupted branch rename or commit.
func finishHarvest(r *Repo, intent *Intent, today time.Time) (string, error) {
	baseline := r.statusesOn(intent.Base)
	if baseline == nil {
		return "", fmt.Errorf("recorded work base is unreadable; fetch before recovering harvest")
	}
	var pending *Feature
	for _, f := range r.Features {
		if Slug(f.Title) != Slug(intent.Outcome) || baseline[f.Rel] != "" {
			continue
		}
		if pending != nil {
			return "", fmt.Errorf("multiple documents record this work; resolve ambiguity before finishing")
		}
		pending = f
	}
	if pending == nil {
		res, err := Harvest(r, intent.Outcome, []string{intent.Outcome}, today)
		if err != nil {
			return "", err
		}
		return res.ID, nil
	}
	if _, err := r.WriteTarget(pending.ID); err != nil {
		return "", err
	}
	if pending.Status != StatusDraft {
		return "", fmt.Errorf("preserved harvest %s is not Draft", pending.ID)
	}
	if claims := r.Claims(pending.ID); len(claims) > 0 {
		return "", fmt.Errorf("preserved harvest is claimed by %s; resume its owner checkout", strings.Join(claims, ", "))
	}
	if _, err := r.gitRun("branch", "-m", pending.Branch()); err != nil {
		return "", err
	}
	if files, err := r.DirtyFiles(); err != nil {
		return "", err
	} else if len(files) > 0 {
		if err := r.commitAll("Harvest " + pending.ID); err != nil {
			return "", fmt.Errorf("harvest files preserved; commit them and retry finish: %w", err)
		}
	}
	return pending.ID, nil
}

// Submission may change only its lifecycle fields; hooks cannot smuggle new work
// between a passed gate and publication, even by amending the stamping commit.
func (r *Repo) onlySubmissionStamp(candidate, path string) bool {
	parent, err := r.gitRun("rev-parse", "HEAD^")
	if err != nil || strings.TrimSpace(parent) != candidate {
		return false
	}
	changed, err := r.gitRun("diff", "--name-only", "-z", candidate, "HEAD", "--")
	if err != nil {
		return false
	}
	paths := gitPaths(changed)
	if len(paths) != 1 || paths[0] != path {
		return false
	}
	before, err := r.gitRun("show", candidate+":"+path)
	if err != nil {
		return false
	}
	after, err := r.gitRun("show", "HEAD:"+path)
	if err != nil {
		return false
	}
	for _, key := range []string{"status", "submitted", "shipped"} {
		before = RemoveField(Normalise(before), key)
		after = RemoveField(Normalise(after), key)
	}
	return before == after
}
