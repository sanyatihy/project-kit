package kit

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Setting work down without giving it up.
//
// Rule 13 described this as four clauses of prose — commit, push, open the pull request
// unfinished, say what remains — which is a sequence an agent has to remember and half of
// which it can get wrong silently. Every other multi-step sequence the protocol described
// has become a command; this is that.

// PauseResult is what a pause did, for reporting and for the handoff.
type PauseResult struct {
	Doc       string
	ID        string
	Branch    string
	Reason    string
	Commit    string // empty when the tree was already clean
	Remote    string // empty when the project has none
	Remains   []string
	PRTitle   string
	Candidate string
}

// PausedReason is the latest recorded pause, empty when the feature is not paused.
//
// Read from `## Log` the way `ParkingReason` reads parking: the entry is already the
// doc's record of what happened to its plan, and a second place to look would be a fact
// with two homes. A resumption is not written — the next commit on the branch is what
// says the work moved, and `next` stops reporting the pause once it does.
func (f *Feature) PausedReason() string {
	reason := ""
	for _, line := range strings.Split(f.Section("Log"), "\n") {
		if _, why, ok := strings.Cut(line, " — paused: "); ok {
			reason = strings.TrimSpace(why)
		}
	}
	return reason
}

// Pause commits what is in the tree, records the reason, and publishes the branch.
//
// It is not `release`: the claim is kept, so nobody else is invited in and the checkout
// stays the holder. What it does not do is submit — paused work is unfinished by
// definition, and a submission would tell a reviewer the opposite.
func Pause(r *Repo, why string, today time.Time) (*PauseResult, error) {
	why = strings.Join(strings.Fields(why), " ")
	if why == "" {
		return nil, fmt.Errorf("say what remains — a pause nobody can read is a branch nobody will come back to")
	}
	id := r.HeldClaim()
	if id == "" {
		return nil, fmt.Errorf("this checkout claims no feature, so there is nothing to pause\nrun kit next to see what it is holding")
	}
	target, err := r.WriteTarget(id)
	if err != nil {
		return nil, err
	}
	if terminal(target.Status) {
		return nil, fmt.Errorf("%s is %s — finished work is not paused work", target.Name(), target.Status)
	}

	res := &PauseResult{Doc: target.Rel, ID: id, Branch: r.HeadBranch(), Reason: why,
		PRTitle: target.Title, Remains: target.OpenTasks()}

	raw, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	updated := appendLog(Normalise(string(raw)),
		"- "+today.UTC().Format(DateLayout)+" — paused: "+why)
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}

	// Everything in the tree, not only the doc. The point of a pause is that work in
	// progress survives the agent that was doing it, and `commitPaths` would leave
	// exactly the half-finished files this exists to preserve.
	res.Commit = "Pause " + id + ": " + clip(why, 60)
	if err := r.commitAll(res.Commit); err != nil {
		return nil, fmt.Errorf("%s records the pause in %s, and committing it failed:\n%v\n"+
			"commit and push the branch yourself — an unpushed pause is work only this clone has",
			target.Name(), target.Rel, err)
	}
	res.Candidate = strings.TrimSpace(r.git("rev-parse", "HEAD"))

	remote, err := pickRemote(r)
	if err != nil {
		return res, fmt.Errorf("%s is committed at %s, and which remote to publish to is "+
			"ambiguous:\n%v\npush the branch yourself — the work is safe here and nowhere else",
			target.Name(), res.Candidate, err)
	}
	// A project with no remote pauses fine; it just cannot publish. Reported rather than
	// failed, because the commit is the part that protects the work.
	if remote == "" {
		return res, nil
	}
	res.Remote = remote
	if _, err := r.gitRun("push", "--porcelain", remote, "HEAD:refs/heads/"+res.Branch); err != nil {
		return res, fmt.Errorf("%s is committed at %s and the push failed:\n%v\n"+
			"push the branch yourself — the work is safe here and nowhere else",
			target.Name(), res.Candidate, err)
	}
	fresh, err := Open(r.Root)
	if err != nil {
		return res, err
	}
	if tip, ok := fresh.remoteTip(remote, res.Branch); !ok || tip != res.Candidate {
		return res, fmt.Errorf("cannot confirm the push of %s; run kit pause again", res.Candidate)
	}
	return res, nil
}
