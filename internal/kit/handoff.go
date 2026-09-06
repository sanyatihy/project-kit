package kit

import "strings"

// integrationRef uses the advertised default branch online. An unavailable remote
// cannot be replaced with a stale local trunk when deciding what has merged.
func (r *Repo) integrationRef() string {
	remote, err := pickRemote(r)
	if err != nil {
		return ""
	}
	if remote != "" && !r.Offline {
		if tip, ok := r.remoteTip(remote, r.Trunk()); ok {
			return tip
		}
		if _, ok := r.remoteHeads(); !ok {
			return ""
		}
		// A new remote may have no trunk yet; local review remains possible.
		if len(r.heads) != 0 {
			return ""
		}
	}
	return r.trunkRef()
}

func (r *Repo) integratedStatuses() map[string]string {
	if !r.integratedAsked {
		r.integratedAsked = true
		if ref := r.integrationRef(); ref != "" {
			r.integratedDocs = r.statusesOn(ref)
		}
	}
	return r.integratedDocs
}

// foundationProblem is shared by the board and claim. It does not pull a shared
// checkout or merge into somebody else's working tree.
func (r *Repo) foundationProblem() string {
	if !r.foundationAsked {
		r.foundationAsked = true
		r.foundationIssue = r.readFoundationProblem()
	}
	return r.foundationIssue
}

func (r *Repo) readFoundationProblem() string {
	if r.Offline && len(r.remotes()) > 0 {
		return "remote ownership and foundation are unknown while offline — reconnect, fetch, then kit next before claiming"
	}
	if !r.HasHistory() {
		return ""
	}
	ref := r.integrationRef()
	if ref == "" {
		return "cannot read the published trunk — run git fetch, then kit next"
	}
	revs := strings.Fields(r.git("rev-parse", "HEAD", ref))
	if len(revs) != 2 || revs[0] != revs[1] {
		return "checkout is behind, ahead of, or cannot read the published trunk — run git fetch, update a trunk checkout, then kit next"
	}
	return ""
}

func (r *Repo) checkoutHandoff(id string) string {
	f := r.FeatureByID(id)
	if f == nil || !IsSubmitted(f.Status) {
		return ""
	}
	if r.Offline && len(r.remotes()) > 0 {
		return id + " has a local submission; remote submission and integration state are unknown while offline. Reconnect, run git fetch, then kit next before deciding what to push or claim."
	}
	if IsSubmitted(r.integratedStatuses()[f.Rel]) {
		return id + " is integrated. Return to a trunk checkout, run git pull --ff-only, then kit next. Leave this branch intact if it carries additional work."
	}
	if r.integratedStatuses() == nil && len(r.remotes()) > 0 {
		return "integration state is unreadable. Run git fetch, then kit next before deciding whether this submission has merged."
	}
	if files, err := r.DirtyFiles(); err != nil {
		return err.Error()
	} else if len(files) > 0 {
		return id + " has changes after submission. Run kit reopen " + id + " \"<why>\", record the review tasks, and repeat validation before resubmitting."
	}
	remote, err := pickRemote(r)
	if err != nil {
		return err.Error()
	}
	if remote == "" {
		return id + " is ready for local review. No remote is configured; keep the work on this branch and hand it to the reviewer."
	}
	tip, _ := r.remoteTip(remote, r.HeadBranch())
	head := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	if tip != head {
		return id + " has local changes to submit. Run the project gate and kit audit --strict, push this branch, then open or update its pull request."
	}
	return id + " is awaiting review. For requested changes, run kit reopen " + id + " \"<why>\", revise tasks, implement, and repeat the gate, submit and push. After handoff, return to an updated trunk checkout and run kit next to continue independent approved work; this review does not stop the session. After the human merges, update trunk again."
}

// mergedDraftClaim identifies implementation merged before submission without
// marking unchecked work complete or mistaking a fresh claim receipt for a merge.
func (r *Repo) mergedDraftClaim(f *Feature) string {
	open, total := f.TaskCount()
	if f.Status != StatusDraft || open == total {
		return ""
	}
	ref := r.integrationRef()
	if ref == "" {
		return ""
	}
	for _, branch := range r.Claims(f.ID) {
		if rev := r.claimRev(branch); rev != "" && r.reaches(rev, ref) {
			return branch
		}
	}
	return ""
}
