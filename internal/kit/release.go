package kit

import (
	"fmt"
	"slices"
	"strings"
)

// Released is a claim that was let go.
type Released struct {
	Branch string
	Remote string // the remote the claim was deleted from, "" when the claim was local only
	Kept   string // where the local branch still is, when deleting it could not run
}

// Release hands a claimed feature back to the board by deleting the branch that claims it.
//
// The remote ref goes first, because that ref is the claim. Deleting only the local copy
// leaves the feature taken for good, which prose could only warn about.
//
// It refuses while the branch carries work, and that refusal is the reason this exists.
// The protocol named `git push --delete` for every kind of stopping, so an agent pausing
// with reviewed commits on the branch followed it and discarded them.
// `BranchBuildsOn` already separates a claim nobody has built on from work in flight, and
// `--force` is the way past it for commits that are genuinely worth nothing.
func Release(r *Repo, id string, force bool) (*Released, error) {
	// Two kinds of work, one claim. What differs is only what the claim branch is
	// allowed to carry: a feature's doc, or the chore's line in the record.
	name, rel := "", ""
	if target := r.FeatureByID(id); target != nil {
		name, rel = target.Name(), target.Rel
	} else if chore := r.ChoreByID(id); chore != nil {
		name, rel = chore.Name(), ChoreLog
	} else {
		return nil, fmt.Errorf("no feature or chore %s", id)
	}

	claims := r.Claims(id)
	switch {
	case len(claims) == 0:
		return nil, fmt.Errorf("nothing claims %s, so there is no claim to release", name)
	// Two branches citing one id is `split-claim`, and choosing between them here would
	// delete one agent's work to tidy up after another's.
	case len(claims) > 1:
		return nil, fmt.Errorf("%s is claimed by %s, and this cannot tell which is the "+
			"claim\nsettle that first — kit audit reports it as split-claim",
			name, strings.Join(claims, " and "))
	}
	branch := claims[0]

	// Releasing somebody else's claim frees a feature a live agent is holding, and the
	// board then hands it to a second one. `Ship` has asked this question since 0023 and
	// `Release` did not, which let a losing claimant take the winner's branch by following
	// the line `claim` printed.
	//
	// The test is a local head, not `Mine`. `Mine` asks whether HEAD is the claim branch,
	// which is right for `ship` — shipping happens in the worktree doing the work — and
	// wrong here: `claim` tells you to make that worktree, so between claiming and moving
	// there you would be unable to release what you had just taken. A local ref means this
	// clone created the branch, which is the same answer for every worktree sharing it and
	// a different one for the agent that only ever saw the claim over the wire.
	//
	// `--force` is the way past, because clearing up after a dead agent is real work — the
	// flag is what makes destroying another agent's claim deliberate.
	if !force && !r.hasRef("refs/heads/"+branch) {
		return nil, fmt.Errorf("%s is claimed by %s, a branch this clone never created\n"+
			"releasing it frees a feature another agent may be working on — leave it "+
			"alone,\nor if that agent is gone: kit release %s --force",
			name, branch, id)
	}

	remote, err := pickRemote(r)
	if err != nil {
		return nil, err
	}

	// Whether a branch carries work cannot be read from its name, and "no work" is the
	// answer that deletes it — so a branch this checkout cannot resolve stops here rather
	// than being released out of ignorance.
	rev := r.resolveBranch(branch)
	if rev == "" {
		return nil, fmt.Errorf("%s is a branch this checkout cannot resolve, so whether it "+
			"carries work cannot be read\nrun git fetch, then release", branch)
	}
	// The ref this deletes lives on the remote, and every copy of it here is a cache: a
	// local head records the last push and `refs/remotes/*` the last fetch. A checkout
	// left behind reads "carries no work" for a branch another checkout has pushed
	// commits to, which is how a release at exit 0 made reviewed work unreachable.
	//
	// Containment rather than equality, because being ahead of the remote is the ordinary
	// state of an agent that has committed and not pushed. Once the remote's tip is an
	// ancestor of what is asked, asking the local copy answers for the remote's commits
	// too. Anything else — behind, diverged, or an object this clone has never had — is
	// unreadable, and unreadable must not be the permissive answer in a command that
	// deletes.
	if !force && remote != "" {
		if tip, ok := r.remoteTip(remote, branch); ok && !r.reaches(tip, rev) {
			return nil, fmt.Errorf("%s/%s is not the copy of %s this checkout has, so what "+
				"the remote carries cannot be read — and it is the remote's branch this "+
				"deletes\nrun git fetch, then release", remote, branch, branch)
		}
	}
	// The same reasoning one level up: with no trunk to diff against, `BranchBuildsOn`
	// answers "carries no work" for a branch full of commits. Silence must not be the
	// permissive answer in the one command that destroys.
	if !force && r.trunkRef() == "" {
		return nil, fmt.Errorf("no default branch resolves here, so whether %s carries work "+
			"cannot be read\nname it: git remote set-head origin --auto\nor if those commits "+
			"are worth nothing: kit release %s --force", branch, id)
	}
	if !force && r.BranchBuildsOn(rev, rel) {
		return nil, fmt.Errorf("%s carries work that is not on %s, and releasing deletes "+
			"it\nthat is a pause rather than an abandon: push the branch, open the pull "+
			"request unfinished, and say why there\nif those commits are worth nothing: "+
			"kit release %s --force", branch, orTrunk(r.Trunk()), id)
	}

	res := &Released{Branch: branch}
	if remote != "" {
		heads, ok := r.remoteHeads()
		if !ok {
			return nil, fmt.Errorf("%s did not answer, and the branch there is the claim "+
				"— release again when it does", remote)
		}
		if slices.Contains(heads, branch) {
			if _, err := r.gitRun("push", remote, "--delete", branch); err != nil {
				return nil, err
			}
			res.Remote = remote
		}
	}

	// The local branch is a copy, and by here the claim itself is already released, so
	// failing to delete it must not fail the command. It fails in the ordinary case: an
	// agent releasing from the worktree it was working in is standing on the branch.
	if r.hasRef("refs/heads/" + branch) {
		if _, err := r.gitRun("branch", "-D", branch); err != nil {
			if res.Kept = r.worktreeFor(branch); res.Kept == "" {
				res.Kept = err.Error()
			}
		}
	}
	return res, nil
}

// worktreeFor is the checkout holding a branch, empty when no worktree has it.
func (r *Repo) worktreeFor(branch string) string {
	path := ""
	for _, line := range strings.Split(r.git("worktree", "list", "--porcelain"), "\n") {
		line = strings.TrimSpace(line)
		if rest, found := strings.CutPrefix(line, "worktree "); found {
			path = rest
		}
		if rest, found := strings.CutPrefix(line, "branch "); found &&
			strings.TrimPrefix(rest, "refs/heads/") == branch {
			return path
		}
	}
	return ""
}
