package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	StatusDraft         = "Draft"
	StatusParked        = "Parked"
	StatusSubmitted     = "Submitted"
	StatusLegacyShipped = "Shipped"
	StatusDropped       = "Dropped"

	gitLogLimit = "500"
	gitTimeout  = 30 * time.Second

	// remoteTimeout bounds the one question the kit asks the network on a read. Bounded
	// on purpose: `kit next ~/code/*` walks every project, and a board that hangs because
	// one remote is unreachable is worse than a board that falls back to a stale mirror
	// and says so.
	//
	// It was 5s, which a working remote did not fit inside: `git ls-remote --heads` to
	// GitHub over SSH measured 5.58s on an ordinary connection, so the review board went
	// blank and `submitted-not-pushed` never ran, with nothing on the page saying why. Tune
	// it against that number, and against RemoteUnanswered — a bound that expires is now
	// reported rather than printed as an empty board.
	remoteTimeout = 15 * time.Second

	// waitDelay is how long a git that has been killed, or has exited, gets before its
	// pipes are closed under it.
	//
	// Without it the deadlines above are not deadlines. `exec.CommandContext` kills
	// `git` and nothing else, so `git-remote-https` — a grandchild — goes on holding the
	// stdout pipe, and `cmd.Output()` blocks on a read that never ends: `kit next`
	// against a black-holed remote came back in 75s against a bound of 15. It is part of
	// the bound rather than added to it, because the bound is what the help promises.
	waitDelay = time.Second
)

// The layout `kit init` writes, and the only layout the kit reads. Alternatives used to
// be accepted here — three feature directories, four names for the as-built doc — for
// projects the kit never scaffolded. Nothing produced them, and `createTechDoc` hard-coded
// one of the four anyway, so a project using a different spelling got a second as-built
// doc rather than an updated one.
//
// `protocols` keeps two names because `CLAUDE.md` is a convention the kit does not own.
var featureDirs = []string{"docs/features"}
var techDocs = []string{"docs/TECH.md"}
var visionDocs = []string{"docs/VISION.md"}
var inboxes = []string{"docs/INBOX.md"}
var protocols = []string{"AGENTS.md", "CLAUDE.md"}

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

	commits        []commit
	refs           []string
	loaded         bool
	claimWorktrees map[string]string // branch -> absolute checkout path, read with refs

	// The default branch, asked once. The flag is what makes "there is no trunk" an
	// answer rather than a question not yet asked: without it, every caller re-ran the
	// whole ladder, and on a board where nothing resolves that is one `symbolic-ref` per
	// claim per feature.
	trunk      string
	trunkAsked bool

	// refsAuthoritative records whether the claim refs came from the remote itself or
	// from a past fetch. Only the second can produce a claim nobody holds, so only the
	// second earns a message about pruning.
	refsAuthoritative bool

	// The remote's branch list, asked once. It cannot change under a single command, and
	// `audit` asks it per Submitted feature: unmemoised, one ls-remote per doc took this
	// repo's audit from under a second to two minutes.
	heads      []string
	headsKnown bool
	headsAsked bool

	// What each remote advertises for each branch, from the same round trip. `release`
	// deletes a ref on the remote, so it has to know whether the copy it read is that ref.
	tips map[string]string

	// headsFailed records that the remote was asked and did not answer — a timeout or a
	// failed call. Distinct from having no remote to ask, which is not a failure and has
	// nothing to report.
	headsFailed bool

	// The answer while it is still in flight. The round trip is the whole cost of a board
	// read, and no local answer depends on it, so it is started first and collected after.
	// Buffered, so the goroutine never outlives the command still holding a send.
	headsPending chan headsResult

	// Every feature doc's status on the remote's trunk, read in one pass. Asked once: it
	// cannot change while one command runs, and submitted-not-pushed needs it per doc.
	trunkDocs       map[string]string
	trunkDocsAsked  bool
	integratedDocs  map[string]string
	integratedAsked bool
	foundationAsked bool
	foundationIssue string
	headDocs        map[string]string
	headDocsAsked   bool

	// The branch this checkout is on and the remotes it has, asked once each. Neither can
	// change while one command runs, and both were being asked per feature: 62 of the 116
	// git processes a board read spawned on a 30-feature project were these two.
	head         string
	headAsked    bool
	remoteNames  []string
	remotesAsked bool

	// Every path in HEAD's tree, read in one `git ls-tree`. `audit` asks whether each
	// feature doc is committed, and one `cat-file` per doc is a cost that grows with the
	// board — which is the shape the git budget exists to catch.
	headPaths      map[string]bool
	headPathsAsked bool

	// Every path in the trunk's tree, read the same way and for the same reason. `next`
	// asks whether each feature's plan has landed, and a `cat-file` per doc is the cost
	// that grows with the board.
	trunkPaths      map[string]bool
	trunkPathsAsked bool

	// The ids the remote has handed out, mirrored in one fetch and read once. The fetch
	// is the only network cost, and whether it succeeded is what separates a reservation
	// that is gone from one this checkout has not heard about.
	reservations      []Reservation
	reservationsFresh bool
	reservationsAsked bool

	// Every ref this repository has, read in one `git show-ref`. The alternative is one
	// `rev-parse` per candidate per feature, which is where 32 of a board read's 58
	// processes went.
	refIndex    map[string]bool
	refsIndexed bool
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
		r.Features = append(r.Features, r.readFeature(filepath.Join(r.FeaturesDir, name)))
	}
}

// readFeature never returns nil. Skipping a doc it could not open made the file invisible
// to every command — absent from the board, unknown to `claim`, and, if it held the
// highest id, a number `kit new` handed out a second time. The filename is still readable,
// so the id stays reserved and `audit` has a file to name.
func (r *Repo) readFeature(path string) *Feature {
	rel, err := filepath.Rel(r.Root, path)
	if err != nil {
		rel = path
	}
	f := &Feature{Path: path, Rel: filepath.ToSlash(rel)}

	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		f.Unreadable = true
		f.ID = f.FilenamePrefix()
	} else {
		text := Normalise(string(raw))
		fields, body := ParseFrontmatter(text)
		f.ID = fields["id"]
		f.Title = fields["title"]
		f.Status = fields["status"]
		f.Needs = ParseIDList(fields["needs"])
		stamp, present := fields["submitted"]
		if !present {
			stamp = fields["shipped"]
		}
		f.Submitted = ParseDate(stamp)
		f.submittedText = strings.TrimSpace(stamp)
		f.needsText = strings.TrimSpace(fields["needs"])
		f.Duplicates = DuplicateKeys(text)
		f.sections = ParseSections(body)
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

// WriteTarget is FeatureByID for the commands that write — `submit`, `drop` and `block`.
//
// Reading a doc tolerates an ambiguity that writing one cannot. A key declared twice makes
// the read and the write different lines: ParseFrontmatter keeps the last occurrence and
// SetField rewrites the first, so `submit` reported success, left two contradictory statuses
// behind and `audit` repeated the same finding with the same fix for ever. Which line the
// author meant is not the kit's to guess.
func (r *Repo) WriteTarget(id string) (*Feature, error) {
	var share []*Feature
	for _, f := range r.Features {
		if f.ID == id {
			share = append(share, f)
		}
	}
	if len(share) == 0 {
		return nil, fmt.Errorf("no feature %q", id)
	}
	// FeatureByID answers with the first match, so an id naming two docs was invisible to
	// the three commands that write: `drop` retired whichever sorted first, and said
	// nothing. `claim` counts them and refuses, because one branch cannot claim two
	// features; a write cannot land on two docs either.
	if len(share) > 1 {
		var where []string
		for _, f := range share {
			where = append(where, f.Rel)
		}
		return nil, fmt.Errorf("%d feature docs declare id %s (%s), so this cannot tell "+
			"which one to act on\nrun kit audit and renumber all but one",
			len(share), id, strings.Join(where, " and "))
	}
	target := share[0]
	if len(target.Duplicates) > 0 {
		return nil, fmt.Errorf("%s declares %s twice in its frontmatter, so the line the "+
			"kit reads is not the line it would rewrite\ndelete the one that is wrong, "+
			"then run this again", target.Name(), strings.Join(target.Duplicates, " and "))
	}
	return target, nil
}

// PlanError rejects ambiguous plans before routing or changing their lifecycle.
func (r *Repo) PlanError(f *Feature) error {
	if _, err := r.WriteTarget(f.ID); err != nil {
		return err
	}
	if f.Unreadable {
		return fmt.Errorf("%s cannot be read", f.Rel)
	}
	if !idListOnlyRe.MatchString(f.needsText) {
		return fmt.Errorf("%s: needs: is not an id list", f.Name())
	}
	if r.needsCycle(f.ID, f.ID, map[string]bool{}) {
		return fmt.Errorf("%s: dependency cycle in needs: — remove the circular dependency", f.Name())
	}
	for _, id := range f.Needs {
		if dep := r.FeatureByID(id); dep != nil {
			if _, err := r.WriteTarget(id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Repo) needsCycle(start, id string, seen map[string]bool) bool {
	if seen[id] {
		return false
	}
	seen[id] = true
	f := r.FeatureByID(id)
	if f == nil || terminal(f.Status) {
		return false
	}
	for _, need := range f.Needs {
		if need == start || r.needsCycle(start, need, seen) {
			return true
		}
	}
	return false
}

// UnmetNeeds requires a submitted prerequisite on the trunk and in this checkout.
// A stamp on an unmerged branch is a review request, not an available foundation.
func (r *Repo) UnmetNeeds(f *Feature) []string {
	var unmet []string
	for _, id := range f.Needs {
		dep := r.FeatureByID(id)
		if dep == nil || !IsSubmitted(dep.Status) ||
			!IsSubmitted(r.integratedStatuses()[dep.Rel]) || !IsSubmitted(r.headStatuses()[dep.Rel]) {
			unmet = append(unmet, id)
		}
	}
	return unmet
}

// PlanLanded reports whether a feature's plan has reached the trunk, which is the bar
// `kit claim` sets before a claim may branch from here.
//
// One question read by two callers, the way `UnmetNeeds` is: `claim` refuses on it, and
// `next` reports the feature as blocked rather than printing a command that exits 2. The
// two answers came from different places once, and the board sent every new project to a
// claim that turned it away.
//
// True where no trunk resolves. There is then nothing for the doc to be absent from, and
// answering otherwise would block the first claim of every project that has one branch
// and no remote.
func (r *Repo) PlanLanded(f *Feature) bool {
	// A plan already on trunk can still carry an unreviewed activation here.
	if f.Status == StatusDraft && r.integratedStatuses()[f.Rel] == StatusParked {
		return false
	}
	// Unknown landing state is handled by foundationProblem, which names the fetch.
	// It must not be reported as a plan known to be absent from the trunk.
	return r.trackedOnTrunk(f.Rel) || r.trunkPaths == nil
}

// Rel makes a path relative to the project root, for display.
func (r *Repo) Rel(path string) string {
	rel, err := filepath.Rel(r.Root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
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

// inboxSep separates the fields of an inbox line:
//
//   - [ ] YYYY-MM-DD · <feature-id> · BLOCKING|FYI · <question>
const inboxSep = "·"

// ActiveFeatures lists features open for work — every Draft, by ID.
//
// Ordering is by ID alone. It used to rank Accepted ahead of Draft, and dropping that
// status left lowest-first as the whole rule: the claim this checkout already holds comes
// first, and `next` handles that separately.
func (r *Repo) ActiveFeatures() []*Feature {
	var out []*Feature
	for _, f := range r.Features {
		// Draft, and anything else that is neither terminal nor missing. A status this
		// version does not know is still open work: dropping `Accepted` would otherwise
		// have made every doc carrying it vanish from the board of every project already
		// using the kit, silently and with no finding to explain it. `no-status` reports
		// it instead, and the feature stays visible while somebody fixes it.
		if f.Status != "" && !terminal(f.Status) {
			out = append(out, f)
		}
	}
	rank := func(status string) int {
		if status == StatusDraft {
			return 0
		}
		return 1
	}
	// No claim-first rank here: "finishing beats starting" is structural now. `next`
	// renders this checkout's own claim as **yours** above everything, and somebody
	// else's under **taken** below what is free, so ordering inside the list cannot move
	// a claimed feature anywhere the groups do not already put it.
	sort.SliceStable(out, func(i, j int) bool {
		if rank(out[i].Status) != rank(out[j].Status) {
			return rank(out[i].Status) < rank(out[j].Status)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// HasHistory reports whether the repo has any commits, distinguishing a project
// that is under way from one that was only just created.
func (r *Repo) HasHistory() bool {
	r.loadGit()
	return len(r.commits) > 0
}
