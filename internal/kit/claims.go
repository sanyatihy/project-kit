package kit

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// The claim: which branch holds which feature, what the trunk is, and what is waiting on
// the far side of ship. A claim is a branch name and nothing else, so all of this is ref
// reading.

// Trunk is the project's default branch: whatever `origin/HEAD` points at, then `main`,
// then `master`.
//
// Asked of the remote rather than assumed, because a project whose default branch is
// neither of the two hard-coded names had every branch on it read as a claim. The answer
// is cached: it cannot change under a single command, and both the claim checks and
// BranchBuildsOn ask for it per feature.
func (r *Repo) Trunk() string {
	if r.trunk == "" {
		r.trunk = r.findTrunk()
	}
	return r.trunk
}

func (r *Repo) findTrunk() string {
	for _, remote := range r.remotes() {
		ref := strings.TrimSpace(r.git("symbolic-ref", "--quiet", "--short",
			"refs/remotes/"+remote+"/HEAD"))
		if name := branchName(ref); name != "" && name != "HEAD" {
			return name
		}
	}
	for _, candidate := range []string{"main", "master"} {
		if r.hasRef("refs/heads/" + candidate) {
			return candidate
		}
	}
	// The remote-tracking copy, before falling back to HEAD. `origin/HEAD` is written by
	// `clone` and not by `push -u`, so a project that started life locally has a real
	// trunk on the remote and no symbolic ref naming it. Without this rung a checkout
	// standing on a claim branch found no trunk at all, and `BranchBuildsOn` answers
	// "carries no work" for an empty trunk — which is the answer that deletes.
	for _, remote := range r.remotes() {
		for _, candidate := range []string{"main", "master"} {
			if r.hasRef("refs/remotes/" + remote + "/" + candidate) {
				return candidate
			}
		}
	}
	// No remote to ask and neither conventional name: fall back to the branch this
	// checkout is on, but only when it is not claim-shaped. A claim branch leads with its
	// feature's ID, so refusing those is what stops a worktree deciding it is the trunk —
	// and a wrong answer here would report every branch as carrying no work.
	if head := r.HeadBranch(); head != "" && idPrefixRe.FindString(head) == "" {
		return head
	}
	return ""
}

// isTrunk reports whether a branch is the default one, and so never a claim. The two
// conventional names stay unconditional: a repo can carry both, and the one that is not
// the default is still not somebody's feature branch.
func (r *Repo) isTrunk(name string) bool {
	return name == "main" || name == "master" || name == "HEAD" || name == r.Trunk()
}

// trunkRef is a revision for the trunk that git can actually resolve. A fresh clone has
// `origin/main` and no local `main` until something checks it out, so comparing against
// the bare name alone would silently answer "nothing changed" for every branch.
func (r *Repo) trunkRef() string {
	trunk := r.Trunk()
	if trunk == "" {
		return ""
	}
	if r.hasRef("refs/heads/" + trunk) {
		return trunk
	}
	for _, remote := range r.remotes() {
		if r.hasRef("refs/remotes/" + remote + "/" + trunk) {
			return remote + "/" + trunk
		}
	}
	return ""
}

// HeadBranch is the branch this checkout is on, empty when HEAD is detached or
// there is no git repository.
//
// A worktree has its own HEAD, so in a worktree this is that worktree's branch —
// which is the whole point: it is how the kit tells the checkout you are standing
// in apart from the ones other agents are standing in.
func (r *Repo) HeadBranch() string {
	if !r.headAsked {
		r.headAsked = true
		r.head = strings.TrimSpace(r.git("symbolic-ref", "--quiet", "--short", "HEAD"))
	}
	return r.head
}

// Claims lists the distinct branches claiming a feature. More than one means two
// agents believe they own it.
//
// A branch is matched on the feature's declared id and on its filename prefix, which
// are the same string in every doc that is not defective. Where they disagree the
// branch is named after the file and mentions the prefix only, so matching the id
// alone loses the claim: the board would offer a feature somebody is already holding.
// `claim` now refuses to create such a claim, so this is for the ones already made.
func (r *Repo) Claims(id string) []string {
	r.loadGit()
	ids := []string{id}
	for _, f := range r.Features {
		if f.ID != id {
			continue
		}
		if prefix := f.FilenamePrefix(); prefix != "" && prefix != id {
			ids = append(ids, prefix)
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, ref := range r.refs {
		name := branchName(ref)
		if r.isTrunk(name) || seen[name] || !mentionsAny(name, ids) {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func mentionsAny(text string, ids []string) bool {
	for _, id := range ids {
		if Mentions(text, id) {
			return true
		}
	}
	return false
}

// Mine reports whether this checkout stands on the branch that won a feature — the one
// named after its doc, and no other name.
//
// Equality, not membership in `Claims`. `Claims` matches every branch mentioning the id,
// which is what a second claimant looks like and not what ownership is: after losing the
// race, `git checkout -b notes-0001-alpha` made the board print the feature as yours and
// let `ship` freeze the winner's doc. `review-0001` and `fix-0001` qualified too, so it
// needed no malice. The claim is the branch the push created, so only that name is it.
//
// One definition for two callers: `next` reports it as yours, and `ship` requires it.
// Shipping from the worktree holding the claim is the documented path; shipping from
// anywhere else is one agent freezing another's work.
func (r *Repo) Mine(id string) bool {
	f := r.FeatureByID(id)
	return f != nil && r.HeadBranch() == f.Branch()
}

// BranchBuildsOn reports whether a branch has changed anything beyond the one file
// given — that is, whether somebody is building on it rather than only proposing it.
//
// This is what separates a claim from a draft awaiting review. Agents draft features
// and leave them for a human, and nothing is committed to the default branch, so the
// proposal has to live on a branch until it is accepted. A branch carrying only the
// feature doc is that; a branch carrying code is work. Git already knows which, so
// neither the doc nor the kit needs a new field to say so.
//
// Unknown answers false: with no trunk to compare against, or a branch that cannot be
// resolved, the quiet answer is the safe one — this feeds a check, and a finding
// invented from missing information is worse than a finding not made.
func (r *Repo) BranchBuildsOn(branch, rel string) bool {
	trunk := r.trunkRef()
	if trunk == "" || branch == "" || branch == r.Trunk() {
		return false
	}
	// Three dots: what the branch changed since it diverged, not what has landed on
	// trunk meanwhile. Otherwise every branch looks busy as soon as trunk moves.
	changed := r.git("diff", "--name-only", trunk+"..."+branch)
	for _, line := range strings.Split(changed, "\n") {
		if path := strings.TrimSpace(line); path != "" && path != rel {
			return true
		}
	}
	return false
}

// Review is one feature whose builder has finished and whose work has not landed.
type Review struct {
	Feature *Feature
	Branch  string
	Since   time.Time // the branch's last commit: when it stopped moving
	Mine    bool      // this reader built it, so it is not this reader's to merge
}

// AwaitingReview lists the features waiting for somebody other than their builder: a
// claim branch the remote has, carrying a copy of the doc that says Shipped, and not yet
// reachable from the trunk.
//
// The status is read from the branch rather than from this checkout, because the branch
// is the only place it exists yet. A coordinator stands on the trunk, where an unmerged
// feature's doc still says whatever it said when the plan round landed — read from here,
// every entry would be missing in exactly the case this exists to report.
//
// Silent rather than wrong wherever the answer cannot be had: offline, no remote, or a
// remote that did not answer reports nothing, and so does a branch whose objects this
// checkout has never fetched. Both are the kit's standing answer to a fact it cannot
// read, and both are recoverable with a fetch.
func (r *Repo) AwaitingReview() []Review {
	heads, ok := r.remoteHeads()
	if !ok {
		return nil
	}
	trunk := r.trunkRef()
	var out []Review
	for _, f := range r.Features {
		ids := claimIDs(f)
		for _, name := range heads {
			if r.isTrunk(name) || !mentionsAny(name, ids) {
				continue
			}
			rev := r.resolveBranch(name)
			if rev == "" || r.statusOn(rev, f.Rel) != StatusShipped {
				continue
			}
			// A merged branch the remote still carries is not waiting for anybody. The
			// local trunk ref answers it, so this asks the network nothing beyond the
			// one ls-remote — the same bargain `claim` makes.
			if trunk != "" && r.reaches(rev, trunk) {
				continue
			}
			out = append(out, Review{Feature: f, Branch: name, Since: r.commitDate(rev),
				Mine: r.BuiltHere(f.ID, rev)})
			break
		}
	}
	return out
}

// OnlyHere reports whether a feature's work exists nowhere but this checkout: no branch
// the remote has claims it, and the trunk on the remote carries no copy of its doc saying
// Shipped.
//
// Two independent negatives, because either alone misfires. A branch the remote has is
// the awaiting-review state and must not report; the trunk's copy is the merged one, and
// a Shipped status on it is the proof the handoff happened.
//
// The status is the test and presence is not. A doc reaches the trunk when it is
// *drafted* — plan rounds merge Drafts — so "the trunk has the file" is true for every
// feature before a line of work exists, and testing that would silence this check
// completely. Reading the age of the commit that last wrote the doc was the earlier
// answer and it misfired the other way: once 0038 had `kit ship` ask for compaction,
// editing a shipped doc became routine, and one blank line appended to a doc merged weeks
// ago reported at HIGH with a fix line that recreated a stale claim branch.
//
// Unknown answers false throughout: offline, no remote, a remote that did not answer, or
// a trunk that exists and this checkout has never fetched. An empty answer is not one of
// them — a remote that says it has no branches has told us where the work is not, which
// is the state four features reached in one session and the reason this check exists.
func (r *Repo) OnlyHere(f *Feature) bool {
	heads, ok := r.remoteHeads()
	if !ok {
		return false
	}
	trunked := false
	for _, name := range heads {
		if !r.isTrunk(name) && mentionsAny(name, claimIDs(f)) {
			return false
		}
		trunked = trunked || r.isTrunk(name)
	}
	statuses, ok := r.trunkStatuses()
	if !ok {
		// Nothing of the trunk to read. A remote holding no trunk at all carries nothing
		// and has said so; a trunk that exists and has not been fetched is ignorance.
		return !trunked
	}
	return statuses[f.Rel] != StatusShipped
}

// trunkStatuses is every feature doc's status on the remote's trunk, read in one pass and
// memoised. False when there is no copy of a trunk to read, which is a fetch away and
// never a finding.
//
// The remote's trunk and not the local one: the question is whether the work reached
// anybody else, and a local trunk holding a merge nobody has seen answers a different one.
//
// Every branch this repo counts as a trunk, not only the one `origin/HEAD` names, and
// Shipped anywhere wins. A stale `origin/HEAD` — the ordinary state of a clone made before
// a default branch was renamed — points at a branch nothing merges to any more, and one
// that fires this HIGH on every shipped feature is worse than no check at all.
//
// Two processes per trunk — `ls-tree` names the blobs, one `cat-file --batch` reads them —
// where asking per feature cost a `log` each plus a `branch --contains` per revision: 63
// processes on this repo, 25 of them three distinct questions.
func (r *Repo) trunkStatuses() (map[string]string, bool) {
	if r.trunkDocsAsked {
		return r.trunkDocs, r.trunkDocs != nil
	}
	r.trunkDocsAsked = true
	if r.FeaturesDir == "" {
		return nil, false
	}
	var found map[string]string
	for _, ref := range r.remoteTrunkRefs() {
		for rel, status := range r.statusesOn(ref) {
			if found == nil {
				found = map[string]string{}
			}
			if found[rel] != StatusShipped {
				found[rel] = status
			}
		}
	}
	r.trunkDocs = found
	return found, found != nil
}

// statusesOn reads every feature doc's status from one revision's tree, nil when it cannot
// be read at all.
func (r *Repo) statusesOn(ref string) map[string]string {
	listing, err := r.gitRun("ls-tree", "-r", ref, "--", r.FeaturesDir)
	if err != nil {
		return nil
	}
	paths := map[string]string{} // blob sha -> path
	var wanted strings.Builder
	for _, line := range strings.Split(listing, "\n") {
		meta, path, found := strings.Cut(strings.TrimSpace(line), "\t")
		if !found {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) < 3 || fields[1] != "blob" {
			continue
		}
		paths[fields[2]] = path
		wanted.WriteString(fields[2] + "\n")
	}
	if len(paths) == 0 {
		return nil
	}
	batch, err := r.gitFed(wanted.String(), "cat-file", "--batch")
	if err != nil {
		return nil
	}

	// `<sha> blob <size>` and then the bytes, so the size is what says where one object
	// ends: a feature doc can hold anything, including a line shaped like the next header.
	out := map[string]string{}
	rest := batch
	for {
		header, body, found := strings.Cut(rest, "\n")
		if !found {
			break
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[1] != "blob" {
			break
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size > len(body) {
			break
		}
		if path, ok := paths[fields[0]]; ok {
			doc, _ := ParseFrontmatter(Normalise(body[:size]))
			out[path] = doc["status"]
		}
		rest = strings.TrimPrefix(body[size:], "\n")
	}
	return out
}

// remoteTrunkRefs is every remote-tracking copy of a trunk this checkout has. Deliberately
// not trunkRef, which prefers a local branch of the same name: what is on this machine is
// not what the work was handed off to.
func (r *Repo) remoteTrunkRefs() []string {
	var out []string
	seen := map[string]bool{}
	for _, remote := range r.remotes() {
		for _, name := range []string{r.Trunk(), "main", "master"} {
			ref := remote + "/" + name
			if name == "" || seen[ref] || !r.hasRef("refs/remotes/"+ref) {
				continue
			}
			seen[ref] = true
			out = append(out, ref)
		}
	}
	return out
}

// BuiltHere reports whether the reader built this feature — the one thing a coordinator
// that is itself an agent has to know, because its own work is the work it may not merge.
//
// Two answers, weakest last. The checkout standing on the claim is unambiguous. Otherwise
// the branch's author is compared with this checkout's git identity, which is worth
// exactly as much as the identities are distinct: on this repo four agents shared the
// operator's, and it separated none of them. That is an argument for giving agents
// identities, not for deleting the arm that pays off when they have them. Unknown answers
// false: this makes the invariant legible, and legibility is what a well-behaved agent
// needs. Bounded by the number of shipped-and-unmerged features, which is small.
func (r *Repo) BuiltHere(id, rev string) bool {
	if r.Mine(id) {
		return true
	}
	who := strings.TrimSpace(r.git("config", "user.email"))
	return who != "" && strings.TrimSpace(r.git("log", "-1", "--format=%ae", rev)) == who
}

// claimIDs are the strings a branch may cite to claim a feature: the id it declares, and
// its filename prefix where the two disagree — the state that makes a claim invisible to
// a search for the id alone.
func claimIDs(f *Feature) []string {
	ids := []string{f.ID}
	if prefix := f.FilenamePrefix(); prefix != "" && prefix != f.ID {
		ids = append(ids, prefix)
	}
	return ids
}

// TasksOn counts the boxes in a revision's copy of a doc. Reports false when that
// revision cannot be read or does not carry the doc, which is the caller's cue to keep
// whatever the checkout says rather than report a count of nothing.
func (r *Repo) TasksOn(rev, rel string) (open, total int, ok bool) {
	out, err := r.gitRun("show", rev+":"+rel)
	if err != nil {
		return 0, 0, false
	}
	_, body := ParseFrontmatter(Normalise(out))
	open, total = CountTasks(ParseSections(body)["Tasks"])
	return open, total, true
}

// statusOn reads a doc's status: from a revision's copy of it, empty when that revision
// does not carry the doc, cannot be read, or the doc declares no status.
func (r *Repo) statusOn(rev, rel string) string {
	out, err := r.gitRun("show", rev+":"+rel)
	if err != nil {
		return ""
	}
	fields, _ := ParseFrontmatter(Normalise(out))
	return fields["status"]
}

// OrphanClaims lists branches named like a claim — a leading four-digit ID — that cite a
// feature doc nobody wrote.
//
// Only branches that look like claims are considered. Ordinary working branches
// cite no ID and must never be reported, or the check becomes noise.
//
// A finding here needs positive evidence, not absence. The protocol puts a draft on a
// branch of its own and commits nothing to the trunk, so from the trunk the doc is
// *always* missing while the feature is in flight — and this reported every open feature
// as an orphan, advising a fix that deletes the draft. So a branch is reported only when
// its own tree can be read and confirms the doc is not there. An unresolvable branch — one
// that exists on a remote nobody has fetched — is passed over: missing a real orphan until
// someone fetches costs far less than telling an agent to delete work it cannot see.
func (r *Repo) OrphanClaims() []string {
	r.loadGit()
	known := map[string]bool{}
	for _, f := range r.Features {
		if f.ID != "" {
			known[f.ID] = true
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, ref := range r.refs {
		name := branchName(ref)
		if r.isTrunk(name) || seen[name] {
			continue
		}
		m := idPrefixRe.FindStringSubmatch(name)
		if m == nil || known[m[1]] {
			continue
		}
		seen[name] = true
		if rev := r.resolveBranch(name); rev == "" || r.BranchCarriesItsDoc(name) {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// HasClaimShapedBranch reports whether any branch leads with a feature ID.
//
// Enough to know somebody is writing a plan somewhere, which is all `no-plan` needs: its
// job is to catch a project with commits and nothing written down, and a branch named
// `0001-…` is not that even when the trunk holds no doc yet.
func (r *Repo) HasClaimShapedBranch() bool {
	r.loadGit()
	for _, ref := range r.refs {
		name := branchName(ref)
		if !r.isTrunk(name) && idPrefixRe.FindString(name) != "" {
			return true
		}
	}
	return false
}

// BranchCarriesItsDoc reports whether a branch holds the feature doc it is named after.
// The doc names the branch and the branch cites the doc, so the filename is derivable and
// no mapping has to be kept.
func (r *Repo) BranchCarriesItsDoc(branch string) bool {
	rev := r.resolveBranch(branch)
	if rev == "" {
		return false
	}
	dir := "docs/features"
	if r.FeaturesDir != "" {
		dir = r.Rel(r.FeaturesDir)
	}
	_, err := r.gitRun("cat-file", "-e", rev+":"+dir+"/"+branch+".md")
	return err == nil
}

// LastMentioned is the newest commit citing this ID, or the zero time.
func (r *Repo) LastMentioned(id string) time.Time {
	r.loadGit()
	for _, c := range r.commits { // git log is newest-first
		if Mentions(c.Message, id) {
			return c.When
		}
	}
	return time.Time{}
}
