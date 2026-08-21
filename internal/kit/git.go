package kit

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Git plumbing: every subprocess the kit runs, and the answers it memoises. Nothing here
// knows what a feature is.

type commit struct {
	When    time.Time
	Message string
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
// gitFed is gitRun with something on stdin, for the one caller that batches: `cat-file
// --batch` reads a list of objects and writes them all back in one process.
func (r *Repo) gitFed(stdin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.Root}, args...)...)
	cmd.Stdin = strings.NewReader(stdin)
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

	// Ask the remote before the local walk rather than after it. One `ls-remote` is 2.3s
	// against GitHub over SSH where everything else here is under a tenth of that, and no
	// local answer depends on it, so the wait overlaps the work instead of following it.
	r.startRemoteHeads()

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
	// `branch refs/heads/<name>`, and the name is the whole rest of it. Keeping only the
	// last path segment was right for the branches the kit names itself, which carry no
	// slash — but `agents/0039-work` then landed in here twice, once whole from the branch
	// list and once as `0039-work`, and one checkout reported itself as two agents holding
	// the feature. `split-claim` said so, and its fix line deletes the work.
	for _, line := range strings.Split(r.git("worktree", "list", "--porcelain"), "\n") {
		if rest, found := strings.CutPrefix(strings.TrimSpace(line), "branch "); found {
			if name := strings.TrimPrefix(rest, "refs/heads/"); name != "" {
				seen[name] = true
			}
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
		r.collectRemoteHeads()
		return r.heads, r.headsKnown
	}
	r.headsAsked = true
	r.apply(r.fetchHeads(r.remotes()))
	return r.heads, r.headsKnown
}

// headsResult is one round's answer, carried back from the goroutine that asked so that
// nothing on the Repo is written from two places at once.
type headsResult struct {
	heads  []string
	known  bool
	failed bool
}

// startRemoteHeads sends the round trip off to run beside the local walk. The remotes are
// resolved here, on the caller's goroutine, so the one below touches nothing but the root
// path — the memo it would otherwise fill is shared, and this keeps it single-writer.
func (r *Repo) startRemoteHeads() {
	if r.Offline || r.headsAsked {
		return
	}
	r.headsAsked = true
	remotes := r.remotes()
	r.headsPending = make(chan headsResult, 1)
	go func() { r.headsPending <- r.fetchHeads(remotes) }()
}

// collectRemoteHeads waits for a round trip started earlier, once.
func (r *Repo) collectRemoteHeads() {
	if r.headsPending == nil {
		return
	}
	r.apply(<-r.headsPending)
	r.headsPending = nil
}

func (r *Repo) apply(res headsResult) {
	r.heads, r.headsKnown, r.headsFailed = res.heads, res.known, res.failed
}

// fetchHeads does the asking and touches no memo, so it is safe on a goroutine of its
// own: everything it decides comes back in the result.
func (r *Repo) fetchHeads(remotes []string) headsResult {
	var out []string
	answered := false
	for _, remote := range remotes {
		ctx, cancel := context.WithTimeout(context.Background(), remoteTimeout)
		cmd := exec.CommandContext(ctx, "git", "-C", r.Root, "ls-remote", "--heads", remote)
		raw, err := cmd.Output()
		cancel()
		if err != nil {
			// One unreachable remote does not invalidate another's answer, but it does
			// mean this one's branches are unknown, so nothing here can be authoritative.
			return headsResult{failed: true}
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
	return headsResult{heads: out, known: answered}
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
func (r *Repo) RemoteUnanswered() bool {
	r.collectRemoteHeads()
	return r.headsFailed
}

// branchName drops a remote prefix, so origin/0009-thing and 0009-thing are
// recognised as one branch rather than two claims on the same feature.
func branchName(ref string) string {
	if i := strings.Index(ref, "/"); i >= 0 && (strings.HasPrefix(ref, "origin/") ||
		strings.HasPrefix(ref, "remotes/")) {
		return ref[i+1:]
	}
	return ref
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
//
// One `ls-tree` for the whole tree, memoised. `audit` asks this per feature to decide
// whether the plan checks apply, and a `cat-file` each took a 30-feature board from 10
// git processes to 40.
func (r *Repo) TrackedAtHEAD(rel string) bool {
	if !r.headPathsAsked {
		r.headPathsAsked = true
		r.headPaths = map[string]bool{}
		for _, path := range gitPaths(r.git("ls-tree", "-r", "-z", "--name-only", "HEAD")) {
			r.headPaths[path] = true
		}
	}
	return r.headPaths[rel]
}

// gitPaths splits a NUL-separated path list into the bytes the filesystem uses.
//
// Every git read here that yields a path asks for `-z`. Without it git C-quotes anything
// outside printable ASCII — `"docs/features/0001-bl\303\240nk.md"` — and every one of
// these answers is then compared against a name `os.ReadDir` produced, so the lookup
// silently misses. One accented character disabled `unfilled-template`, `no-tasks` and
// `done-not-shipped` on that doc, and made `shipped-not-pushed` fire at HIGH on work
// that was pushed. `-z` also settles a newline inside a filename, which no line-oriented
// parse can.
func gitPaths(out string) []string {
	var paths []string
	for _, path := range strings.Split(out, "\x00") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

// trackedAt reports whether a repo-relative path exists in a revision's tree.
func (r *Repo) trackedAt(rev, rel string) bool {
	_, err := r.gitRun("cat-file", "-e", rev+":"+rel)
	return err == nil
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

// DirtyFiles lists what this checkout has not committed: changed, staged and untracked
// paths, as `git status --porcelain` reports them. Files git ignores are not among them,
// which is what keeps build output and local settings out of the answer.
//
// Unknown answers empty, the way every other git question here does. This one feeds a
// refusal, so inventing files from a failed call would block a ship for no reason.
func (r *Repo) DirtyFiles() []string {
	out, err := r.gitRun("status", "--porcelain", "-z")
	if err != nil {
		return nil
	}
	var paths []string
	records := gitPaths(out)
	for i := 0; i < len(records); i++ {
		// Two status columns, a space, then the path: "?? scratch/out.json".
		if len(records[i]) <= 3 {
			continue
		}
		paths = append(paths, records[i][3:])
		// A rename or a copy carries its source in a record of its own, with no status
		// columns on it. Read as an ordinary entry it would lose its first three bytes.
		if strings.ContainsAny(records[i][:2], "RC") {
			i++
		}
	}
	return paths
}

// reaches reports whether a revision has landed on another — whether the branch's work
// is already contained in the trunk.
func (r *Repo) reaches(rev, trunk string) bool {
	_, err := r.gitRun("merge-base", "--is-ancestor", rev, trunk)
	return err == nil
}

// hasRef reports whether a fully-qualified ref exists, from one `git show-ref` read once.
//
// Existence is all any caller here needs, and asking per candidate cost a process each:
// with ten claims that was a third of a board read. A repository with no refs at all
// answers false for everything, which is correct — there is nothing to resolve.
func (r *Repo) hasRef(ref string) bool {
	if !r.refsIndexed {
		r.refsIndexed = true
		r.refIndex = map[string]bool{}
		for _, line := range strings.Split(r.git("show-ref"), "\n") {
			if _, name, found := strings.Cut(strings.TrimSpace(line), " "); found {
				r.refIndex[name] = true
			}
		}
	}
	return r.refIndex[ref]
}

// resolveBranch finds a revision for a branch name, widening from the local ref to each
// remote-tracking one, and returns "" when nothing resolves.
func (r *Repo) resolveBranch(branch string) string {
	if r.hasRef("refs/heads/" + branch) {
		return "refs/heads/" + branch
	}
	for _, remote := range r.remotes() {
		if ref := "refs/remotes/" + remote + "/" + branch; r.hasRef(ref) {
			return ref
		}
	}
	return ""
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
