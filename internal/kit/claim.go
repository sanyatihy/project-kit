package kit

import (
	"fmt"
	"os"
	"path/filepath"
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

	// The exclude file the claim wrote, and why it could not. Both empty when git
	// already ignored the worktree directory and there was nothing to write.
	Excluded   string
	ExcludeErr error

	// Worktree is the checkout to work the claim in, named whether or not this run
	// made it. Created is false when the path was already a worktree of the claim
	// branch, and WorktreeErr says what stopped it otherwise.
	Worktree        string
	WorktreeCreated bool
	WorktreeErr     error
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
		// Not "never written": a `## Tasks` carrying prose and no boxes counts here too,
		// and it plainly was written. What is true of every case is that there is nothing
		// under the heading an agent could build.
		return nil, &NotYours{
			fmt.Sprintf("%s has no plan to build: ## %s",
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

	// The same question asked of HEAD rather than of the doc. The claim pushes HEAD, so
	// every commit the trunk does not have travels under the new feature's name:
	// reproduced in the worktree that had just shipped 0001, where `kit claim 0002`
	// exited 0 and the branch it pushed logged 0001's commits against the trunk. That
	// pull request contains another feature, and only its author could merge it.
	if trunk := r.trunkRef(); trunk != "" && !r.reaches("HEAD", trunk) {
		return nil, fmt.Errorf("this checkout is ahead of %s, so the branch this would "+
			"push carries commits that are not %s's — another feature's work, which only "+
			"its own author could merge.\nClaim from a checkout of the trunk: git worktree "+
			"add --detach %s %s, then claim there", r.Trunk(), target.ID,
			r.WorktreePath(r.Trunk()), r.Trunk())
	}

	// One feature per checkout, which until now was a rule in prose and nothing else:
	// the board prints "take one of those in a worktree of its own, never here" and
	// `claim` took it anyway. Two units of work in one checkout share one branch, and
	// that is one pull request nobody can review.
	//
	// The branch this checkout is on, not what it has built: the check above catches a
	// claim carrying commits, and a worktree made a moment ago carries none. Any name
	// citing an id counts, because that is what counts as a claim everywhere else.
	if held := r.HeldClaim(); held != "" {
		return nil, fmt.Errorf("this checkout is on %s, which claims %s — one feature per "+
			"checkout.\nClaim from a checkout of the trunk: git worktree add --detach "+
			"%s %s, then claim there",
			r.HeadBranch(), held, r.WorktreePath(orTrunk(r.Trunk())), orTrunk(r.Trunk()))
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
	claimed := &Claimed{Branch: branch, Remote: remote}
	claimed.Excluded, claimed.ExcludeErr = r.excludeWorktrees()
	claimed.Worktree = r.WorktreePath(branch)
	claimed.WorktreeCreated, claimed.WorktreeErr = r.addWorktree(branch, claimed.Worktree)
	return claimed, nil
}

// addWorktree creates the checkout the claim is worked in, and reports whether this run
// made it.
//
// It runs after the push for the reason the exclude does: the claim is won by then and
// nothing here may undo it, so a failure is reported beside the claim rather than instead
// of it. An agent told it lost a feature it actually holds moves on for good.
//
// A path that is already a worktree of the claim branch is success, so a second claim in
// a checkout that has one is not an error. Anything else there is left where it is and
// named — it is somebody's directory, and this is not the command that removes one.
func (r *Repo) addWorktree(branch, path string) (bool, error) {
	if held := r.worktreeBranch(path); held != "" {
		if held == branch {
			return false, nil
		}
		return false, fmt.Errorf("%s is already a worktree of %s", path, held)
	}
	if _, err := os.Stat(path); err == nil {
		return false, fmt.Errorf("%s already exists and is not a worktree of %s", path, branch)
	}
	if _, err := r.gitRun("worktree", "add", path, branch); err != nil {
		return false, err
	}
	return true, nil
}

// worktreeBranch names the branch checked out at a path, and "" when git lists no
// worktree there. Records are separated by a blank line, so a path is matched against the
// branch of its own record rather than the next one's.
func (r *Repo) worktreeBranch(path string) string {
	here := false
	for _, line := range strings.Split(r.git("worktree", "list", "--porcelain"), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			here = filepath.Clean(strings.TrimPrefix(line, "worktree ")) == filepath.Clean(path)
		case here && strings.HasPrefix(line, "branch "):
			return strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		}
	}
	return ""
}

// excludeWorktrees puts the worktree directory out of git's reach in this clone, and
// names the file it wrote. Both answers empty when `git check-ignore` reports the path
// already covered, which is the second claim in any clone.
//
// It runs after the push, because the claim is won by then and nothing here may undo it:
// a write that fails is reported beside the claim rather than instead of it.
//
// `.git/info/exclude` and not a committed `.gitignore`: where an operator keeps worktrees
// is a fact about a machine, and a tracked line carries one operator's layout to every
// clone. `git clone` does not carry the exclude, which is why this belongs in `claim` —
// it runs in every clone, where `init` runs in one.
func (r *Repo) excludeWorktrees() (string, error) {
	// Non-empty output is git naming the pattern that covers the path. An empty answer
	// is "nothing ignores this", which is also what a failed call looks like — and
	// writing the line twice is cheaper to live with than not writing it at all.
	if strings.TrimSpace(r.git("check-ignore", WorktreesDir+"/")) != "" {
		return "", nil
	}
	common := strings.TrimSpace(r.git("rev-parse", "--git-common-dir"))
	if common == "" {
		return "", fmt.Errorf("git named no repository directory to write the exclude in")
	}
	// The common dir, not this worktree's own: `info/exclude` is shared by every
	// worktree of one clone, and a linked worktree's `.git` holds no `info` at all.
	if !filepath.IsAbs(common) {
		common = filepath.Join(r.Root, common)
	}
	path := filepath.Join(common, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	// A last line with no newline on it would otherwise swallow the pattern into itself.
	lead := ""
	if old, err := os.ReadFile(path); err == nil && len(old) > 0 && !strings.HasSuffix(string(old), "\n") {
		lead = "\n"
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(lead + WorktreesDir + "/\n")
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	return path, nil
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

// WorktreesDir is where a claim's worktree goes: inside the project, so the parent stays
// a list of repositories and each live claim opens in the editor as its own repository
// with its own branch and diff. The leading dot is load-bearing — ripgrep and the Go
// toolchain both skip a dot directory on the name alone, so the second copy of the source
// stays out of every recursive read.
const WorktreesDir = ".worktrees"

// MainWorktree is the checkout that owns the repository, which `git worktree list
// --porcelain` names in its first record. Answers the project root when git names
// nothing, which is a directory that is not a repository at all.
func (r *Repo) MainWorktree() string {
	for _, line := range strings.Split(r.git("worktree", "list", "--porcelain"), "\n") {
		if rest, found := strings.CutPrefix(strings.TrimSpace(line), "worktree "); found {
			return rest
		}
	}
	return r.Root
}

// WorktreePath is where the worktree for a branch goes. Absolute and anchored at the main
// worktree, because the path `claim` printed was relative to the current directory: right
// from the trunk, and a sibling of whatever worktree the agent happened to stand in
// otherwise. Named for the branch, so the listing reads as the board.
func (r *Repo) WorktreePath(branch string) string {
	return filepath.Join(r.MainWorktree(), WorktreesDir, branch)
}
