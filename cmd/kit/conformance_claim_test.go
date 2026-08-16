package main

// Conformance: `kit claim`. The command exists for one situation — two agents
// reaching for the same feature — so most of this file sets that situation up for
// real, with a bare remote and two clones, rather than asserting on messages.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// projectAt wraps a directory that is not the one t.TempDir handed out, so a test
// can put several projects and a bare remote side by side.
func projectAt(t *testing.T, root string) *project {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("preparing %s: %v", root, err)
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return &project{t: t, Root: root}
}

// gitIn runs git somewhere that is not a project, for setting up the remote.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = dir, gitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// twoAgents builds the situation the whole feature exists for: one bare remote
// carrying an Accepted feature, and two clones of it that have each other's work
// nowhere in sight.
//
// Both sit on the same commit, which is the case that matters and the one that is
// easy to get wrong. Pushing a branch to a remote that already has it, at the very
// commit being pushed, is "Everything up-to-date" and exits 0 — so a claim that
// trusts the exit code hands one feature to both of them.
func twoAgents(t *testing.T) (*project, *project) {
	t.Helper()
	base := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(base); err == nil {
		base = resolved
	}
	origin := filepath.Join(base, "origin.git")
	gitIn(t, base, "init", "-q", "--bare", "-b", "main", origin)

	a := projectAt(t, filepath.Join(base, "a"))
	a.run("init").ok()
	feature(a, "0007", "widget", "A widget", "Accepted", "- [ ] Build it")
	a.git("remote", "add", "origin", origin)
	a.commitAll("the plan")
	a.git("push", "-q", "origin", "HEAD:refs/heads/main")

	gitIn(t, base, "clone", "-q", origin, filepath.Join(base, "b"))
	return a, projectAt(t, filepath.Join(base, "b"))
}

// resolves reports whether a ref exists in this checkout, for tests that turn on the
// difference between knowing a branch's name and having its objects.
func (p *project) resolves(ref string) bool {
	p.t.Helper()
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	cmd.Dir, cmd.Env = p.Root, gitEnv()
	return cmd.Run() == nil
}

// headBranch is the branch this checkout is on. Read rather than assumed, because the
// fixtures deliberately inherit no init.defaultBranch and so may start on either name.
func (p *project) headBranch() string {
	p.t.Helper()
	cmd := exec.Command("git", "symbolic-ref", "--quiet", "--short", "HEAD")
	cmd.Dir, cmd.Env = p.Root, gitEnv()
	out, err := cmd.Output()
	if err != nil {
		p.t.Fatalf("reading HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// remoteBranches lists the remote-tracking refs this checkout has cached, which is the
// state that goes stale and the reason these tests exist.
func (p *project) remoteBranches() []string {
	p.t.Helper()
	return p.branchesWith("-r")
}

// branches lists the project's local branch names.
func (p *project) branches() []string {
	p.t.Helper()
	return p.branchesWith()
}

func (p *project) branchesWith(extra ...string) []string {
	p.t.Helper()
	cmd := exec.Command("git", append([]string{"branch"}, append(extra,
		"--format=%(refname:short)")...)...)
	cmd.Dir, cmd.Env = p.Root, gitEnv()
	out, err := cmd.Output()
	if err != nil {
		p.t.Fatalf("listing branches: %v", err)
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func TestClaimPushesTheBranchAndSaysSo(t *testing.T) {
	a, _ := twoAgents(t)

	a.run("claim", "0007").ok().has("claimed", "0007-widget", "git worktree add")

	// The claim is the ref on the remote, so it is the remote that has to have it.
	// A local branch alone is the state the old protocol could not distinguish from
	// a claim nobody else can see.
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", "0007-widget")
	cmd.Dir, cmd.Env = a.Root, gitEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("reading the remote: %v", err)
	}
	if !strings.Contains(string(out), "refs/heads/0007-widget") {
		t.Errorf("claim exited 0 but the remote has no 0007-widget; got %q", out)
	}
}

// The test this feature is for.
func TestTwoAgentsClaimingAtOnceProduceExactlyOneWinner(t *testing.T) {
	a, b := twoAgents(t)

	first := a.run("claim", "0007")
	// b cloned before a claimed, so its view of the remote is stale in exactly the
	// way a real second agent's is: nothing it can read locally says 0007 is taken.
	second := b.run("claim", "0007")

	if first.Code != 0 {
		t.Fatalf("the first agent should have won, exited %d%s", first.Code, first.context())
	}
	if second.Code != 1 {
		t.Fatalf("the second agent exited %d, want 1 — both agents believe they own "+
			"0007%s", second.Code, second.context())
	}
	second.errHas("not claimed")
}

func TestLosingAClaimLeavesNoBranchBehind(t *testing.T) {
	// A branch left over from a lost race is indistinguishable from a claim, so the
	// loser would go on to be reported as a second claimant of a feature it does not
	// have — and audit would call it a split claim.
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.run("claim", "0007").fails(1)

	for _, name := range b.branches() {
		if strings.HasPrefix(name, "0007") {
			t.Errorf("the losing agent kept branch %q", name)
		}
	}
}

func TestClaimingTwiceFromTheSameCheckoutLosesTheSecondTime(t *testing.T) {
	// The same guarantee seen from one side: the second call did not take anything,
	// and saying otherwise would make the command useless for deciding who works.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	a.run("claim", "0007").fails(1).errHas("already claimed", "0007-widget")
}

func TestClaimRefusalsThatAreNotRaces(t *testing.T) {
	// All of these mean "not yours, take another" and share the exit code with a
	// lost race, because that is the same instruction to the agent that gets one.
	for _, tc := range []struct {
		name  string
		setUp func(p *project)
		says  string
	}{
		// A Draft is claimable — status orders the board and does not gate it. What is
		// refused is a doc nobody wrote, whatever its status: `unfilled-template`
		// exempts Draft, so without this a doc straight out of `kit new` would be
		// handed to an agent with three empty sections and every command exiting 0.
		{"a doc nobody wrote", func(p *project) {
			unwritten(p, "0008", "unwritten")
			p.commitAll("an empty doc")
		}, "no plan to build"},
		{"a feature that already shipped", func(p *project) {
			feature(p, "0008", "done", "Done", "Shipped", "- [x] Finished")
			p.commitAll("shipped work")
		}, "Shipped"},
		{"a feature a human has to unblock", func(p *project) {
			feature(p, "0008", "stuck", "Stuck", "Accepted", "- [ ] Something")
			p.commitAll("the plan")
			p.run("block", "0008", "Which way round?").ok()
		}, "blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := twoAgents(t)
			tc.setUp(a)
			a.run("claim", "0008").fails(1).errHas("not claimed", tc.says)

			for _, name := range a.branches() {
				if strings.HasPrefix(name, "0008") {
					t.Errorf("refusing to claim still left branch %q", name)
				}
			}
		})
	}
}

func TestTheFirstClaimOfAFreshProjectIsNotRefused(t *testing.T) {
	// The trunk barrier must not break a project that has no trunk to land on. A fresh
	// `kit init` with one branch and no remote is how every project starts, and there
	// the doc is on the only branch there is: `Trunk()` falls back to that branch, so
	// the doc is on the trunk and the check passes rather than being skipped. The skip
	// itself is for the states where no trunk resolves at all — a detached HEAD, or a
	// checkout whose only branch is claim-shaped — and this is the case that would be
	// broken if the barrier assumed a trunk exists.
	p := cleanProject(t)
	p.commitAll("the plan")

	p.run("claim", "0001").ok().has("claimed", "0001-thing", "no remote")
}

func TestClaimRefusesAnIDTwoDocsShare(t *testing.T) {
	// Exit 2, not 1: this is not someone else holding the feature, it is a question
	// with no answer. One branch cannot claim one of two features, and picking the
	// first doc silently is how the ID stops being a primary key.
	a, _ := twoAgents(t)
	a.write("docs/features/0007-widget-as-well.md", doc("0007", "Widget as well",
		"Accepted", "", "- [ ] Something"))
	a.commitAll("a second doc with the same id")

	a.run("claim", "0007").fails(2).errHas("share id 0007")
}

func TestClaimRefusesADocWhoseIDAndFilenameDisagree(t *testing.T) {
	// Exit 1 and before any git call. `Branch()` comes from the filename and the board
	// looks a claim up by the id:, so claiming this creates a branch nothing connects
	// to the feature — the board goes on offering it and the next agent gets a raw git
	// error about a branch that already exists. One rename fixes it, so this is
	// NotYours: take something else meanwhile, the same instruction a lost race carries.
	a, _ := twoAgents(t)
	a.write("docs/features/0002-confused.md",
		doc("0009", "Confused", "Draft", "", "- [ ] Something"))
	a.commitAll("a doc whose id and filename disagree")

	r := a.run("claim", "0009").fails(1)
	r.errHas("not claimed", "0009", "0002")

	// Nothing left behind under either number, locally or on the remote — the branch
	// this would have created is the one the board cannot see.
	for _, name := range a.branches() {
		if strings.HasPrefix(name, "0002") || strings.HasPrefix(name, "0009") {
			t.Errorf("a refused claim still created branch %q", name)
		}
	}
	cmd := exec.Command("git", "ls-remote", "--heads", "origin")
	cmd.Dir, cmd.Env = a.Root, gitEnv()
	if out, err := cmd.Output(); err != nil {
		t.Fatalf("reading the remote: %v", err)
	} else if strings.Contains(string(out), "confused") {
		t.Errorf("a refused claim pushed a branch anyway: %q", out)
	}

	// The quiet half, in the same project: a doc whose two agree still claims fine.
	a.run("claim", "0007").ok().has("claimed", "0007-widget")
}

func TestAClaimMadeBeforeTheFixStaysVisible(t *testing.T) {
	// Belt and braces for docs already in this state. `claim` refuses to create such a
	// claim now, so the branch here is made the way the old `claim` made it: named from
	// the filename, mentioning nothing the board searched for. Matching the filename
	// prefix as well as the id is what keeps it visible rather than orphaning it.
	p := cleanProject(t)
	p.write("docs/features/0002-confused.md",
		doc("0009", "Confused", "Draft", "", "- [ ] Something"))
	p.commitAll("a doc whose id and filename disagree")

	// The quiet half first: with no branch there is no claim, and matching the prefix
	// must not invent one out of a feature nobody has taken. Blocked, not ready — the
	// board withholds a mismatched doc because `claim` refuses it — but the point here
	// is that it is not reported as *taken*.
	if got := boardGroup(p.plan("next", "--json"), "0009"); got != "blocked" {
		t.Errorf("with no branch, 0009 is %s on the board, want blocked", got)
	}

	p.git("branch", "0002-confused")

	plan := p.plan("next", "--json")
	if got := boardGroup(plan, "0009"); got != "taken" {
		t.Errorf("0009 is %s on the board while 0002-confused holds it, want taken", got)
	}
	for _, item := range plan.Taken {
		if item.ID == "0009" && item.ClaimedBy != "0002-confused" {
			t.Errorf("the board names %q as the claimant, want 0002-confused", item.ClaimedBy)
		}
	}
}

// boardGroup says which group `next` put a feature in, which is the whole question a
// claim being visible turns on. Read from --json because that is the format agents
// parse, and because "ready" and "taken" are one word apart in the rendered page.
func boardGroup(plan kit.Plan, id string) string {
	for _, group := range []struct {
		name  string
		items []kit.Item
	}{
		{"yours", plan.Yours}, {"ready", plan.Ready},
		{"taken", plan.Taken}, {"blocked", plan.Blocked},
	} {
		for _, item := range group.items {
			if item.ID == id {
				return group.name
			}
		}
	}
	return "absent"
}

func TestAMismatchedFeatureIsNeverHandedOutTwice(t *testing.T) {
	// The whole scenario, as measured: `claim 0009` created `0002-confused`, `next` went
	// on saying "start here 0009", and the second agent got exit 2 with a raw git error —
	// the code documented as "it could not run", so it escalates instead of moving on.
	//
	// The first agent's claim is made by hand because `claim` now refuses to make one;
	// what is asserted is that from here nobody is handed the feature a second time.
	a, b := twoAgents(t)
	a.write("docs/features/0002-confused.md",
		doc("0009", "Confused", "Draft", "", "- [ ] Something"))
	a.commitAll("a doc whose id and filename disagree")
	a.git("push", "-q", "origin", "HEAD:refs/heads/main")
	a.git("push", "-q", "origin", "HEAD:refs/heads/0002-confused")
	b.git("fetch", "-q", "origin")
	b.git("merge", "-q", "--ff-only", "origin/main")

	if got := boardGroup(b.plan("next", "--json"), "0009"); got != "taken" {
		t.Errorf("the second agent's board says 0009 is %s, want taken — it is being "+
			"handed a feature somebody already holds", got)
	}
	b.run("next").ok().lacks("kit claim 0009")

	// Exit 1, not the exit 2 this used to produce, and no git talking to an agent that
	// has no idea what a ref is. Whichever refusal answers first, the instruction is the
	// same: this is not yours to take, and here is the one edit that unsticks it.
	second := b.run("claim", "0009").fails(1)
	second.errHas("not claimed")
	second.lacks("fatal:", "already exists")
}

func TestClaimWithoutARemoteSaysNobodyElseCanSeeIt(t *testing.T) {
	// Most projects have no remote on day one, so refusing would make the kit
	// unusable before the first push. It still has to say what the claim is worth.
	p := cleanProject(t)
	p.commitAll("the plan")

	r := p.run("claim", "0001").ok()
	r.has("claimed", "0001-thing", "no remote")
	if !strings.Contains(r.out(), "no other agent can see it") {
		t.Errorf("a local-only claim did not say it is invisible%s", r.context())
	}
}

func TestClaimUsageErrorsAreExitTwo(t *testing.T) {
	a, _ := twoAgents(t)
	a.run("claim").fails(2)
	a.run("claim", "0099").fails(2).errHas("no feature")
}

func TestClaimRefusesWhenTheBranchWouldCarryNoPlan(t *testing.T) {
	// The claim pushes HEAD, so a plan that is only in the working tree does not travel
	// with it. This exited 0, reported a won claim, and sent the agent to a worktree
	// holding nothing but the README — where `kit next` said the project was not set up.
	a, _ := twoAgents(t)
	feature(a, "0008", "uncommitted", "Uncommitted", "Accepted", "- [ ] Something")
	// Deliberately not committed.

	r := a.run("claim", "0008").fails(2)
	r.errHas("not committed", "carries no plan")

	// Exit 2 and not 1: nobody else holds this, so "take another" would be wrong advice.
	// And nothing may be left behind, locally or on the remote.
	for _, name := range a.branches() {
		if strings.HasPrefix(name, "0008") {
			t.Errorf("a refused claim still created branch %q", name)
		}
	}
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", "0008-uncommitted")
	cmd.Dir, cmd.Env = a.Root, gitEnv()
	if out, err := cmd.Output(); err != nil {
		t.Fatalf("reading the remote: %v", err)
	} else if strings.Contains(string(out), "0008") {
		t.Errorf("a refused claim pushed a branch anyway: %q", out)
	}
}

func TestAMergedAndDeletedBranchIsNotAClaim(t *testing.T) {
	// The whole loop the kit exists to run: a draft is proposed on a branch, the pull
	// request merges, and the branch is deleted. Git prunes remote-tracking refs on
	// neither fetch nor pull, so the ref survived and the feature was taken for good —
	// `next` said to stop and wait, `claim` reported a lost race against a branch that
	// existed nowhere, and `audit` said the docs and the repo agreed.
	a, b := twoAgents(t)

	// b holds a fetched view of the branch; a is where the merge happens.
	a.run("claim", "0007").ok()
	b.git("fetch", "origin")
	a.git("push", "origin", "--delete", "0007-widget")

	// b pulls, as any agent would after a merge. `git pull` does not prune.
	b.git("fetch", "origin")
	if !strings.Contains(strings.Join(b.remoteBranches(), " "), "0007-widget") {
		t.Skip("this git prunes on fetch, so the stale ref cannot be set up")
	}

	if r := b.run("next").ok(); strings.Contains(r.out(), "taken") {
		t.Errorf("a deleted branch was still reported as a claim%s", r.context())
	}
	b.run("claim", "0007").ok().has("claimed", "0007-widget")
}

func TestOfflineReadsTheMirrorAndAcceptsTheCost(t *testing.T) {
	// --offline is the escape for a network that is not there, and the cost is stated
	// rather than hidden: claims come from the last fetch, so a branch deleted since then
	// still reads as one. Pinned so nobody "fixes" it into a silent network call.
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("fetch", "origin")
	a.git("push", "origin", "--delete", "0007-widget")
	b.git("fetch", "origin")
	if !strings.Contains(strings.Join(b.remoteBranches(), " "), "0007-widget") {
		t.Skip("this git prunes on fetch, so the stale ref cannot be set up")
	}

	if r := b.run("next", "--offline").ok(); !strings.Contains(r.out(), "taken") {
		t.Errorf("--offline did not read claims from the mirror%s", r.context())
	}
	if r := b.run("next").ok(); strings.Contains(r.out(), "taken") {
		t.Errorf("without --offline the remote should have been asked%s", r.context())
	}
}

func TestAnUnreachableRemoteNamesPruneRatherThanAPhantomDelete(t *testing.T) {
	// When the remote cannot be asked, claims fall back to the mirror — and the mirror is
	// what names branches nobody holds. The refusal has to say so, because the advice it
	// used to print unconditionally (`git push --delete` then `git branch -d`) fails with
	// two errors when the branch is already gone from both places.
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("fetch", "origin")
	a.git("push", "origin", "--delete", "0007-widget")
	b.git("fetch", "origin")
	if !strings.Contains(strings.Join(b.remoteBranches(), " "), "0007-widget") {
		t.Skip("this git prunes on fetch, so the stale ref cannot be set up")
	}
	b.git("remote", "set-url", "origin", filepath.Join(b.Root, "does-not-exist.git"))

	b.run("claim", "0007").fails(1).errHas("fetch --prune")
}

func TestClaimTakesADraft(t *testing.T) {
	// The gate that moved. `status:` orders the board and does not decide who may
	// work: an agent takes a Draft, settles what the doc leaves open, and the human
	// reviews the implementation — which is the review that was always going to
	// happen. Accepting first is a signal, not a precondition.
	p := cleanProject(t)
	feature(p, "0004", "drafted", "Drafted", "Draft", "- [ ] Build it")
	p.commitAll("a draft on the trunk")

	p.run("claim", "0004").ok().has("claimed", "0004-drafted")
}

func TestClaimRefusesADocNobodyWrote(t *testing.T) {
	// The load-bearing half of the above. unfilled-template exempts Draft, and that
	// exemption was only safe while a Draft could not be claimed — without this, a
	// doc straight out of `kit new` is handed to an agent to build and every command
	// exits 0.
	p := cleanProject(t)
	unwritten(p, "0004", "unwritten")
	p.commitAll("an empty doc")

	r := p.run("claim", "0004").fails(1)
	r.errHas("no plan to build")
	// Naming the sections, because "write the plan" is not actionable on its own.
	r.errHas("Problem")
}

func TestClaimRefusesWhatItsNeedsHaveNotShipped(t *testing.T) {
	// Measured on a fresh project: `kit next` said 0002 was waiting on 0001 and
	// `kit claim 0002` took it anyway, so an agent claiming by ID built on a foundation
	// that is not there. Exit 1, like a lost race — the instruction is the same one.
	p := cleanProject(t)
	p.write("docs/features/0002-store.md",
		doc("0002", "Store", "Accepted", "needs: 0001", "- [ ] Do it"))
	p.commitAll("a chain of two")

	r := p.run("claim", "0002").fails(1)
	r.errHas("not claimed", "0001")

	// Nothing left behind, locally or on the remote — a branch is a claim.
	for _, name := range p.branches() {
		if strings.HasPrefix(name, "0002") {
			t.Errorf("a refused claim still created branch %q", name)
		}
	}
}

func TestClaimTakesAFeatureWhoseNeedsHaveShipped(t *testing.T) {
	// The quiet half. A dependency that has shipped is not an obstacle, and refusing
	// here would make `needs:` a field that never releases anything.
	p := cleanProject(t)
	p.write("docs/features/0001-thing.md",
		doc("0001", "A thing", "Shipped", "shipped: 2026-01-02", "- [x] Do the thing"))
	p.write("docs/features/0002-store.md",
		doc("0002", "Store", "Accepted", "needs: 0001", "- [ ] Do it"))
	p.commitAll("one shipped, one waiting on it")

	p.run("claim", "0002").ok().has("claimed", "0002-store")
}

func TestNextAndClaimAgreeAboutAnUnwrittenDoc(t *testing.T) {
	// A board that offers what the next command refuses is the loop 0013 exists to
	// stop: the agent follows the documented path and is turned away at the end of it.
	p := cleanProject(t)
	unwritten(p, "0004", "unwritten")
	p.commitAll("an empty doc")

	r := p.run("next").ok()
	r.has("blocked", "no plan yet")
	r.lacks("kit claim 0004")
}

func TestTheBoardDoesNotOfferWhatClaimRefuses(t *testing.T) {
	// The loop 0013 exists to stop: `claim` now refuses a mismatched doc, so a board
	// that still lists it under "start here" sends every agent to a command that turns
	// it away. Any refusal a command makes, the board has to reflect — the converse of
	// the rule 0023 established for needs:.
	p := cleanProject(t)
	p.write("docs/features/0002-confused.md",
		doc("0009", "Confused", "Accepted", "", "- [ ] Something"))
	p.commitAll("a doc whose id and filename disagree")

	if got := boardGroup(p.plan("next", "--json"), "0009"); got != "blocked" {
		t.Errorf("next put the mismatched feature in %q, want blocked — claim refuses it", got)
	}
	p.run("next").ok().has("rename one")

	// And a doc whose names agree is still offered, so this cannot fire on a healthy one.
	if got := boardGroup(p.plan("next", "--json"), "0001"); got != "ready" {
		t.Errorf("an agreeing doc landed in %q, want ready", got)
	}
}
