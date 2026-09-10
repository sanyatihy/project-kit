package kit

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Where two features will meet in the same file.
//
// The two sides are not the same kind of fact. A live claim's files are evidence — what
// its branch has actually changed. A free feature's files are prose its author wrote
// before starting. Nothing here refuses anything, because a claim made in prose cannot
// carry a refusal; `needs-unstated` is LOW for the same reason.

// backtickRe matches a backticked token, which is how these docs write a path.
var backtickRe = regexp.MustCompile("`([^`\n]+)`")

// NamedPaths lists the repo-relative files a feature names in `## Design` and `## Tasks`
// and that exist in this checkout, deduplicated and ordered.
//
// Existence is what makes prose usable: `internal/kit/audit.go` is a path and "the audit
// registry" is not, and a token naming no file is dropped rather than guessed at. A doc
// that names none is silent — measured 2026-09-10, 35 of the 83 docs here name any, at a
// median of one, which is the size of what this can report.
func (r *Repo) NamedPaths(f *Feature) []string {
	seen := map[string]bool{}
	for _, heading := range []string{"Design", "Tasks"} {
		for _, m := range backtickRe.FindAllStringSubmatch(f.Section(heading), -1) {
			rel := strings.TrimSpace(m[1])
			// A path has no spaces, stays inside the project, and names a file that is
			// there. Everything else in backticks is a command, a flag or a heading.
			if rel == "" || strings.ContainsAny(rel, " \t") ||
				strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
				continue
			}
			if info, err := os.Stat(filepath.Join(r.Root, rel)); err != nil || info.IsDir() {
				continue
			}
			seen[rel] = true
		}
	}
	return sorted(seen)
}

// ChangedPaths lists what a revision has changed against the trunk.
//
// A revision and not a branch name: a claim another agent pushed exists here only as
// `refs/remotes/*`, and `work/0001` alone resolves to nothing, so the diff failed
// silently and the overlap went unreported exactly when it mattered most. `stalled`
// resolves the same way before asking git anything.
//
// Three dots, as `BranchBuildsOn` uses: what the branch changed since it diverged, not
// what has landed on trunk meanwhile. Unknown answers nothing — with no trunk to compare
// against, a report invented from missing information is worse than none.
//
// Memoised for the life of the Repo, because the answer cannot change inside one command
// and `Survey` asks it once per ready feature. Without this a board of F such features
// and C claims spent F×C diffs on C distinct answers, which is the growth-with-board-size
// regression `TestABoardReadKeepsGitWorkBoundedAsFeaturesGrow` exists to refuse.
func (r *Repo) ChangedPaths(rev string) []string {
	trunk := r.trunkRef()
	if trunk == "" || rev == "" {
		return nil
	}
	if cached, ok := r.changed[rev]; ok {
		return cached
	}
	seen := map[string]bool{}
	for _, rel := range gitPaths(r.git("diff", "--name-only", "-z", trunk+"..."+rev)) {
		seen[rel] = true
	}
	out := sorted(seen)
	if r.changed == nil {
		r.changed = map[string][]string{}
	}
	r.changed[rev] = out
	return out
}

// Overlap is a live claim a free feature would meet, and where.
type Overlap struct {
	ID     string // the feature whose claim has changed these files
	Branch string
	Paths  []string
}

// OverlapsFor lists the live claims whose work touches files this feature names.
//
// Its own claim is not an overlap with itself, and a terminal feature is not live work.
func (r *Repo) OverlapsFor(f *Feature) []Overlap {
	named := r.NamedPaths(f)
	if len(named) == 0 {
		return nil
	}
	want := map[string]bool{}
	for _, rel := range named {
		want[rel] = true
	}
	var out []Overlap
	for _, other := range r.Features {
		if other.ID == "" || other.ID == f.ID || terminal(other.Status) {
			continue
		}
		for _, branch := range r.Claims(other.ID) {
			hit := map[string]bool{}
			for _, rel := range r.ChangedPaths(r.claimProgressRev(branch)) {
				// A branch carrying only its own doc is a claim with nothing in it yet.
				if want[rel] && rel != other.Rel {
					hit[rel] = true
				}
			}
			if len(hit) > 0 {
				out = append(out, Overlap{ID: other.ID, Branch: branch, Paths: sorted(hit)})
				break // one report per feature, not one per branch it is claimed on
			}
		}
	}
	return out
}

func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
