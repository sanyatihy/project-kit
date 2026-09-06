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
	Base      string `json:"base"`
	Candidate string `json:"candidate"`
	Branch    string `json:"branch"`
	Checkout  string `json:"checkout"`
	Published bool   `json:"published"`
	Next      string `json:"next"`
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
		// A partially written harvest is preserved for explicit recovery, not allocated again.
		for _, f := range r.Features {
			if Slug(f.Title) == Slug(intent.Outcome) {
				return nil, fmt.Errorf("%s already records this outcome; preserve it and resume its feature branch before finishing", f.Rel)
			}
		}
		res, err := Harvest(r, intent.Outcome, []string{intent.Outcome}, today)
		if err != nil {
			return nil, err
		}
		id = res.ID
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
	if !IsSubmitted(f.Status) {
		if _, err := Submit(templates, r, id, today); err != nil {
			return nil, err
		}
		r, err = Open(r.Root)
		if err != nil {
			return nil, err
		}
	}
	h, err := publishHandoff(r)
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
			if f.Status != StatusDraft {
				return nil, fmt.Errorf("planning finish requires Draft plans; %s is %s", f.ID, f.Status)
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
	return publishHandoff(r)
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

func publishHandoff(r *Repo) (*Handoff, error) {
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
	h := &Handoff{Base: base, Candidate: candidate, Branch: branch, Checkout: r.Root}
	if remote == "" {
		h.Next = "agent: add the intended remote and retry finish; PR review is pending, not complete"
		return h, nil
	}
	if _, ok := r.remoteHeads(); !ok {
		return nil, fmt.Errorf("publication is unknown; reconnect and retry kit finish")
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
	h.Published = true
	h.Next = "agent: open or update the PR for this base and candidate using the available forge interface. Human: approve and merge. Then agent: return to an updated trunk checkout, run kit next, and continue independent approved work."
	return h, nil
}
