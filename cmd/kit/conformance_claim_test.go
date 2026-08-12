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

// branches lists the project's local branch names.
func (p *project) branches() []string {
	p.t.Helper()
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
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
		{"a draft nobody settled", func(p *project) {
			feature(p, "0008", "unsettled", "Unsettled", "Draft", "- [ ] Something")
			p.commitAll("a draft")
		}, "Draft"},
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
