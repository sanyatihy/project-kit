package kit

import (
	"fmt"
	"strings"
)

// NotYours reports that a feature is not this agent's to take: the push lost the
// race, a branch already claims it, an open question blocks it, or the doc is in a
// state nobody may claim.
//
// It is a separate type from an ordinary error because it is a normal answer and
// not a broken invocation. An agent that gets one moves to the next feature; an
// agent that gets an ordinary error has something to fix before it can move at all.
type NotYours struct {
	Why  string
	Then string
}

func (e *NotYours) Error() string { return e.Why }

// Claimed is a claim that was taken.
type Claimed struct {
	Branch string
	Remote string // "" when the project has no remote and the claim is local only
}

// ClaimFeature takes a feature by creating the branch that claims it and pushing
// that branch to the remote.
//
// The push is the claim, and it is the only part of this that is atomic. Every
// check before it is a courtesy that produces a better message in the common case;
// none of them can close the window, because another agent can act between the
// check and the push. What closes it is the remote: a ref update is a
// compare-and-swap against the value the server advertised, so of any number of
// simultaneous claimants exactly one creates the ref.
//
// Nothing local is written until the push has been won, so losing leaves no branch
// to clean up and no cleanup step that can fail on its own.
func ClaimFeature(r *Repo, id string) (*Claimed, error) {
	var target *Feature
	sharing := 0
	for _, f := range r.Features {
		if f.ID == id {
			target, sharing = f, sharing+1
		}
	}
	switch {
	case target == nil:
		return nil, fmt.Errorf("no feature %s", id)
	// One branch cannot claim one of two features, which is the whole reason
	// duplicate-id is HIGH. Refusing is the only honest answer available here.
	case sharing > 1:
		return nil, fmt.Errorf("%d feature docs share id %s, so a branch named after "+
			"it would claim both — run kit audit and renumber all but one", sharing, id)
	}

	switch target.Status {
	case StatusAccepted, StatusInProgress:
	case StatusDraft:
		return nil, &NotYours{target.Name() + " is a Draft — nobody has settled the design",
			"settle it and set status: Accepted, then claim it"}
	case "":
		return nil, &NotYours{target.Name() + " has no status:, so there is no work to claim",
			"give it one: " + strings.Join(AllStatuses, ", ")}
	default:
		return nil, &NotYours{
			fmt.Sprintf("%s is %s — it is not open work", target.Name(), target.Status),
			"claim something kit next offers"}
	}

	if blockers := r.BlockersFor(id); len(blockers) > 0 {
		return nil, &NotYours{
			fmt.Sprintf("%s is blocked: %s", target.Name(), clip(blockers[0], 70)),
			"only a human can answer that, in " + r.Rel(r.Inbox)}
	}

	// Cheap and fallible, and worth doing anyway: a claim already visible locally
	// gets a message naming the branch instead of a push rejection naming a ref.
	if claims := r.Claims(id); len(claims) > 0 {
		return nil, &NotYours{
			fmt.Sprintf("%s is already claimed by %s", target.Name(), strings.Join(claims, " and ")),
			"the branch is the claim — fetch first if you think it is stale"}
	}

	branch := target.Branch()
	if !r.HasHistory() {
		return nil, fmt.Errorf("this project has no commits, so a branch has nothing " +
			"to point at — commit first, then claim")
	}

	remotes := r.remotes()
	remote := ""
	switch {
	case len(remotes) == 0:
		// A project with no remote is a real and normal thing, and refusing would
		// make the kit unusable before the first push. The claim still happens; the
		// caller says plainly that nobody else can see it.
	case contains(remotes, "origin"):
		remote = "origin"
	case len(remotes) == 1:
		remote = remotes[0]
	default:
		return nil, fmt.Errorf("this project has several remotes and no origin (%s), "+
			"and a claim must not guess which one other agents read",
			strings.Join(remotes, ", "))
	}

	if remote != "" {
		ref := "refs/heads/" + branch
		out, pushErr := r.gitRun("push", "--porcelain",
			"--force-with-lease="+ref+":", remote, "HEAD:"+ref)
		switch flag := pushFlag(out, ref); {
		case flag == pushCreated:
		case flag != "":
			return nil, &NotYours{
				fmt.Sprintf("%s already has %s — another agent claimed %s first",
					remote, branch, target.Name()),
				"run kit next and take something else"}
		// The push said nothing about our ref, so nothing is known about who owns it.
		// This must never be reported as a lost race: an agent told it lost moves on
		// for good, and a dropped connection or a rejected credential would quietly
		// retire a feature that nobody actually has.
		case pushErr != nil:
			return nil, pushErr
		default:
			return nil, fmt.Errorf("git push reported nothing about %s:\n%s", ref, out)
		}
	}

	if _, err := r.gitRun("branch", branch, "HEAD"); err != nil {
		return nil, err
	}
	if remote != "" {
		// Best effort. The claim is the ref on the remote and it is already won; a
		// missing upstream is a convenience this should not fail a claim over.
		_, _ = r.gitRun("branch", "--set-upstream-to="+remote+"/"+branch, branch)
	}
	return &Claimed{Branch: branch, Remote: remote}, nil
}

// Status flags from `git push --porcelain`, which prefixes each ref's line with
// one. Only a created ref is a won claim.
const pushCreated = "*"

// pushFlag reads `git push --porcelain` output and returns the status flag for one
// ref: "*" created it, "=" it was already there, "!" rejected, " " updated, and ""
// when the push never mentioned the ref at all.
//
// The flag is read instead of the exit code because the two disagree in exactly the
// case that matters. A push to a branch the remote already has, at the commit we
// are pushing, prints "Everything up-to-date" and exits 0 — so an agent trusting
// the exit code concludes it won a claim that someone else made, and two agents
// walk away owning one feature.
func pushFlag(out, ref string) string {
	for _, line := range strings.Split(Normalise(out), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		if _, to, found := strings.Cut(fields[1], ":"); found && to == ref {
			return fields[0]
		}
	}
	return ""
}

// remotes lists the project's git remotes.
func (r *Repo) remotes() []string {
	var out []string
	for _, line := range strings.Split(r.git("remote"), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			out = append(out, name)
		}
	}
	return out
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
