package kit

import (
	"fmt"
	"slices"
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
// The local branch is created first and deleted again if the push does not win, so a
// claim is held here and on the remote or in neither place. The other order left the ref
// on the remote when the local step failed — `git branch` rejects a name git accepts as
// a refspec, `-0007-dashed` for one — and the claim was then nobody's: `claim` called the
// feature taken and `release` called the branch one this clone never created.
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
	// The same key disagreeing with itself, and it breaks the same guarantee. The
	// branch comes from the filename and `Claims` matches the id, so claiming this
	// would create a branch the board cannot see: `next` goes on offering the feature
	// and the next agent gets a raw git error about a branch that already exists.
	// NotYours because one rename fixes it — and because meanwhile the agent is meant
	// to take something else, which is what a lost race says too.
	case target.FilenamePrefix() != "" && target.FilenamePrefix() != target.ID:
		return nil, &NotYours{
			fmt.Sprintf("%s declares id %s but its file is named %s, so the branch "+
				"claiming it would not mention %s", target.Name(), target.ID,
				target.FilenamePrefix(), target.ID),
			"make them match — rename the file, or fix the id: — then claim again"}
	}

	// Draft is the only status work starts from. Shipped and Dropped are end states, and
	// a claim on either reopens finished history. What else refuses a claim is an
	// unwritten plan (below), a BLOCKING inbox line, and an unmet `needs:`.
	switch target.Status {
	case StatusDraft:
	case "":
		return nil, &NotYours{target.Name() + " has no status:, so there is no work to claim",
			"give it one: " + strings.Join(AllStatuses, ", ")}
	default:
		return nil, &NotYours{
			fmt.Sprintf("%s is %s — it is not open work", target.Name(), target.Status),
			"claim something kit next offers"}
	}

	// The load-bearing half of letting a Draft be claimed. `unfilled-template` exempts
	// Draft, and that exemption was only safe while Drafts could not be claimed — without
	// this, `claim` hands an agent a doc straight out of `kit new` with three empty
	// sections and every command still exits 0.
	if unwritten := target.Unwritten(); len(unwritten) > 0 {
		return nil, &NotYours{
			fmt.Sprintf("%s has no plan to build: ## %s never written",
				target.Name(), strings.Join(unwritten, ", ## ")),
			"write it first — a claim on an unwritten doc sends an agent to build nothing"}
	}

	if blockers := r.BlockersFor(id); len(blockers) > 0 {
		return nil, &NotYours{
			fmt.Sprintf("%s is blocked: %s", target.Name(), clip(blockers[0], 70)),
			"only a human can answer that, in " + r.Rel(r.Inbox)}
	}

	// The same call `Survey` makes, so the board and the command cannot disagree. Without
	// it `next` said 0002 was waiting on 0001 and `claim 0002` took it anyway, which sends
	// an agent to build on a foundation that is not there. `NotYours` because the agent is
	// meant to take something else — the same instruction a lost race carries.
	if unmet := r.UnmetNeeds(target); len(unmet) > 0 {
		return nil, &NotYours{
			fmt.Sprintf("%s waits on %s: not shipped yet", target.Name(),
				strings.Join(unmet, ", ")),
			"claim something kit next offers"}
	}

	branch := target.Branch()
	if !r.HasHistory() {
		return nil, fmt.Errorf("this project has no commits, so a branch has nothing " +
			"to point at — commit first, then claim")
	}

	// The claim pushes HEAD under the feature's name, so whatever HEAD does not carry,
	// the claim does not carry either. Without this the push reports a won claim, the
	// worktree it prints is empty, and `kit next` there says the project is not set up —
	// every step exiting 0 while the agent loses the plan it was sent to build.
	if !r.TrackedAtHEAD(target.Rel) {
		return nil, fmt.Errorf("%s is not committed on %s, so the branch this would push "+
			"carries no plan.\nLand the plan first — a pull or merge request, if the default branch "+
			"is protected — then claim", target.Rel, orTrunk(r.Trunk()))
	}

	// The trunk is where a claim has to branch from, and the check above does not say so:
	// a doc committed on the plan branch is in HEAD's tree, so the claim passes and the
	// branch it pushes is a child of that plan branch. Integrating any of it then means
	// merging into the branch its own author coordinated. That is how four features came
	// to be merged by their builder with no branch ever pushed for review.
	//
	// An ordinary error rather than NotYours: nothing on an unlanded round is claimable,
	// so an agent told to take something else meets the same wall at the next id.
	if !r.PlanLanded(target) {
		return nil, fmt.Errorf("%s is not on %s, so this claim would branch off the plan "+
			"round instead of the trunk, and only its own author could merge it.\n"+
			"Land the round first — a pull or merge request, if the default branch is "+
			"protected — then claim", target.Rel, r.Trunk())
	}

	remote, err := pickRemote(r)
	if err != nil {
		return nil, err
	}

	// Cheap and fallible, and worth doing anyway: a claim already visible locally
	// gets a message naming the branch instead of a push rejection naming a ref.
	if claims := r.Claims(id); len(claims) > 0 {
		// The branch that held a draft is named after the doc, which is also the
		// claim. Before the draft merges the two differ — a draft branch carries the
		// doc, a fresh claim carries nothing — but afterwards both are identical to
		// the trunk, so nothing here can tell them apart. Saying which fixes apply
		// beats reporting a claim that may have nobody behind it.
		then := "the branch is the claim"
		switch {
		case !r.RefsAreAuthoritative():
			then = "this was read from the last fetch, not from the remote, so it may name a\n" +
				"  branch that no longer exists — run git fetch --prune, then claim again"
		case len(claims) == 1 && !r.BranchBuildsOn(claims[0], target.Rel):
			// A branch carrying no work is either an agent that just claimed it or a
			// leftover from a merged draft, and nothing here can tell those apart. The
			// way out is `kit release`, which checks ownership and refuses over work;
			// naming the raw git delete instead handed a losing claimant the one command
			// that takes the winner's branch.
			then = "that branch carries no work. If an agent just took it, leave it alone.\n" +
				"  If it is a leftover from a draft that already merged, end the claim with\n" +
				"  kit release " + target.ID + " — it refuses if the branch turns out to carry work"
		}
		return nil, &NotYours{
			fmt.Sprintf("%s is already claimed by %s", target.Name(), strings.Join(claims, " and ")),
			then}
	}

	if _, err := r.gitRun("branch", branch, "HEAD"); err != nil {
		return nil, err
	}

	if remote != "" {
		ref := "refs/heads/" + branch
		out, pushErr := r.gitRun("push", "--porcelain",
			"--force-with-lease="+ref+":", remote, "HEAD:"+ref)
		switch flag := pushFlag(out, ref); {
		case flag == pushCreated:
		case flag != "":
			return nil, r.unclaim(branch, &NotYours{
				fmt.Sprintf("%s already has %s — another agent claimed %s first",
					remote, branch, target.Name()),
				"run kit next and take something else"})
		// The push said nothing about our ref, so nothing is known about who owns it.
		// This must never be reported as a lost race: an agent told it lost moves on
		// for good, and a dropped connection or a rejected credential would quietly
		// retire a feature that nobody actually has.
		case pushErr != nil:
			return nil, r.unclaim(branch, pushErr)
		default:
			return nil, r.unclaim(branch,
				fmt.Errorf("git push reported nothing about %s:\n%s", ref, out))
		}
		// Best effort. The claim is the ref on the remote and it is already won; a
		// missing upstream is a convenience this should not fail a claim over.
		_, _ = r.gitRun("branch", "--set-upstream-to="+remote+"/"+branch, branch)
	}
	return &Claimed{Branch: branch, Remote: remote}, nil
}

// unclaim deletes the branch a claim created and hands back the reason it is stopping,
// so a claim nobody won leaves nothing behind that reads as one.
//
// The delete's own failure is not reported: nothing is checked out on a branch made a
// moment ago, and the caller's reason for stopping is the one worth printing.
func (r *Repo) unclaim(branch string, cause error) error {
	_, _ = r.gitRun("branch", "-D", branch)
	return cause
}

// pickRemote names the remote a claim is made on and released from.
//
// A project with no remote is a real and normal thing and answers "", because refusing
// there would make the kit unusable before the first push; the caller says plainly that
// nobody else can see the claim. Several remotes and no origin refuses outright: a claim
// must not guess which one the other agents read.
func pickRemote(r *Repo) (string, error) {
	remotes := r.remotes()
	switch {
	case len(remotes) == 0:
		return "", nil
	case slices.Contains(remotes, "origin"):
		return "origin", nil
	case len(remotes) == 1:
		return remotes[0], nil
	}
	return "", fmt.Errorf("this project has several remotes and no origin (%s), and a "+
		"claim must not guess which one other agents read", strings.Join(remotes, ", "))
}

// orTrunk names the default branch, or says "the current branch" when there is no trunk
// to name. A message that says "not committed on " reads as a bug.
func orTrunk(trunk string) string {
	if trunk == "" {
		return "this branch"
	}
	return trunk
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
	if r.remotesAsked {
		return r.remoteNames
	}
	r.remotesAsked = true
	var out []string
	for _, line := range strings.Split(r.git("remote"), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			out = append(out, name)
		}
	}
	r.remoteNames = out
	return out
}
