package kit

import (
	"crypto/rand"
	"fmt"
	"strings"
)

func confirmedRef(id string) string { return "refs/kit/owned/" + id }

func attemptRef(id string) string { return "refs/kit/claims/" + id }

func (r *Repo) attempt(id string) string {
	return strings.TrimSpace(r.git("rev-parse", "--verify", "--quiet", attemptRef(id)))
}

// otherClaims omits only this clone's recorded attempt; a local branch alone is
// not evidence that a previous invocation created it.
func (r *Repo) otherClaims(id, branch string) []string {
	claims := r.Claims(id)
	if r.attempt(id) == "" {
		return claims
	}
	var others []string
	for _, claim := range claims {
		if claim != branch {
			others = append(others, claim)
		}
	}
	return others
}

func (r *Repo) prepareClaim(f *Feature) (string, error) {
	branch, ref := f.ClaimBranch(), attemptRef(f.ID)
	attempt := r.attempt(f.ID)
	if attempt == "" {
		if r.hasRef("refs/heads/" + branch) {
			return "", fmt.Errorf("%s exists without a claim receipt — inspect that local branch before claiming", branch)
		}
		tree, err := r.gitRun("rev-parse", "HEAD^{tree}")
		if err != nil {
			return "", err
		}
		attempt, err = r.gitRun("commit-tree", strings.TrimSpace(tree), "-p", "HEAD", "-m", "Kit claim "+f.ID+" "+rand.Text())
		if err != nil {
			return "", err
		}
		attempt = strings.TrimSpace(attempt)
		if _, err = r.gitRun("update-ref", ref, attempt, ""); err != nil {
			return "", fmt.Errorf("another local claim attempt started — retry kit claim %s: %w", f.ID, err)
		}
	}
	base := strings.TrimSpace(r.git("rev-parse", attempt+"^"))
	head := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	if base == "" || base != head {
		return "", fmt.Errorf("the retained attempt has a different foundation — preserve any work, run kit release %s, then claim from the updated trunk", f.ID)
	}
	if r.hasRef("refs/heads/" + branch) {
		if strings.TrimSpace(r.git("rev-parse", "refs/heads/"+branch)) != attempt {
			return "", fmt.Errorf("%s carries work beyond its unconfirmed attempt — preserve it before retrying", branch)
		}
	} else if _, err := r.gitRun("branch", branch, attempt); err != nil {
		return "", err
	}
	return attempt, nil
}

// resumeClaim reconciles a receipt with authoritative remote evidence. Unknown
// evidence keeps both the local ref and receipt, making another retry safe.
func (r *Repo) resumeClaim(f *Feature, remote string) (*Claimed, bool, error) {
	attempt := r.attempt(f.ID)
	if attempt == "" {
		return nil, false, nil
	}
	branch := f.ClaimBranch()
	if remote == "" {
		if r.hasRef("refs/heads/" + branch) {
			res, err := r.finishClaim(branch, remote)
			return res, true, err
		}
		return nil, false, nil
	}
	if _, ok := r.remoteHeads(); !ok {
		return nil, true, fmt.Errorf("remote state is unreadable; attempt preserved — retry kit claim %s when it is reachable", f.ID)
	}
	tip, exists := r.remoteTip(remote, branch)
	if !exists {
		return nil, false, nil
	}
	confirmed := strings.TrimSpace(r.git("rev-parse", "--verify", "--quiet", confirmedRef(f.ID))) == attempt
	if confirmed && r.hasRef("refs/heads/"+branch) {
		res, err := r.finishClaim(branch, remote)
		return res, true, err
	}
	if tip != attempt && !r.hasObject(tip) {
		if _, err := r.gitRun("fetch", "--no-tags", remote, "refs/heads/"+branch); err != nil {
			return nil, true, fmt.Errorf("attempt preserved; fetch the claim and retry: %w", err)
		}
	}
	if tip == attempt || r.reaches(attempt, tip) {
		if !r.hasRef("refs/heads/" + branch) {
			if _, err := r.gitRun("branch", branch, tip); err != nil {
				return nil, true, err
			}
		}
		if others := r.otherClaims(f.ID, branch); len(others) > 0 {
			return nil, true, fmt.Errorf("%s also claims %s — settle the split claim before resuming", strings.Join(others, ", "), f.ID)
		}
		res, err := r.finishClaim(branch, remote)
		return res, true, err
	}
	if err := r.forgetAttempt(f.ID, branch, attempt); err != nil {
		return nil, true, err
	}
	return nil, true, &NotYours{branch + " belongs to another claimant", "run kit next and choose other work"}
}

func (r *Repo) hasObject(rev string) bool {
	return strings.TrimSpace(r.git("rev-parse", "--verify", "--quiet", rev+"^{commit}")) != ""
}

// forgetAttempt deletes only the unchanged, unoccupied losing attempt. Valuable
// local changes survive even if somebody else won the remote claim.
func (r *Repo) forgetAttempt(id, branch, attempt string) error {
	if r.hasRef("refs/heads/" + branch) {
		if r.worktreeFor(branch) != "" || strings.TrimSpace(r.git("rev-parse", "refs/heads/"+branch)) != attempt {
			return fmt.Errorf("another claimant won %s; local work preserved — inspect %s before cleaning up the failed attempt", id, branch)
		}
		if _, err := r.gitRun("update-ref", "-d", "refs/heads/"+branch, attempt); err != nil {
			return err
		}
	}
	_, err := r.gitRun("update-ref", "-d", attemptRef(id), attempt)
	return err
}

// claimOnly recognizes the recorded receipt or its empty rebased equivalent.
// The exact unique message must survive; other empty commits still count as work.
func (r *Repo) claimOnly(id, rev string) bool {
	attempt := r.attempt(id)
	if attempt == "" {
		return false
	}
	if rev == attempt {
		return true
	}
	original := strings.TrimSpace(r.git("show", "-s", "--format=%B", attempt))
	candidate := strings.TrimSpace(r.git("show", "-s", "--format=%B", rev))
	if original == "" || original != candidate {
		return false
	}
	parents := strings.Fields(r.git("show", "-s", "--format=%P", rev))
	if len(parents) != 1 {
		return false
	}
	trees := strings.Fields(r.git("rev-parse", rev+"^{tree}", parents[0]+"^{tree}"))
	return len(trees) == 2 && trees[0] == trees[1]
}
