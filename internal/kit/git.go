package kit

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"sort"
	"strconv"
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
	cmd.WaitDelay = waitDelay
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// gitFed is gitRun with something on stdin, for the one caller that batches: `cat-file
// --batch` reads a list of objects and writes them all back in one process.
func (r *Repo) gitFed(stdin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.Root}, args...)...)
	cmd.WaitDelay = waitDelay
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
	cmd.WaitDelay = waitDelay
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
	// `for-each-ref` and not `git branch`, whose business is a page for a person to read:
	// it prints `(no branch, rebasing 0002-beta)` and `(HEAD detached at …)` among the
	// names, and `%(refname:short)` disambiguates a branch a tag shadows as
	// `heads/0002-beta`. Each was read here as a second branch claiming the feature, so
	// `split-claim` fired at a name nobody could act on and `release` refused for good.
	// The ref list has no such entries and `lstrip=2` takes the name whole.
	seen := map[string]bool{}
	for _, line := range strings.Split(
		r.git("for-each-ref", "--format=%(refname:lstrip=2)", "refs/heads/"), "\n") {
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
	tips   map[string]string // "<remote>/<branch>" -> the commit the remote advertises
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
	r.heads, r.tips, r.headsKnown, r.headsFailed = res.heads, res.tips, res.known, res.failed
}

// fetchHeads does the asking and touches no memo, so it is safe on a goroutine of its
// own: everything it decides comes back in the result.
func (r *Repo) fetchHeads(remotes []string) headsResult {
	var out []string
	tips := map[string]string{}
	answered := false
	for _, remote := range remotes {
		// The deadline is remoteTimeout less what WaitDelay may spend closing the pipes
		// after it fires, so the whole question stays inside the bound the help states.
		ctx, cancel := context.WithTimeout(context.Background(), remoteTimeout-waitDelay)
		cmd := exec.CommandContext(ctx, "git", "-C", r.Root, "ls-remote", "--heads", remote)
		cmd.WaitDelay = waitDelay
		raw, err := cmd.Output()
		cancel()
		if err != nil {
			// One unreachable remote does not invalidate another's answer, but it does
			// mean this one's branches are unknown, so nothing here can be authoritative.
			return headsResult{failed: true}
		}
		answered = true
		for _, line := range strings.Split(string(raw), "\n") {
			sha, ref, found := strings.Cut(strings.TrimSpace(line), "\t")
			if !found {
				continue
			}
			if name := strings.TrimPrefix(ref, "refs/heads/"); name != ref && name != "" {
				out = append(out, name)
				tips[remote+"/"+name] = sha
			}
		}
	}
	return headsResult{heads: out, tips: tips, known: answered}
}

// remoteTip is the commit a remote advertises for a branch, and false when that could not
// be had: offline, a remote that did not answer, or a branch it does not carry.
//
// The sha comes from the same `ls-remote` the board already runs, so it costs nothing
// beyond keeping the half of each line that was being thrown away. It is the only reading
// of a claim branch that cannot be stale — `refs/remotes/*` is a cache of the last fetch,
// and a local head is a cache of the last push.
func (r *Repo) remoteTip(remote, branch string) (string, bool) {
	if _, ok := r.remoteHeads(); !ok {
		return "", false
	}
	sha, ok := r.tips[remote+"/"+branch]
	return sha, ok
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
	return r.headsFailed || (r.reservationsAsked && !r.reservationsFresh && !r.Offline && len(r.remotes()) > 0)
}

// branchName drops a remote prefix, so origin/0009-thing and 0009-thing are
// recognised as one branch rather than two claims on the same feature.
//
// Every remote this project has configured, not the conventional name alone. A clone
// whose remote is called `upstream` read its trunk out of `upstream/HEAD` and kept the
// prefix, so the trunk was `upstream/main` — a name no ref resolves. Everything that
// compares against the trunk then had nothing to compare with: `stalled` could never
// fire, and `release` refused every claim.
//
// `remotes/` stays unconditional: it is the prefix of a ref path rather than a remote's
// name, and no project can configure it away.
func (r *Repo) branchName(ref string) string {
	name, rest, found := strings.Cut(ref, "/")
	if found && (name == "remotes" || slices.Contains(r.remotes(), name)) {
		return rest
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

// TagDistance is the newest tag reachable from the trunk and how many trunk commits are
// not in it. False when there is no trunk to measure against, or no tag at all.
//
// The trunk and not HEAD: a claim branch is ahead of the trunk by its own work, and that
// work is not a release nobody cut. One `describe --long`, which answers both halves at
// once — `v0.2.0-235-g3a913ba` is the tag, the distance and the commit. Parsed from the
// right, because a tag name may carry the separator and the last two fields never do.
func (r *Repo) TagDistance() (tag string, behind int, ok bool) {
	ref := r.trunkRef()
	if ref == "" {
		return "", 0, false
	}
	out := strings.TrimSpace(r.git("describe", "--tags", "--long", ref))
	commit := strings.LastIndex(out, "-")
	if commit <= 0 {
		return "", 0, false
	}
	count := strings.LastIndex(out[:commit], "-")
	if count <= 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(out[count+1 : commit])
	if err != nil {
		return "", 0, false
	}
	return out[:count], n, true
}

// gitPaths splits a NUL-separated path list into the bytes the filesystem uses.
//
// Every git read here that yields a path asks for `-z`. Without it git C-quotes anything
// outside printable ASCII — `"docs/features/0001-bl\303\240nk.md"` — and every one of
// these answers is then compared against a name `os.ReadDir` produced, so the lookup
// silently misses. One accented character disabled `unfilled-template` and
// `done-not-submitted` on that doc, and made `submitted-not-pushed` fire at HIGH on work
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

// trackedOnTrunk reports whether a repo-relative path exists in the trunk's tree, and
// false whenever no trunk resolves — there is then nothing for the path to be absent
// from, which is the state every project starts in.
//
// One `ls-tree` for the whole tree, memoised, for the reason TrackedAtHEAD is one: the
// board asks this per feature, and a `cat-file` each is a cost that grows with the board.
func (r *Repo) trackedOnTrunk(rel string) bool {
	if !r.trunkPathsAsked {
		r.trunkPathsAsked = true
		if ref := r.integrationRef(); ref != "" {
			if out, err := r.gitRun("ls-tree", "-r", "-z", "--name-only", ref); err == nil {
				r.trunkPaths = map[string]bool{}
				for _, path := range gitPaths(out) {
					r.trunkPaths[path] = true
				}
			}
		}
	}
	return r.trunkPaths[rel]
}

// emptyTree names the object a reservation ref points at, so that ref carries nothing
// but its own name.
//
// Asked of git rather than written down: the well-known value is a SHA-1, and a
// repository whose objects are SHA-256 has a different one. Nothing is written and
// nothing needs to be — every repository knows the empty tree without being told, so it
// resolves in a project that has no commits at all. `--stdin` with no stdin is git
// reading the null device, which is what makes the tree empty.
func (r *Repo) emptyTree() (string, error) {
	out, err := r.gitRun("hash-object", "-t", "tree", "--stdin")
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(out)
	if sha == "" {
		return "", fmt.Errorf("git hash-object named no object for the empty tree")
	}
	return sha, nil
}

// commitDate is when a revision was last committed, or the zero time. How long a branch
// has waited is measured from here rather than from the doc's submitted:, which is a date
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
// refusal, so inventing files from a failed call would block a submit for no reason.
func (r *Repo) DirtyFiles() ([]string, error) {
	out, err := r.gitRun("status", "--porcelain", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
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
	return paths, nil
}

// commitPaths commits exactly the paths named and nothing else.
//
// The pathspec is on both halves: `add` stages those paths, and `commit` takes them as
// its own argument, so anything a concurrent process staged in between stays out of the
// commit. Paths are absolute, because a project root and a git top-level are not always
// the same directory.
func (r *Repo) commitPaths(message string, paths ...string) error {
	if _, err := r.gitRun(append([]string{"add", "--"}, paths...)...); err != nil {
		return err
	}
	_, err := r.gitRun(append([]string{"commit", "--quiet", "-m", message, "--"}, paths...)...)
	return err
}

// commitAll stages the whole working tree and commits it. `harvest` is the only caller:
// every other command commits the paths it wrote, and this one commits work it did not do.
func (r *Repo) commitAll(message string) error {
	if _, err := r.gitRun("add", "-A"); err != nil {
		return err
	}
	_, err := r.gitRun("commit", "--quiet", "-m", message)
	return err
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
	cmd.WaitDelay = waitDelay
	if out, err := cmd.Output(); err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return root
		}
	}
	return start
}
