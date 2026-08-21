package kit

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	StatusDraft   = "Draft"
	StatusShipped = "Shipped"
	StatusDropped = "Dropped"

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

	// The answer while it is still in flight. The round trip is the whole cost of a board
	// read, and no local answer depends on it, so it is started first and collected after.
	// Buffered, so the goroutine never outlives the command still holding a send.
	headsPending chan headsResult

	// Every feature doc's status on the remote's trunk, read in one pass. Asked once: it
	// cannot change while one command runs, and shipped-not-pushed needs it per doc.
	trunkDocs      map[string]string
	trunkDocsAsked bool

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
		fields, body := ParseFrontmatter(Normalise(string(raw)))
		f.ID = fields["id"]
		f.Title = fields["title"]
		f.Status = fields["status"]
		f.Needs = ParseIDList(fields["needs"])
		f.Shipped = ParseDate(fields["shipped"])
		f.shippedText = strings.TrimSpace(fields["shipped"])
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

// UnmetNeeds lists the ids a feature is still waiting on, in declared order.
//
// Shipped is the bar rather than merged. `kit ship` is the moment the kit knows a
// feature's work is done; merging is a human's and follows. Reading merge state
// instead would make the board's answer depend on how quickly somebody clicked.
//
// An id that names nothing, or names a feature that was Dropped, counts
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
		// using the kit, silently and with no finding to explain it. `bad-status` reports
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
