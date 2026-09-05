package kit

import (
	"fmt"
	"strings"
)

// Released is a claim that was let go.
type Released struct {
	Branch string
	Remote string // the remote the claim was deleted from, "" when the claim was local only
	Kept   string // where the local branch still is, when deleting it could not run
}

// Release surrenders a claim. Without force, every unmerged commit and every
// uncommitted file is work to preserve. The remote deletion leases its observed tip.
func Release(r *Repo, id string, force bool) (*Released, error) {
	target, err := r.WriteTarget(id)
	if err != nil {
		return nil, err
	}

	claims := r.Claims(id)
	switch {
	case len(claims) == 0:
		return nil, fmt.Errorf("nothing claims %s, so there is no claim to release", target.Name())
	// Two branches citing one id is `split-claim`, and choosing between them here would
	// delete one agent's work to tidy up after another's.
	case len(claims) > 1:
		return nil, fmt.Errorf("%s is claimed by %s, and this cannot tell which is the "+
			"claim\nsettle that first — kit audit reports it as split-claim",
			target.Name(), strings.Join(claims, " and "))
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
			target.Name(), branch, id)
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
	work := false
	if !force {
		unmerged, err := r.gitRun("rev-list", "--count", r.trunkRef()+".."+rev)
		if err != nil {
			return nil, err
		}
		work = strings.TrimSpace(unmerged) != "0"
		if path := r.worktreeFor(branch); path != "" {
			checkout := &Repo{Root: path}
			dirty, err := checkout.DirtyFiles()
			if err != nil {
				return nil, err
			}
			if len(dirty) > 0 {
				return nil, fmt.Errorf("%s has uncommitted work in %s — commit or preserve it before releasing", branch, path)
			}
		}
	}
	if work {
		return nil, fmt.Errorf("%s carries work that is not on %s, and releasing deletes "+
			"it\nthat is a pause rather than an abandon: push the branch, open the pull "+
			"request unfinished, and say why there\nif those commits are worth nothing: "+
			"kit release %s --force", branch, orTrunk(r.Trunk()), id)
	}

	res := &Released{Branch: branch}
	if remote != "" {
		_, ok := r.remoteHeads()
		if !ok {
			return nil, fmt.Errorf("%s did not answer, and the branch there is the claim "+
				"— release again when it does", remote)
		}
		if tip, exists := r.remoteTip(remote, branch); exists {
			ref := "refs/heads/" + branch
			if _, err := r.gitRun("push", "--force-with-lease="+ref+":"+tip,
				remote, ":"+ref); err != nil {
				return nil, err
			}
			res.Remote = remote
		}
	}

	// `claim` creates the worktree, so removing it is the other half of that and not a
	// courtesy: without this every release would find the branch checked out and could
	// never delete it. Only the one the kit made at the path it chooses — a worktree an
	// operator put somewhere else is theirs, and is reported below rather than removed.
	if own := r.WorktreePath(branch); r.worktreeBranch(own) == branch {
		if _, err := r.gitRun("worktree", "remove", own); err != nil {
			res.Kept = own
		}
	}

	// The local branch is a copy, and by here the claim itself is already released, so
	// failing to delete it must not fail the command. It fails when the worktree above
	// could not go: an agent releasing from the checkout it was working in.
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
