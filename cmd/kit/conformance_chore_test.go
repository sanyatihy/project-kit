package main

// Conformance: `kit chore`. The command exists so that work with no doc is coordinated
// the same way work with one is, so most of this asks git and the remote what happened
// rather than asserting on messages.

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// gitOutput runs git in a project and hands back what it printed, for the assertions
// that are about what a branch carries rather than about which branches exist.
func gitOutput(t *testing.T, p *project, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = p.Root, gitEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(out)
}

func TestAChoreIsClaimedByThePushAndCarriesItsOwnRecord(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)

	r := a.run("chore", "Fix the redirect loop").ok()
	r.has("0008-fix-the-redirect-loop", "the push is the claim", "docs/CHORES.md")

	// The claim is the ref on the remote, exactly as it is for a feature.
	if !hasRemoteBranch(t, a, "0008-fix-the-redirect-loop") {
		t.Error("chore exited 0 and the remote has no branch, so nothing claims the work")
	}
	// One id space: the chore took the number a feature would have taken next.
	if got := a.reservedIDs(); !slices.Contains(got, "0008") {
		t.Errorf("the remote reserves %v, none of which is the chore's id — a feature "+
			"can still be handed it", got)
	}

	// The record travels on the branch, so the worktree the agent is sent to has it.
	record := gitOutput(t, a, "show", "0008-fix-the-redirect-loop:docs/CHORES.md")
	for _, want := range []string{"- 0008 ", "Fix the redirect loop", "kit@example.invalid"} {
		if !strings.Contains(record, want) {
			t.Errorf("the branch's record never says %q:\n%s", want, record)
		}
	}

	// And this checkout is untouched: it is the trunk one, shared with whoever else
	// reads the board here, so a chore may not leave a change standing in it.
	if dirty := gitOutput(t, a, "status", "--porcelain"); strings.TrimSpace(dirty) != "" {
		t.Errorf("chore left this checkout dirty:\n%s", dirty)
	}
	if a.exists("docs/CHORES.md") {
		t.Error("chore wrote the record into the trunk checkout rather than onto the branch")
	}
}

func TestAChoreAndAFeatureCanNeverShareAnID(t *testing.T) {
	t.Parallel()
	// One space, so the branch and the record cite a number that names one thing. A
	// chore with an id of its own would collide with a feature's the first time either
	// side skipped a number.
	a, _ := twoAgents(t)
	a.run("chore", "Refresh the country table").ok().has("0008-refresh-the-country-table")

	a.run("new", "Later").ok().has("0009-later.md")
	if a.exists("docs/features/0008-later.md") {
		t.Error("new was handed the id the chore is holding")
	}
}

func TestASecondAgentSentTheSameChoreIsTurnedAway(t *testing.T) {
	t.Parallel()
	// The failure the whole feature is for: two agents given the same bug both start it.
	// The branch list comes from the remote, so the second agent is turned away by what
	// the first one pushed — a fetch away from nothing at all, which is why it fetches.
	a, b := twoAgents(t)
	a.run("chore", "Fix the redirect loop").ok()
	b.git("fetch", "-q", "origin")

	// Exit 1, the code that means "not yours, take something else": the work is real
	// and somebody has it.
	b.run("chore", "fix the REDIRECT loop!").fails(1).errHas("already claimed", "0008")

	for _, name := range b.branches() {
		if strings.Contains(name, "redirect") {
			t.Errorf("the turned-away agent kept branch %q", name)
		}
	}
}

func TestAChoreInFlightIsNotReportedAsAnOrphanClaim(t *testing.T) {
	t.Parallel()
	// A chore's branch cites an id no feature doc declares, which is what `orphan-claim`
	// looks for. Its own tree carries the record, and that is the positive evidence a
	// feature's branch gives by carrying its doc.
	a, b := twoAgents(t)
	a.run("chore", "Fix the redirect loop").ok()
	b.git("fetch", "-q", "origin")

	for _, p := range []*project{a, b} {
		if codes := p.auditCodes(); slices.Contains(codes, "orphan-claim") {
			t.Errorf("a chore in flight was reported as an orphan claim; audit said %v", codes)
		}
	}
}

func TestChoreRefusalsThatStopABadClaim(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		setUp func(p *project)
		title string
		says  string
	}{
		// The hole this feature closes at the other end: a branch citing a four-digit
		// number claims that feature, so a chore may not name itself into one.
		{"a title carrying a four-digit number", func(*project) {},
			"Fix the 2026 pricing", "four-digit number"},
		{"a title that slugs to nothing", func(*project) {}, "!!!", "no usable characters"},
		// One unit of work per checkout. Taking a chore from the worktree holding a
		// feature puts both on one branch, which is one pull request nobody can review.
		{"a checkout that already claims a feature", func(p *project) {
			p.run("claim", "0007").ok()
			p.git("checkout", "-q", "0007-widget")
		}, "Fix the redirect loop", "one unit of work per checkout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := twoAgents(t)
			tc.setUp(a)

			a.run("chore", tc.title).fails(2).errHas(tc.says)

			for _, name := range a.branches() {
				if strings.HasPrefix(name, "0008") {
					t.Errorf("a refused chore still created branch %q", name)
				}
			}
			if got := a.reservedIDs(); slices.Contains(got, "0008") {
				t.Errorf("a refused chore burned an id: the remote reserves %v", got)
			}
		})
	}
}

func TestAChoreWithNoRemoteSaysWhatItCannotGive(t *testing.T) {
	t.Parallel()
	// The same bargain `new` and `claim` make: a project with no remote still works,
	// and the run says out loud that nothing reserved the id and nobody else can see
	// the claim.
	p := cleanProject(t)
	p.commitAll("the plan")

	r := p.run("chore", "Tidy the scaffolding").ok()
	r.has("0002-tidy-the-scaffolding", "no remote", "nothing reserved 0002")
}
