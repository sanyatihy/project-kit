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
	if f == nil || f.Status != StatusShipped {
		return ""
	}
	if r.integratedStatuses()[f.Rel] == StatusShipped {
		return id + " is integrated. Return to a trunk checkout, run git pull --ff-only, then kit next. Leave this branch intact if it carries additional work."
	}
	if r.integratedStatuses() == nil && len(r.remotes()) > 0 {
		return "integration state is unreadable. Run git fetch, then kit next before deciding whether this submission has merged."
	}
	if files, err := r.DirtyFiles(); err != nil {
		return err.Error()
	} else if len(files) > 0 {
		return id + " has changes after submission. Run kit reset " + id + " \"<why>\", record the review tasks, and repeat validation before resubmitting."
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
	return id + " is awaiting review. For requested changes, run kit reset " + id + " \"<why>\", revise tasks, implement, and repeat the gate, ship and push. After the human merges, return to an updated trunk checkout."
}
