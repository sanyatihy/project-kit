package kit

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	StatusDraft    = "Draft"
	StatusAccepted = "Accepted"
	StatusShipped  = "Shipped"

	gitLogLimit = "500"
	gitTimeout  = 30 * time.Second

	// remoteTimeout bounds the one question the kit asks the network on a read. Bounded
	// on purpose: `kit next ~/code/*` walks every project, and a board that hangs because
	// one remote is unreachable is worse than a board that falls back to a stale mirror
	// and says so.
	//
	// It was 5s, which a working remote did not fit inside: `git ls-remote --heads` to
	// GitHub over SSH measured 5.58s on an ordinary connection, so the review board went
	// blank and `shipped-not-pushed` never ran, with nothing on the page saying why. Tune
	// it against that number, and against RemoteUnanswered — a bound that expires is now
	// reported rather than printed as an empty board.
	remoteTimeout = 15 * time.Second
)

// AllStatuses is every status a feature doc may declare.
//
// There is deliberately no "In progress": the claim branch says work has started, and a
// second copy of that fact in the doc could only ever disagree with it. A doc still
// carrying the old value is reported as `bad-status`, which is the migration.
var AllStatuses = []string{StatusAccepted, StatusDraft, "Dropped", StatusShipped,
	"Superseded"}

// terminal reports whether a status is an end state, after which a branch bearing the
// feature's name is a merged leftover rather than somebody's work in progress.
func terminal(status string) bool {
	return status == StatusShipped || status == "Dropped" || status == "Superseded"
}

var notAFeature = map[string]bool{
	"TEMPLATE.md": true, "README.md": true, "BACKLOG.md": true, "INBOX.md": true,
}

var featureDirs = []string{"docs/features", "features", "doc/features"}
var techDocs = []string{"docs/TECH.md", "TECH.md", "docs/ARCHITECTURE.md", "ARCHITECTURE.md"}
var visionDocs = []string{"docs/VISION.md", "VISION.md", "docs/PRODUCT.md"}
var inboxes = []string{"docs/INBOX.md", "INBOX.md"}
var protocols = []string{"AGENTS.md", "CLAUDE.md"}

var idPrefixRe = regexp.MustCompile(`^(\d+)`)

// Feature is one feature document.
type Feature struct {
	Path    string
	Rel     string
	ID      string
	Title   string
	Status  string
	Needs   []string // feature ids that must ship before this can start
	Shipped time.Time
	Check   time.Time
	Checked time.Time

	// Review is the human's mark on one feature: set, a person merges it whatever the
	// trunk allows. Empty and absent are the same answer — the coordinator may merge —
	// so every doc written before the field existed keeps working untouched.
	Review string

	sections map[string]string
}

// NeedsHuman reports whether a person must see this feature merged, whatever branch
// protection allows. Any value counts, not only the documented `human`: a field somebody
// wrote in and the kit did not recognise should ask for a person rather than quietly
// waving the change through, which is the only direction that fails safe.
func (f *Feature) NeedsHuman() bool { return f.Review != "" }

// Name is the ID and title, for display.
func (f *Feature) Name() string {
	id := f.ID
	if id == "" {
		id = "????"
	}
	return id + " " + f.Title
}

// Section returns a section's trimmed content, empty when absent.
func (f *Feature) Section(heading string) string { return f.sections[heading] }

// PlannedSections are what an agent needs in order to work a feature at all. `Log` is
// not among them: it is written as the work happens, so it is checked at Shipped only.
var PlannedSections = []string{"Problem", "Design", "Tasks"}

// Unwritten lists the sections an agent would need and cannot read, in document order.
//
// Shared by `audit`, which reports it, and `claim`, which refuses over it. Once a Draft
// can be claimed, the emptiness test is the only thing standing between a doc straight
// out of `kit new` and an agent sent to build it — so both have to mean the same thing
// by "nobody wrote the plan". The template's guidance lives in comments, which every
// parser here strips, so an empty section was genuinely never written.
func (f *Feature) Unwritten() []string {
	var out []string
	for _, heading := range PlannedSections {
		if f.Section(heading) == "" {
			out = append(out, heading)
		}
	}
	return out
}

// OpenTasks lists unticked task boxes in document order.
func (f *Feature) OpenTasks() []string { return Unticked(f.Section("Tasks")) }

// TaskCount is the open and total box counts. A feature with no boxes at all has
// nothing for an agent to do and needs different advice from one whose boxes are
// all ticked.
func (f *Feature) TaskCount() (open, total int) { return CountTasks(f.Section("Tasks")) }

// NextTask is the one to do now. List order is the order of work.
func (f *Feature) NextTask() string {
	if tasks := f.OpenTasks(); len(tasks) > 0 {
		return tasks[0]
	}
	return ""
}

// Branch is the branch name that claims this feature: the doc's filename without
// its extension. The doc names the branch and the branch cites the doc, so either
// one gives you the other and there is no mapping to keep in sync.
func (f *Feature) Branch() string {
	return strings.TrimSuffix(filepath.Base(f.Path), ".md")
}

// FilenamePrefix is the leading number of the filename, for the id cross-check.
func (f *Feature) FilenamePrefix() string {
	if m := idPrefixRe.FindStringSubmatch(filepath.Base(f.Path)); m != nil {
		return m[1]
	}
	return ""
}

// Repo is a project on disk. Layout is discovered, so there is no config to drift.
type Repo struct {
	Root     string
	Name     string
	Features []*Feature

	FeaturesDir string // "" when the project is not set up
	Tech        string
	Vision      string
	Inbox       string
	Protocol    string
	Backlog     string // agents write to it, so the secret scan has to open it

	// Offline stops the kit asking the remote anything, at the cost of reading claims
	// from a mirror that may be stale.
	Offline bool

	commits []commit
	refs    []string
	trunk   string
	loaded  bool

	// refsAuthoritative records whether the claim refs came from the remote itself or
	// from a past fetch. Only the second can produce a claim nobody holds, so only the
	// second earns a message about pruning.
	refsAuthoritative bool

	// The remote's branch list, asked once. It cannot change under a single command, and
	// `audit` asks it per Shipped feature: unmemoised, one ls-remote per doc took this
	// repo's audit from under a second to two minutes.
	heads      []string
	headsKnown bool
	headsAsked bool

	// headsFailed records that the remote was asked and did not answer — a timeout or a
	// failed call. Distinct from having no remote to ask, which is not a failure and has
	// nothing to report.
	headsFailed bool
}

type commit struct {
	When    time.Time
	Message string
}

// Open reads a project.
func Open(root string) (*Repo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	r := &Repo{Root: abs, Name: filepath.Base(abs)}
	r.FeaturesDir = r.firstDir(featureDirs)
	r.Tech = r.firstFile(techDocs)
	r.Vision = r.firstFile(visionDocs)
	r.Inbox = r.firstFile(inboxes)
	r.Protocol = r.firstFile(protocols)
	if r.FeaturesDir != "" {
		p := filepath.Join(r.FeaturesDir, "BACKLOG.md")
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			r.Backlog = p
		}
	}
	r.loadFeatures()
	return r, nil
}

func (r *Repo) firstDir(rels []string) string {
	for _, rel := range rels {
		p := filepath.Join(r.Root, rel)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

func (r *Repo) firstFile(rels []string) string {
	for _, rel := range rels {
		p := filepath.Join(r.Root, rel)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

func (r *Repo) loadFeatures() {
	if r.FeaturesDir == "" {
		return
	}
	entries, err := os.ReadDir(r.FeaturesDir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || notAFeature[e.Name()] {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		if f := r.readFeature(filepath.Join(r.FeaturesDir, name)); f != nil {
			r.Features = append(r.Features, f)
		}
	}
}

func (r *Repo) readFeature(path string) *Feature {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil // an unreadable file is a filesystem problem, not a doc problem
	}
	fields, body := ParseFrontmatter(Normalise(string(raw)))
	rel, err := filepath.Rel(r.Root, path)
	if err != nil {
		rel = path
	}
	f := &Feature{
		Path:     path,
		Rel:      filepath.ToSlash(rel),
		ID:       fields["id"],
		Title:    fields["title"],
		Status:   fields["status"],
		Needs:    ParseIDList(fields["needs"]),
		Shipped:  ParseDate(fields["shipped"]),
		Check:    ParseDate(fields["check"]),
		Checked:  ParseDate(fields["checked"]),
		Review:   fields["review"],
		sections: ParseSections(body),
	}
	if f.Title == "" {
		f.Title = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	return f
}

// FeatureByID finds a feature by its declared id, or nil.
func (r *Repo) FeatureByID(id string) *Feature {
	for _, f := range r.Features {
		if f.ID == id {
			return f
		}
	}
	return nil
}

// UnmetNeeds lists the ids a feature is still waiting on, in declared order.
//
// Shipped is the bar rather than merged. `kit ship` is the moment the kit knows a
// feature's work is done; merging is a human's and follows. Reading merge state
// instead would make the board's answer depend on how quickly somebody clicked.
//
// An id that names nothing, or names a feature that was Dropped or Superseded, counts
// as unmet: it can never be satisfied, so the work genuinely cannot start. `audit`
// reports why; `next` only has to refuse to hand it out.
func (r *Repo) UnmetNeeds(f *Feature) []string {
	var unmet []string
	for _, id := range f.Needs {
		if id == f.ID {
			continue // self-reference is a cycle, and audit says so
		}
		if dep := r.FeatureByID(id); dep == nil || dep.Status != StatusShipped {
			unmet = append(unmet, id)
		}
	}
	return unmet
}

// NeedsCycles lists loops in the needs: graph, each as the ids in the loop starting
// and ending at the same one.
//
// Reported rather than merely avoided: a loop means every feature in it waits on
// itself forever, so `next` would go quiet about all of them and never say why. One
// finding per loop, not one per feature in it — the loop is the defect and it has one
// fix.
func (r *Repo) NeedsCycles() [][]string {
	// Iterative depth-first search with an explicit path. `state` is the standard
	// three-colour marking: unseen, on the current path, and finished. A node already
	// finished cannot start a new cycle, which is what keeps a diamond-shaped graph
	// from being walked exponentially.
	const (
		unseen = iota
		onPath
		done
	)
	state := map[string]int{}
	var cycles [][]string
	seen := map[string]bool{}

	var walk func(id string, path []string)
	walk = func(id string, path []string) {
		state[id] = onPath
		path = append(path, id)
		f := r.FeatureByID(id)
		if f != nil {
			for _, next := range f.Needs {
				if r.FeatureByID(next) == nil {
					continue // a missing id is needs-unknown's business, not a loop
				}
				switch state[next] {
				case onPath:
					// Report from where the loop closes, so the output is the loop
					// itself rather than the walk that happened to find it.
					at := slices.Index(path, next)
					loop := append(append([]string{}, path[at:]...), next)
					if key := strings.Join(loop, ">"); !seen[key] {
						seen[key] = true
						cycles = append(cycles, loop)
					}
				case unseen:
					walk(next, path)
				}
			}
		}
		state[id] = done
	}

	for _, f := range r.Features {
		if f.ID != "" && state[f.ID] == unseen {
			walk(f.ID, nil)
		}
	}
	return cycles
}

// Rel makes a path relative to the project root, for display.
func (r *Repo) Rel(path string) string {
	rel, err := filepath.Rel(r.Root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func (r *Repo) git(args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.Root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// gitRun runs git and reports what went wrong, returning stdout either way.
//
// The reads above use git(), which treats a failure as "nothing found" because a
// project with no commits and a project whose log could not be read look the same
// to a report. Anything that changes state needs the opposite: a push that did not
// happen must never be mistaken for one that found nothing to do.
func (r *Repo) gitRun(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.Root}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return stdout.String(), fmt.Errorf("git %s: %s", args[0], detail)
	}
	return stdout.String(), nil
}

func (r *Repo) loadGit() {
	if r.loaded {
		return
	}
	r.loaded = true

	// --all, because in a pull-request flow every commit is made on a branch that has
	// not merged yet. Reading HEAD's log alone made the kit blind to all work in
	// progress: a feature being actively built looked exactly like an abandoned one.
	raw := r.git("log", "--all", "-n"+gitLogLimit, "--format=%cI\x1f%B\x1e")
	for _, chunk := range strings.Split(raw, "\x1e") {
		when, message, found := strings.Cut(strings.TrimSpace(chunk), "\x1f")
		if !found {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, when)
		if err != nil {
			continue
		}
		r.commits = append(r.commits, commit{When: parsed, Message: message})
	}

	// Local branches and worktree branches are this machine's own state and cannot be
	// stale. What the remote has is a different question, and `refs/remotes/*` is only a
	// cache of the last fetch — git prunes it on neither fetch nor pull, so a branch
	// deleted when its pull request merged stays there forever and reads as a live
	// claim. Ask the remote instead; fall back to the cache only when it cannot answer.
	seen := map[string]bool{}
	for _, line := range strings.Split(r.git("branch", "--format=%(refname:short)"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			seen[line] = true
		}
	}
	if heads, ok := r.remoteHeads(); ok {
		r.refsAuthoritative = true
		for _, name := range heads {
			seen[name] = true
		}
	} else {
		for _, line := range strings.Split(r.git("branch", "-r", "--format=%(refname:short)"), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				seen[line] = true
			}
		}
	}
	for _, line := range strings.Split(r.git("worktree", "list", "--porcelain"), "\n") {
		if strings.HasPrefix(line, "branch ") {
			parts := strings.Split(strings.TrimSpace(line), "/")
			seen[parts[len(parts)-1]] = true
		}
	}
	for name := range seen {
		r.refs = append(r.refs, name)
	}
	sort.Strings(r.refs)
}

// remoteHeads lists the branch names every remote actually has right now, reporting
// false when nothing could be asked: no remote, offline, or a call that failed.
//
// One `ls-remote` per remote. It transfers no objects and writes nothing — the kit still
// never mutates refs behind an agent's back — and it is bounded by a short timeout of its
// own, because `kit next ~/code/*` fans out over every project and one unreachable remote
// must not hang the board.
func (r *Repo) remoteHeads() ([]string, bool) {
	if r.Offline {
		return nil, false
	}
	if r.headsAsked {
		return r.heads, r.headsKnown
	}
	r.headsAsked = true
	var out []string
	answered := false
	for _, remote := range r.remotes() {
		ctx, cancel := context.WithTimeout(context.Background(), remoteTimeout)
		cmd := exec.CommandContext(ctx, "git", "-C", r.Root, "ls-remote", "--heads", remote)
		raw, err := cmd.Output()
		cancel()
		if err != nil {
			// One unreachable remote does not invalidate another's answer, but it does
			// mean this one's branches are unknown, so nothing here can be authoritative.
			r.headsFailed = true
			return nil, false
		}
		answered = true
		for _, line := range strings.Split(string(raw), "\n") {
			_, ref, found := strings.Cut(strings.TrimSpace(line), "\t")
			if !found {
				continue
			}
			if name := strings.TrimPrefix(ref, "refs/heads/"); name != ref && name != "" {
				out = append(out, name)
			}
		}
	}
	r.heads, r.headsKnown = out, answered
	return out, answered
}

// RemoteUnanswered reports whether something this command needed from the remote could
// not be had: it was asked and the call failed or timed out.
//
// False where there was nothing to ask — no remote, `--offline`, or a command that never
// needed the branch list. Those are not failures and have nothing to report; this is for
// the case where an answer exists and the kit could not get it, which otherwise reaches
// the reader as a page with nothing on it.
//
// Read it after the work, not before: it reports what actually went unanswered rather
// than provoking a call nobody needed.
func (r *Repo) RemoteUnanswered() bool { return r.headsFailed }

// branchName drops a remote prefix, so origin/0009-thing and 0009-thing are
// recognised as one branch rather than two claims on the same feature.
func branchName(ref string) string {
	if i := strings.Index(ref, "/"); i >= 0 && (strings.HasPrefix(ref, "origin/") ||
		strings.HasPrefix(ref, "remotes/")) {
		return ref[i+1:]
	}
	return ref
}

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
		if r.git("rev-parse", "--verify", "--quiet", candidate) != "" {
			return candidate
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
	candidates := []string{trunk}
	for _, remote := range r.remotes() {
		candidates = append(candidates, remote+"/"+trunk)
	}
	for _, ref := range candidates {
		if r.git("rev-parse", "--verify", "--quiet", ref) != "" {
			return ref
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
	return strings.TrimSpace(r.git("symbolic-ref", "--quiet", "--short", "HEAD"))
}

// RefsAreAuthoritative reports whether the claim refs were read from the remotes
// themselves rather than from the last fetch. Only the second can name a branch that no
// longer exists, so only the second needs saying out loud.
func (r *Repo) RefsAreAuthoritative() bool {
	r.loadGit()
	return r.refsAuthoritative
}

// TrackedAtHEAD reports whether a repo-relative path exists in the commit HEAD points
// at — that is, whether it would travel with a branch pushed from here.
func (r *Repo) TrackedAtHEAD(rel string) bool {
	return r.trackedAt("HEAD", rel)
}

// trackedAt reports whether a repo-relative path exists in a revision's tree.
func (r *Repo) trackedAt(rev, rel string) bool {
	_, err := r.gitRun("cat-file", "-e", rev+":"+rel)
	return err == nil
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

// Mine reports whether the branch this checkout is on is one of the branches claiming a
// feature — that is, whether the claim is this agent's rather than somebody else's.
//
// One definition for two callers: `next` reports it as yours, and `ship` requires it.
// Shipping from the worktree holding the claim is the documented path; shipping from
// anywhere else is one agent freezing another's work.
func (r *Repo) Mine(id string) bool {
	return contains(r.Claims(id), r.HeadBranch())
}

// ClaimFor returns the branch claiming a feature, or "". The default branch
// never counts as a claim.
func (r *Repo) ClaimFor(id string) string {
	if claims := r.Claims(id); len(claims) > 0 {
		return claims[0]
	}
	return ""
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
// the remote has claims it, and the commit that last wrote its doc is on no
// remote-tracking ref either.
//
// Two independent negatives, because either alone misfires. The remote's branch list is
// live but says nothing about a feature whose branch was deleted after merging; the
// remote-tracking refs know that one and are only as fresh as the last fetch. Both have
// to agree that the work is nowhere before this says so.
//
// Unknown answers false throughout: offline, no remote, a remote that did not answer, or
// a doc git has never seen. Nothing here is worth a finding invented from silence.
func (r *Repo) OnlyHere(f *Feature) bool {
	heads, ok := r.remoteHeads()
	if !ok {
		return false
	}
	for _, name := range heads {
		if !r.isTrunk(name) && mentionsAny(name, claimIDs(f)) {
			return false
		}
	}
	rev := strings.TrimSpace(r.git("log", "-1", "--format=%H", "--", f.Rel))
	if rev == "" {
		return false
	}
	return strings.TrimSpace(r.git("branch", "--remotes", "--contains", rev)) == ""
}

// BuiltHere reports whether the reader built this feature — the one thing a coordinator
// that is itself an agent has to know, because its own work is the work it may not merge.
//
// Two answers, weakest last. The checkout standing on the claim is unambiguous. Otherwise
// the branch's author is compared with this checkout's git identity, which is only worth
// anything where agents have identities of their own: measured on this repo, every commit
// from four different agents carried the operator's name, and nothing — not this, not a
// forge rule requiring a non-author reviewer — could tell them apart. Unknown answers
// false: this makes the invariant legible, and legibility is all a well-behaved agent
// needs.
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

// commitDate is when a revision was last committed, or the zero time. How long a branch
// has waited is measured from here rather than from the doc's shipped:, which is a date
// and so cannot tell an hour from a day.
func (r *Repo) commitDate(rev string) time.Time {
	if out := strings.TrimSpace(r.git("log", "-1", "--format=%cI", rev)); out != "" {
		if when, err := time.Parse(time.RFC3339, out); err == nil {
			return when
		}
	}
	return time.Time{}
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

// reaches reports whether a revision has landed on another — whether the branch's work
// is already contained in the trunk.
func (r *Repo) reaches(rev, trunk string) bool {
	_, err := r.gitRun("merge-base", "--is-ancestor", rev, trunk)
	return err == nil
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

// resolveBranch finds a revision for a branch name, widening from the local ref to each
// remote-tracking one, and returns "" when nothing resolves.
func (r *Repo) resolveBranch(branch string) string {
	candidates := []string{"refs/heads/" + branch}
	for _, remote := range r.remotes() {
		candidates = append(candidates, "refs/remotes/"+remote+"/"+branch)
	}
	for _, ref := range candidates {
		if r.git("rev-parse", "--verify", "--quiet", ref) != "" {
			return ref
		}
	}
	return ""
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

// LastTouched is the last commit date for a path, falling back to mtime when
// untracked. A check that silently does nothing on an untracked file is worse
// than no check at all.
func (r *Repo) LastTouched(path string) time.Time {
	if out := strings.TrimSpace(r.git("log", "--all", "-1", "--format=%cI", "--", path)); out != "" {
		if parsed, err := time.Parse(time.RFC3339, out); err == nil {
			return parsed
		}
	}
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

// OpenQuestion is one unticked inbox line.
//
// Feature is the ID from the line's second field, empty when the line does not
// follow the documented format. Blocking decisions use it in preference to the
// whole line, so a question that merely mentions another feature in passing does
// not stop work on it.
type OpenQuestion struct {
	When    time.Time
	Text    string
	Feature string
	FYI     bool
}

// inboxSep separates the fields of an inbox line:
//
//   - [ ] YYYY-MM-DD · <feature-id> · BLOCKING|FYI · <question>
const inboxSep = "·"

// parseInboxLine pulls the feature ID and the BLOCKING/FYI marker out of a line
// in the documented format, reporting whether it was in that format at all.
//
// The question text is deliberately not validated: it is prose and may contain
// anything, including the separator.
func parseInboxLine(text string) (id string, fyi, ok bool) {
	fields := strings.SplitN(text, inboxSep, 4)
	if len(fields) < 3 {
		return "", false, false
	}
	if ParseDate(strings.TrimSpace(fields[0])).IsZero() {
		return "", false, false
	}
	marker := strings.TrimSpace(fields[2])
	switch {
	case strings.EqualFold(marker, "FYI"):
		fyi = true
	case strings.EqualFold(marker, "BLOCKING"):
		fyi = false
	default:
		return "", false, false
	}
	if id = strings.TrimSpace(fields[1]); id == "" {
		return "", false, false
	}
	return id, fyi, true
}

// OpenQuestions reads the inbox, skipping fenced examples.
func (r *Repo) OpenQuestions() []OpenQuestion {
	if r.Inbox == "" {
		return nil
	}
	raw, err := os.ReadFile(r.Inbox)
	if err != nil {
		return nil
	}
	var out []OpenQuestion
	for _, text := range Unticked(Normalise(string(raw))) {
		if text == "" {
			continue
		}
		q := OpenQuestion{When: ParseDate(text), Text: text}
		if id, fyi, ok := parseInboxLine(text); ok {
			q.Feature, q.FYI = id, fyi
		} else {
			q.FYI = isFYI(text)
		}
		out = append(out, q)
	}
	return out
}

// BlockersFor lists open questions that block this feature.
//
// A line marked FYI is a note, not a blocker — it is recorded so the human sees
// it, but work continues. Anything else blocks, because an unmarked question is
// more likely to be a real one than a courtesy note.
//
// A line in the documented format is matched on its feature-ID field alone.
// Anything else falls back to scanning the whole line, which over-blocks: a
// question that names the ID only in passing will stop work. That is the safe
// direction to be wrong in — over-blocking is visible and annoying, while missing
// a blocker sends an agent into work a human has not cleared. `inbox-unparsed`
// reports the lines that land here so they can be fixed rather than tolerated.
func (r *Repo) BlockersFor(id string) []string {
	var out []string
	for _, q := range r.OpenQuestions() {
		if q.FYI {
			continue
		}
		matched := Mentions(q.Text, id)
		if q.Feature != "" {
			matched = Mentions(q.Feature, id)
		}
		if matched {
			out = append(out, q.Text)
		}
	}
	return out
}

func isFYI(text string) bool {
	for _, field := range strings.Split(text, "·") {
		if strings.EqualFold(strings.TrimSpace(field), "FYI") {
			return true
		}
	}
	return false
}

// ActiveFeatures lists features open for work: In progress, then Accepted, then Draft,
// then by ID. That ordering is the whole WIP policy.
//
// Draft is in here because `status:` orders the board rather than gating it. Accepted
// means the operator has looked and blessed it, so it goes out first; a Draft is
// invented and not yet blessed, so it goes out last and the agent should expect to
// settle its `## Open questions` before building. Neither is permission.
func (r *Repo) ActiveFeatures() []*Feature {
	rank := map[string]int{StatusAccepted: 0, StatusDraft: 1}
	var out []*Feature
	for _, f := range r.Features {
		if _, ok := rank[f.Status]; ok {
			out = append(out, f)
		}
	}
	// No claim-first rank here: "finishing beats starting" is structural now. `next`
	// renders this checkout's own claim as **yours** above everything, and somebody
	// else's under **taken** below what is free, so ordering inside the list cannot move
	// a claimed feature anywhere the groups do not already put it.
	sort.SliceStable(out, func(i, j int) bool {
		if rank[out[i].Status] != rank[out[j].Status] {
			return rank[out[i].Status] < rank[out[j].Status]
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ProjectRoot finds the repo root containing start, so commands work from any
// subdirectory. Falls back to start when there is no git.
func ProjectRoot(start string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", start, "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return root
		}
	}
	return start
}

// HasHistory reports whether the repo has any commits, distinguishing a project
// that is under way from one that was only just created.
func (r *Repo) HasHistory() bool {
	r.loadGit()
	return len(r.commits) > 0
}
