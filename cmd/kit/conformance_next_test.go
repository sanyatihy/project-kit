package main

// Conformance: `kit next`. This is the command an agent runs first in every
// session, so what it hands back — and what it refuses to hand back — is the whole
// contract.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// feature writes a minimal but valid feature doc. The template's full shape is
// covered by the `new` suite; what matters here is status, ID, and task list.
// ask records an open question in a feature doc, which is the only place one lives.
func ask(p *project, rel, question string) { askOn(p, rel, "2026-01-01", question) }

// askOn is ask for the checks that turn on how old the question is.
func askOn(p *project, rel, date, question string) {
	p.t.Helper()
	p.write(rel, p.read(rel)+"\n## Open questions\n\n- [ ] "+date+" · "+question+"\n")
}

func feature(p *project, id, slug, title, status, tasks string) {
	p.t.Helper()
	p.write("docs/features/"+id+"-"+slug+".md", fmt.Sprintf(
		`---
id: %s
title: %s
status: %s
created: 2026-01-01
---

## Problem
Something needs doing.

## Design
This way.

## Tasks
%s

## Log
- 2026-01-01 — started.
`, id, title, status, tasks))
}

// unwritten writes a doc straight out of `kit new`: the headings are there and
// nothing under them. `claim` refuses it whatever its status.
func unwritten(p *project, id, slug string) {
	p.t.Helper()
	p.write("docs/features/"+id+"-"+slug+".md",
		"---\nid: "+id+"\ntitle: Unwritten\nstatus: Draft\ncreated: 2026-01-01\n---\n\n"+
			"## Problem\n\n## Design\n\n## Tasks\n\n## Log\n")
}

// plan runs `kit next --json` and returns this project's plan, for the assertions
// that are about the structure rather than the printed page.
func (p *project) plan(args ...string) kit.Plan {
	p.t.Helper()
	r := p.run(args...).ok()
	var plan kit.Plan
	if err := json.Unmarshal([]byte(r.Stdout), &plan); err != nil {
		p.t.Fatalf("next --json emitted invalid JSON: %v\n%s", err, r.Stdout)
	}
	return plan
}

func TestNextHandsBackTheTopmostUntickedBox(t *testing.T) {
	t.Parallel()
	// List order is the order of work, which is what lets ordering need no syntax.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft",
		"- [x] Schema\n- [ ] The POST handler\n- [ ] The redirect")

	r := p.run("next").ok()
	r.has("start here", "0001", "The POST handler")
	r.lacks("The redirect")
}

func TestFinishingBeatsStartingWithoutAStatusToSaySo(t *testing.T) {
	t.Parallel()
	// What `In progress` used to order, done by the groups instead. In the checkout
	// holding a claim, that feature is what `next` names — ahead of anything free, and
	// with no status anywhere asserting that work had started.
	p := setUp(t)
	feature(p, "0001", "first", "First", "Draft", "- [ ] Do the first thing")
	feature(p, "0002", "second", "Second", "Draft", "- [ ] Do the second thing")
	p.commitAll("two features")
	p.git("checkout", "-q", "-b", "0002-second")

	plan := p.plan("next", "--json")
	if len(plan.Yours) != 1 || plan.Yours[0].ID != "0002" {
		t.Fatalf("the checkout's own claim is not what next names: %+v", plan.Yours)
	}
	if plan.Yours[0].Status == "In progress" {
		t.Errorf("0002 carries a status the kit no longer has: %+v", plan.Yours[0])
	}
	out := p.run("next").ok().Stdout
	if strings.Index(out, "0002") > strings.Index(out, "0001") {
		t.Errorf("free 0001 was listed before the claim this checkout holds:\n%s", out)
	}
}

func TestNextTreatsABranchAsAClaim(t *testing.T) {
	t.Parallel()
	// The claim is a branch name and nothing else — no lock file, no state to sync.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")
	p.commitAll("add the feature doc")
	p.git("branch", "0001-shorten")

	r := p.run("next").ok()
	r.has("taken", "0001-shorten")
	// A taken feature must not also be offered as the thing to start.
	if strings.Contains(r.Stdout, "start here") {
		t.Errorf("a claimed feature was still offered as work:\n%s", r.Stdout)
	}
}

func TestNextReportsThisCheckoutsOwnClaimAsYours(t *testing.T) {
	t.Parallel()
	// Measured before this existed: standing in the checkout that claims 0001,
	// `next` reported 0001 as taken — by you — which is the tool stating a fact
	// about the reader back to them as somebody else's.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")
	p.commitAll("add the feature doc")
	p.git("checkout", "-q", "-b", "0001-shorten")

	r := p.run("next").ok()
	r.has("yours", "0001", "The POST handler")
	if strings.Contains(r.Stdout, "taken") {
		t.Errorf("the checkout's own claim was reported as taken:\n%s", r.Stdout)
	}
}

func TestNextJSONSeparatesYoursFromTaken(t *testing.T) {
	t.Parallel()
	// --json is the interface agents actually read, so the distinction has to exist
	// there and not only in the prose above it.
	p := setUp(t)
	feature(p, "0001", "mine", "Mine", "Draft", "- [ ] Do it")
	feature(p, "0002", "theirs", "Theirs", "Draft", "- [ ] Do that")
	p.commitAll("two features")
	p.git("branch", "0002-theirs")
	p.git("checkout", "-q", "-b", "0001-mine")

	plan := p.plan("next", "--json")
	if len(plan.Yours) != 1 || plan.Yours[0].ID != "0001" {
		t.Errorf("yours = %+v, want just 0001", plan.Yours)
	}
	if !plan.Yours[0].Mine {
		t.Error("the checkout's own claim is not flagged mine")
	}
	if len(plan.Taken) != 1 || plan.Taken[0].ID != "0002" {
		t.Errorf("taken = %+v, want just 0002", plan.Taken)
	}
	if plan.Claiming != "0001" {
		t.Errorf("claiming = %q, want 0001", plan.Claiming)
	}
}

func TestNextReportsProgressThroughTheBoxes(t *testing.T) {
	t.Parallel()
	// With one feature the next box is the whole state. With several the question is
	// which are moving, and only a count answers it. In --json too, because that is
	// what anything watching the plan from outside has to read.
	p := setUp(t)
	feature(p, "0001", "part", "Partly done", "Draft",
		"- [x] One\n- [x] Two\n- [ ] Three\n- [ ] Four")
	p.commitAll("a feature part-done")

	p.run("next").ok().has("2/4")

	plan := p.plan("next", "--json")
	all := append(append([]kit.Item{}, plan.Yours...), plan.Ready...)
	if len(all) != 1 {
		t.Fatalf("expected one open feature, got %+v", all)
	}
	if all[0].Done != 2 || all[0].Total != 4 {
		t.Errorf("done/total = %d/%d, want 2/4", all[0].Done, all[0].Total)
	}
}

func TestNextWaitsForDeclaredNeedsAndReleasesOnShip(t *testing.T) {
	t.Parallel()
	// The whole point of the field: what is ready is the answer to "can these be
	// worked at once", so it has to be right in both directions — refusing work that
	// cannot start, and releasing it the moment the dependency ships.
	p := setUp(t)
	feature(p, "0001", "base", "Base", "Draft", "- [ ] Do it")
	p.write("docs/features/0002-store.md",
		doc("0002", "Store", "Draft", "needs: 0001", "- [ ] Do it"))
	p.commitAll("a chain of two")

	plan := p.plan("next", "--json")
	if len(plan.Ready) != 1 || plan.Ready[0].ID != "0001" {
		t.Errorf("ready = %+v, want just 0001", plan.Ready)
	}
	if len(plan.Blocked) != 1 || plan.Blocked[0].ID != "0002" {
		t.Fatalf("blocked = %+v, want just 0002", plan.Blocked)
	}
	if !slices.Contains(plan.Blocked[0].Needs, "0001") {
		t.Errorf("blocked 0002 needs = %v, want it to name 0001", plan.Blocked[0].Needs)
	}
	p.run("next").ok().has("needs 0001 to merge first")

	// With no remote, the submitted snapshot on local trunk is integrated.
	p.write("docs/features/0001-base.md",
		doc("0001", "Base", "Submitted", "submitted: 2026-01-02", "- [x] Do it"))
	p.commitAll("submit 0001")

	plan = p.plan("next", "--json")
	if len(plan.Blocked) != 0 {
		t.Errorf("blocked = %+v, want nothing once the dependency submitted", plan.Blocked)
	}
	if len(plan.Ready) != 1 || plan.Ready[0].ID != "0002" {
		t.Errorf("ready = %+v, want 0002 released", plan.Ready)
	}
}

func TestNextSaysWhenADependencyCanNeverShip(t *testing.T) {
	t.Parallel()
	// The board promised a wait that never ends: "waiting on: needs 0001 to merge first"
	// about a feature that is Dropped. The two states need opposite fixes — one ends on
	// its own, the other only when somebody edits needs: — so the line has to say which.
	dead := setUp(t)
	dead.write("docs/features/0001-base.md", doc("0001", "Base", "Dropped", "", "- [ ] Do it"))
	dead.write("docs/features/0002-store.md",
		doc("0002", "Store", "Draft", "needs: 0001", "- [ ] Do it"))

	r := dead.run("next").ok()
	r.has("blocked", "0002", "can never submit")
	r.lacks("to merge first")

	// The whole sentence, read from --json rather than the page: the printed line is
	// clipped for a narrow terminal, and agents read this format.
	blocked := dead.plan("next", "--json").Blocked
	if len(blocked) != 1 {
		t.Fatalf("blocked = %+v, want just 0002", blocked)
	}
	if !strings.Contains(blocked[0].WaitingOn, "Dropped") {
		t.Errorf("waiting_on = %q, want it to name the status that can never submit",
			blocked[0].WaitingOn)
	}

	// And a dependency that simply has not submitted yet keeps the wait that does end.
	pending := setUp(t)
	feature(pending, "0001", "base", "Base", "Draft", "- [ ] Do it")
	pending.write("docs/features/0002-store.md",
		doc("0002", "Store", "Draft", "needs: 0001", "- [ ] Do it"))
	pending.run("next").ok().has("needs 0001 to merge first").lacks("can never submit")
}

func TestNextAlwaysEndsWithAnInstruction(t *testing.T) {
	t.Parallel()
	// The invariant, not any particular sentence. A state the tool can reach and
	// does not close is a state something else decides — which is how an agent
	// facing a board of taken and blocked features started working on one of them.
	// Checked against the guidance the plan carries rather than the rendered text:
	// the text is wrapped for a narrow terminal, so asserting phrases in it breaks
	// on line breaks rather than on the thing being tested.
	tell := []string{"kit start", "start a round", "nothing is free", "write the plan in"}
	ends := func(out string) bool {
		for _, phrase := range tell {
			if strings.Contains(out, phrase) {
				return true
			}
		}
		return false
	}

	t.Run("nothing exists", func(t *testing.T) {
		p := setUp(t)
		if out := p.run("next").ok().Stdout; !ends(out) {
			t.Errorf("no instruction:\n%s", out)
		}
	})
	t.Run("something ready", func(t *testing.T) {
		p := setUp(t)
		feature(p, "0001", "go", "Go", "Draft", "- [ ] Do it")
		if out := p.run("next").ok().Stdout; !ends(out) {
			t.Errorf("no instruction:\n%s", out)
		}
	})
	t.Run("one doc, and nobody wrote it", func(t *testing.T) {
		// The state `kit new` leaves behind, and the documented next step from it. The
		// board has one blocked feature and nothing free, so the "nothing is free" arm
		// used to answer — telling an agent to stop and wait for somebody else to write
		// the doc it had just created itself.
		p := setUp(t)
		unwritten(p, "0001", "unwritten")
		out := p.run("next").ok().Stdout
		if !ends(out) {
			t.Errorf("no instruction:\n%s", out)
		}
		if !strings.Contains(out, "docs/features/0001-unwritten.md") {
			t.Errorf("the guidance never names the doc to write:\n%s", out)
		}
		if strings.Contains(out, "Stopping here is the right outcome") {
			t.Errorf("next told the agent to wait for a doc only it can write:\n%s", out)
		}
	})
	t.Run("everything taken or blocked", func(t *testing.T) {
		// The state that actually failed: one feature claimed by somebody else, the
		// rest waiting on it. Nothing for this agent, and it must be told so.
		p := setUp(t)
		feature(p, "0001", "taken", "Taken", "Draft", "- [ ] Do it")
		p.write("docs/features/0002-waiting.md",
			doc("0002", "Waiting", "Draft", "needs: 0001", "- [ ] Do it"))
		p.commitAll("two features")
		p.git("branch", "0001-taken")

		out := p.run("next").ok().Stdout
		if !strings.Contains(out, "nothing is free") {
			t.Errorf("the state that broke it does not close:\n%s", out)
		}
		if !strings.Contains(out, "not yours") {
			t.Errorf("nothing tells the agent the claimed feature is off limits:\n%s", out)
		}
	})
}

func TestNextIsBlockedByAnOpenBlockingQuestion(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")
	ask(p, "docs/features/0001-shorten.md", "Should codes expire?")

	r := p.run("next").ok()
	r.has("blocked", "waiting on", "Should codes expire?")
	if strings.Contains(r.Stdout, "start here") {
		t.Errorf("a blocked feature was offered as work:\n%s", r.Stdout)
	}
}

func TestNextBlocksAFeatureWithEveryBoxTicked(t *testing.T) {
	t.Parallel()
	// Nothing left to do is not the same as ready to work: it needs submitting or a
	// next task, and silently offering nothing would hide that.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [x] Done\n- [x] Also done")

	p.run("next").ok().has("blocked", "every task is ticked")
}

func TestNextOffersADraftAsWork(t *testing.T) {
	t.Parallel()
	// `status:` orders the board and does not gate it. A Draft is invented and not yet
	// blessed, which is a reason to rank it last and no reason to withhold it — the
	// operator reviews the implementation either way, and stopping every agent until
	// they transition a field is a stop before a review rather than an extra one.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")

	r := p.run("next").ok()
	r.has("start here", "0001", "kit start 0001")
	r.lacks("waiting on a human")
}

func TestNextDoesNotLimitIndependentAgents(t *testing.T) {
	t.Parallel()
	// Counted from claims, which is the only record that somebody started something.
	const warning = "in progress at once"

	claimed := func(p *project, ids ...string) {
		for _, id := range ids {
			feature(p, id, id, "Feature "+id, "Draft", "- [ ] Task "+id)
		}
		p.commitAll("the features")
		for _, id := range ids {
			p.git("branch", id+"-"+id)
		}
	}

	atLimit := setUp(t)
	claimed(atLimit, "0001", "0002")
	atLimit.run("next").ok().lacks(warning)

	over := setUp(t)
	claimed(over, "0001", "0002", "0003")
	over.run("next").ok().lacks(warning)
}

func TestNextPrintsTheCommandThatStartsTheWork(t *testing.T) {
	t.Parallel()
	// The output is meant to be actionable without reading the docs first.
	//
	// The command is the claim, not the worktree. It used to print `git worktree add
	// -b`, which builds somewhere to work before knowing the work is yours — and
	// skips the only step that settles who gets it. `kit claim` prints the worktree
	// command itself, once it has won.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")

	r := p.run("next").ok()
	r.has("kit start 0001", "docs/features/0001-shorten.md")
	if strings.Contains(r.Stdout, "worktree add") {
		t.Errorf("next offered a worktree before the feature was claimed:\n%s", r.Stdout)
	}
}

func TestNextNeverOffersAClaimThatWouldBeRefused(t *testing.T) {
	t.Parallel()
	// The state every new project reaches at step 9 of the protocol, as it was measured:
	// `next` printed `kit claim 0008` and exited 0, and `kit claim 0008` exited 2 saying
	// to land the round first. The board and the command were asking different questions
	// about the same feature; they ask one now.
	a, b := twoAgents(t)
	b.run("claim", "0007").ok() // so the round is the only thing left to offer
	trunk := a.headBranch()
	a.git("checkout", "-q", "-b", "plan/round-two")
	feature(a, "0008", "later", "A later thing", "Draft", "- [ ] Build it")
	a.commitAll("the second round")

	r := a.run("next").ok()
	r.has("blocked", "0008", "plan/round-two")
	r.lacks("kit start 0008")
	// The pairing is the point: what the board withholds is exactly what the command
	// refuses, so neither can send an agent to the other and be wrong.
	a.run("claim", "0008").fails(2).errHas("Land the round first")

	plan := a.plan("next", "--json")
	if len(plan.Blocked) != 1 || !plan.Blocked[0].Unlanded {
		t.Fatalf("blocked = %+v, want 0008 flagged unlanded", plan.Blocked)
	}
	// And the closing line is the reader's own next action. "Stopping here is the right
	// outcome" would leave the round sitting where nobody can build on it.
	if !strings.Contains(plan.Guidance, "open a pull request for human review on main") {
		t.Errorf("guidance says %q, which never names the planning review handoff", plan.Guidance)
	}

	// The quiet half: the same feature, once the round has landed.
	a.git("checkout", "-q", trunk)
	a.git("merge", "-q", "--no-ff", "-m", "land the round", "plan/round-two")
	a.git("push", "origin", "HEAD")

	a.run("next").ok().has("start here", "0008", "kit start 0008")
}

func TestNextWillNotHeadlineAnotherFeatureInAClaimedCheckout(t *testing.T) {
	t.Parallel()
	// Measured before this existed: standing in 0001's checkout, `next` offered 0002
	// under "start here". Followed literally that begins a second feature in the
	// first one's branch, and the two submit as one.
	p := setUp(t)
	feature(p, "0001", "mine", "Mine", "Draft", "- [ ] Finish this")
	feature(p, "0002", "other", "Other", "Draft", "- [ ] A different thing")
	p.commitAll("two features")
	p.git("checkout", "-q", "-b", "0001-mine")

	r := p.run("next").ok()
	if strings.Contains(r.Stdout, "start here") {
		t.Errorf("a claimed checkout was told to start another feature:\n%s", r.Stdout)
	}
	// The free work is still reported — the point is where to do it, not secrecy.
	r.has("0002", "not in this checkout", "kit start 0002")
}

func TestAClaimedCheckoutIsNotOfferedOtherWorkEvenWhenItsOwnIsBlocked(t *testing.T) {
	t.Parallel()
	// The suppression keys off what the checkout claims, not off whether that
	// feature happens to be workable. A blocked feature is still the reason this
	// branch exists, and starting a second one here would still be wrong.
	p := setUp(t)
	feature(p, "0001", "mine", "Mine", "Draft", "- [ ] Finish this")
	feature(p, "0002", "other", "Other", "Draft", "- [ ] A different thing")
	p.commitAll("two features")
	p.git("checkout", "-q", "-b", "0001-mine")
	p.run("block", "0001", "Which way round?").ok()

	r := p.run("next").ok()
	r.has("blocked", "0001")
	if strings.Contains(r.Stdout, "start here") {
		t.Errorf("a claimed checkout with blocked work was told to start another "+
			"feature:\n%s", r.Stdout)
	}
}

func TestNextOnAnUnsetUpProjectSaysHowToFixItAndExitsTwo(t *testing.T) {
	t.Parallel()
	// Nothing was surveyed, so an empty board is not a report that there is nothing to
	// do. It exited 0, which is the same answer a project with every feature submitted
	// gets — and the one a script cannot tell apart from it.
	p := newProject(t)
	p.run("next").fails(2).has("not set up yet", "kit init")
	// --json is the format agents read, and it carries the same conclusion.
	p.run("next", "--json").fails(2).outHas("not set up yet")
}

func TestNextIsNotBlockedByAQuestionInAnotherFeaturesDoc(t *testing.T) {
	t.Parallel()
	// The doc is the id. A question that merely names 0007 in its prose used to block
	// 0007 by a whole-line match, and the cause was invisible in the output.
	p := setUp(t)
	feature(p, "0007", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")
	feature(p, "0012", "codes", "Codes", "Draft", "- [ ] The encoder")
	ask(p, "docs/features/0012-codes.md", "Should this match what we did in 0007?")

	p.run("next").ok().has("start here", "The POST handler")
}

func TestNextBlocksOnAnUndatedQuestion(t *testing.T) {
	t.Parallel()
	// A box nobody dated cannot age into `question-stale`, but it still asks something.
	// Blocking is the safe direction: missing one sends an agent into work a human has
	// not cleared, and `question-informal` makes the line visible.
	p := setUp(t)
	feature(p, "0007", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")
	p.write("docs/features/0007-shorten.md",
		p.read("docs/features/0007-shorten.md")+
			"\n## Open questions\n\n- [ ] is 0007 really the right approach\n")

	p.run("next").ok().has("blocked")
}

func TestNextTellsAnEmptyPlanApartFromAFinishedOne(t *testing.T) {
	t.Parallel()
	// Both leave nothing to do, and the fixes are opposite: one needs the plan
	// written, the other needs submitting.
	never := setUp(t)
	feature(never, "0001", "empty", "Empty", "Draft", "")
	never.run("next").ok().has("no task boxes yet")

	finished := setUp(t)
	feature(finished, "0001", "done", "Done", "Draft", "- [x] All of it")
	finished.run("next").ok().has("every task is ticked")
}

func TestNextSaysWhenTheProtocolIsStale(t *testing.T) {
	t.Parallel()
	// `next` is the command the protocol tells agents to run at the start of every
	// session; `audit` is not. One binary on $PATH serves every project and only
	// `kit init` rewrites AGENTS.md, so a project keeps following whatever rules it was
	// scaffolded with and nothing inside it can tell — the failure 0009 recorded, where a
	// real run went two features behind and "nothing looked wrong from inside".
	stale := setUp(t)
	feature(stale, "0001", "thing", "A thing", "Draft", "- [ ] Do the thing")
	stale.write("AGENTS.md", "# House rules\n\nOurs.\n\n## Work protocol\n\nRules from an older kit.\n")

	r := stale.run("next").ok() // never changes the exit code: the plan is still right
	r.has("AGENTS.md is older than this kit", "kit init")
	if !strings.Contains(r.out(), "start here") {
		t.Errorf("the warning replaced the board instead of preceding it%s", r.context())
	}
	if !stale.plan("next", "--json").ProtocolStale {
		t.Error("--json did not carry protocol_stale; agents read that format, not the prose")
	}

	// And silence when the protocol is current, or the line becomes noise nobody reads.
	current := setUp(t)
	feature(current, "0001", "thing", "A thing", "Draft", "- [ ] Do the thing")
	fresh := current.run("next").ok()
	fresh.lacks("older than this kit")
	if current.plan("next", "--json").ProtocolStale {
		t.Error("a freshly scaffolded project was reported as stale")
	}
}

// hangingGit puts a `git` ahead of the real one on PATH that answers `ls-remote` the way
// a black-holed remote does: it leaves a process holding the stdout pipe it inherited.
// Everything else passes straight through. The sleeper is killed when the test ends, so
// nothing outlives the run.
func hangingGit(t *testing.T) string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("finding git: %v", err)
	}
	dir := t.TempDir()
	pids := filepath.Join(dir, "pids")
	// It exits 0 rather than hanging itself, which is the harder half of the same bug:
	// the deadline has nothing left to kill, so without WaitDelay the read never ends.
	script := fmt.Sprintf(`#!/bin/sh
for a in "$@"; do
	if [ "$a" = "ls-remote" ]; then
		sleep 45 &
		echo $! >> %q
		exit 0
	fi
done
exec %q "$@"
`, pids, realGit)
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatalf("writing the shim: %v", err)
	}
	t.Cleanup(func() {
		raw, err := os.ReadFile(pids)
		if err != nil {
			return
		}
		for _, field := range strings.Fields(string(raw)) {
			if pid, err := strconv.Atoi(field); err == nil {
				if proc, err := os.FindProcess(pid); err == nil {
					_ = proc.Kill()
				}
			}
		}
	})
	return dir
}

func TestTheRemoteQuestionComesBackWhileAGrandchildHoldsThePipe(t *testing.T) {
	t.Parallel()
	// `exec.CommandContext` kills `git` and nothing git started, so `git-remote-https`
	// goes on holding the stdout pipe and `cmd.Output()` blocks on a read that never
	// ends. Measured against a black-holed remote: 75s against the 15s the help states,
	// and 15s once cmd.WaitDelay closed the pipes.
	//
	// The shim is that shape with the network taken out, so this is a second either way
	// rather than a minute — and it is the case where the deadline cannot help at all,
	// because the process it would kill has already exited.
	p := setUp(t)
	feature(p, "0001", "thing", "A thing", "Draft", "- [ ] Build it")
	p.commitAll("the plan")
	p.git("remote", "add", "origin", filepath.Join(t.TempDir(), "origin.git"))

	cmd := exec.Command(kitBinary(t), "next")
	cmd.Dir = p.Root
	cmd.Env = append(gitEnv("TERM=dumb"),
		"PATH="+hangingGit(t)+string(os.PathListSeparator)+os.Getenv("PATH"))
	started := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting kit next: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// Well above the bound and well under the 45s the sleeper holds the pipe, so a
	// failure here is the read never ending rather than a slow machine.
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("kit next exited %v while the remote went unanswered", err)
		}
		t.Logf("the board came back in %s", time.Since(started).Round(time.Millisecond))
	case <-time.After(25 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatal("kit next never came back: the remote question is bounded by a deadline " +
			"that kills git and not what git left holding the pipe")
	}
}

func TestNextRefusesAPathArgument(t *testing.T) {
	t.Parallel()
	// `next` reads the project you are standing in and nothing else. A path is not a
	// board it can render, so it is a broken invocation: exit 2, the reason on stderr,
	// and the shell loop that does what the argument was reaching for. Silently reading
	// the current project instead would answer a question nobody asked.
	p := setUp(t)
	feature(p, "0001", "thing", "A thing", "Draft", "- [ ] Do the thing")

	for _, arg := range []string{p.Root, filepath.Join(p.Root, "gone")} {
		r := p.run("next", arg).fails(2)
		r.errHas("takes no arguments", "cd $d && kit next")
		r.lacks("start here")
	}
}

// shipOnItsBranch builds and ships a feature the way the protocol does — on its claim
// branch, pushed, with the trunk left untouched — and leaves the checkout back on the
// trunk. That is the state a coordinator reads, and the one where the trunk's copy of
// the doc still says what it said when the round landed.
func shipOnItsBranch(p *project, id, slug, title string) {
	p.t.Helper()
	p.run("claim", id).ok()
	w := p.claimWorktree("work/" + id)
	w.write("thing.txt", "the work\n")
	feature(w, id, slug, title, "Submitted", "- [x] Build it")
	w.commitAll("build and submit " + id)
	w.git("push", "-q", "origin", "HEAD:refs/heads/work/"+id)
}

func TestNextNamesASpikeRatherThanRefusingIt(t *testing.T) {
	t.Parallel()
	// Work that started without a doc. The board routes claims and plan rounds; a branch
	// that is neither has nothing to refuse, so the stopping line answers a question the
	// operator at the keyboard did not ask.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	a.git("checkout", "-q", "-b", "spike/try-something")

	r := a.run("next").ok().has("spike/try-something", "not routing it")
	if strings.Contains(r.Stdout, "Stopping here is the right outcome") {
		t.Errorf("the board refused a branch it does not route:\n%s", r.Stdout)
	}
	// The command under the guidance is the one that ends a spike, not the one that
	// starts the route the operator did not take.
	r.has("kit harvest").lacks(`kit new "<what the feature does>"`)

	// The conservative direction: a branch citing an id is a claim however it is spelled,
	// so a mistyped one is reported by orphan-claim rather than becoming a spike.
	a.git("checkout", "-q", "-b", "0009-mistyped")
	a.run("next").ok().lacks("not routing it")
}

func TestGuidanceHandsReviewToTheHuman(t *testing.T) {
	t.Parallel()
	// 0013's rule: never leave an agent without an instruction. A board with nothing
	// free and a feature pushed and unmerged has work on it, so "stopping here is the
	// right outcome" is the one answer that must not be printed.
	a, _ := twoAgents(t)
	shipOnItsBranch(a, "0007", "widget", "A widget")

	r := a.run("next").ok().has("awaiting review", "human reviewer")
	if strings.Contains(r.Stdout, "Stopping here is the right outcome") {
		t.Errorf("next told the agent to stop while a feature waits for review%s", r.context())
	}

	// The same conclusion in the format agents parse, per the reason Guidance ships in
	// both: a board with no instruction is where an agent decides for itself.
	if guidance := a.plan("next", "--json").Guidance; !strings.Contains(guidance, "awaiting review") {
		t.Errorf("--json guidance says %q, which never mentions the review waiting", guidance)
	}
}

func TestNextShowsWhatIsAwaitingReview(t *testing.T) {
	t.Parallel()
	// The blind spot this closes: from the trunk, a feature submitted on its branch and
	// never merged is invisible. The doc here still says Draft — the status that
	// matters is the one on the branch, which is the only place the submit happened.
	a, _ := twoAgents(t)
	shipOnItsBranch(a, "0007", "widget", "A widget")

	a.run("next").ok().has("awaiting review", "work/0007", "waiting")

	plan := a.plan("next", "--json")
	if len(plan.AwaitingReview) != 1 {
		t.Fatalf("--json reports %d awaiting review, want 1: %+v", len(plan.AwaitingReview),
			plan.AwaitingReview)
	}
	if got := plan.AwaitingReview[0]; got.ID != "0007" || got.Branch != "work/0007" {
		t.Errorf("awaiting review entry is %+v, want 0007 on work/0007", got)
	}
	if plan.AwaitingReview[0].Since.IsZero() {
		t.Errorf("the entry says nothing about how long it has waited: %+v",
			plan.AwaitingReview[0])
	}

	// The quiet half: once the work is on the trunk there is nothing to say, whether or
	// not the branch was deleted afterwards.
	a.git("merge", "-q", "--no-ff", "-m", "land 0007", "work/0007")
	a.git("push", "origin", "HEAD")
	if got := a.plan("next", "--json").AwaitingReview; len(got) != 0 {
		t.Errorf("after landing, %d still awaiting review: %+v", len(got), got)
	}
}

func TestASubmittedButUnmergedFeatureAppearsOnceOnTheBoard(t *testing.T) {
	t.Parallel()
	// It was listed under `taken` and again under `awaiting review`, which reads as two
	// features. The second is the true one: nothing is left to build, and the claim is
	// only waiting for somebody to merge it.
	a, _ := twoAgents(t)
	shipOnItsBranch(a, "0007", "widget", "A widget")

	plan := a.plan("next", "--json")
	if !plan.Empty() {
		t.Errorf("a submitted-and-unmerged feature is still on the board: %+v", plan)
	}
	if len(plan.AwaitingReview) != 1 {
		t.Fatalf("awaiting review = %+v, want just 0007", plan.AwaitingReview)
	}

	r := a.run("next").ok().has("awaiting review", "0007")
	if strings.Contains(r.Stdout, "taken") {
		t.Errorf("0007 is reported as taken as well as awaiting review:\n%s", r.Stdout)
	}
	// And one instruction under it rather than two: the guidance says merging is the
	// work, so the command for drafting the next feature must not be printed beside it.
	r.lacks("kit new")
}

func TestTheReviewBoardSaysWhoBuiltIt(t *testing.T) {
	t.Parallel()
	// A coordinator that is itself an agent needs one fact from this board: which
	// entries are its own. Its own work is the work it may not merge, and nothing else
	// on the board answers it.
	//
	// The checkout standing on the claim branch is the whole answer. Comparing the
	// branch's author with this checkout's git identity was the earlier one and
	// distinguished nobody: measured on this repo, four agents shared the operator's.
	a, b := twoAgents(t)
	shipOnItsBranch(a, "0007", "widget", "A widget")

	// The builder's own checkout, standing where the work was done.
	a = a.claimWorktree("work/0007")
	if got := a.plan("next", "--json").AwaitingReview; len(got) != 1 || !got[0].Mine {
		t.Errorf("the builder's own board does not say it built 0007: %+v", got)
	}
	a.run("next").ok().has("you built this")

	// A different checkout is free to merge it — and saying "you built this" there
	// would be the kit inventing a reason to stop.
	b.git("fetch", "-q", "origin")
	got := b.plan("next", "--json").AwaitingReview
	if len(got) != 1 {
		t.Fatalf("the coordinator's board reports %d awaiting review, want 1: %+v", len(got), got)
	}
	if got[0].Mine {
		t.Errorf("0007 is marked as the coordinator's own work: %+v", got[0])
	}
	b.run("next").ok().lacks("you built this")
}

func TestTheBoardCountsBoxesFromTheClaimBranch(t *testing.T) {
	t.Parallel()
	// The ticks live on the claim branch until it merges, so counting the checkout's copy
	// reported 0/3 for a feature somebody was two-thirds through — indistinguishable from
	// an agent that had done nothing.
	p := setUp(t)
	feature(p, "0001", "widget", "Widget", "Draft", "- [ ] One\n- [ ] Two\n- [ ] Three")
	p.commitAll("the plan")
	trunk := p.headBranch()

	p.git("checkout", "-q", "-b", "0001-widget")
	feature(p, "0001", "widget", "Widget", "Draft", "- [x] One\n- [x] Two\n- [ ] Three")
	p.write("widget.go", "package widget\n")
	p.commitAll("0001: two boxes")
	p.git("checkout", "-q", trunk)

	taken := p.plan("next", "--json").Taken
	if len(taken) != 1 {
		t.Fatalf("want 0001 taken, got %+v", taken)
	}
	if taken[0].Done != 2 || taken[0].Total != 3 {
		t.Errorf("the board says %d/%d; the branch has two of three ticked",
			taken[0].Done, taken[0].Total)
	}
	p.run("next").ok().has("2/3")
}

func TestTheBoardKeepsItsOwnCountWhenTheBranchCannotBeRead(t *testing.T) {
	t.Parallel()
	// A claim branch that predates the doc — or one whose objects were never fetched —
	// carries no copy to count. Reporting nothing there would be worse than reporting the
	// checkout's number, which is at least the plan as the trunk knows it.
	p := setUp(t)
	p.write("seed.txt", "before any feature\n")
	p.commitAll("seed")
	p.git("branch", "0001-widget")

	feature(p, "0001", "widget", "Widget", "Draft", "- [x] One\n- [ ] Two\n- [ ] Three")
	p.commitAll("the plan")

	taken := p.plan("next", "--json").Taken
	if len(taken) != 1 {
		t.Fatalf("want 0001 taken, got %+v", taken)
	}
	if taken[0].Done != 1 || taken[0].Total != 3 {
		t.Errorf("the board says %d/%d; with an unreadable branch it keeps the checkout's 1/3",
			taken[0].Done, taken[0].Total)
	}
}

func TestATakenEntrySaysWhenItsBranchLastMoved(t *testing.T) {
	t.Parallel()
	// A count alone cannot tell an agent working steadily from one that stopped an hour
	// in. `stalled` only speaks at three days, and by then the answer has cost two.
	p := setUp(t)
	feature(p, "0001", "widget", "Widget", "Draft", "- [ ] One\n- [ ] Two")
	p.commitAll("the plan")
	trunk := p.headBranch()
	p.git("checkout", "-q", "-b", "0001-widget")
	p.write("widget.go", "package widget\n")
	p.commitAllAt("0001: started", daysAgo(2))
	p.git("checkout", "-q", trunk)

	taken := p.plan("next", "--json").Taken
	if len(taken) != 1 || taken[0].Moved == nil {
		t.Fatalf("--json carries no moved time for a taken feature: %+v", taken)
	}
	if days := int(time.Since(*taken[0].Moved).Hours() / 24); days != 2 {
		t.Errorf("moved reads %d days old, want 2", days)
	}
	p.run("next").ok().has("0001-widget, 2 days ago")
}

func TestTheBoardCountsTheBoxesTheRemoteHas(t *testing.T) {
	t.Parallel()
	// One feature, two checkouts, one instant. The ticks live on the claim branch, and
	// each checkout holds a copy of that branch which can be behind: b fetched when the
	// claim was made and a has been working since. Counting the copy printed 0/1 in b and
	// 1/2 in a for the same feature, and 0/1 is the picture of an agent that has done
	// nothing.
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	b.git("fetch", "-q", "origin")

	w := a.claimWorktree("work/0007")
	feature(w, "0007", "widget", "A widget", "Draft", "- [x] Build it\n- [ ] Submit it")
	w.commitAll("0007: one box")
	w.git("push", "-q", "origin", "work/0007")

	stale := b.plan("next", "--json").Taken
	if len(stale) != 1 {
		t.Fatalf("want 0007 taken in the other checkout, got %+v", stale)
	}
	if stale[0].Total != 0 {
		t.Errorf("a checkout that has never fetched the claim reports %d/%d for it",
			stale[0].Done, stale[0].Total)
	}

	// And with the objects in hand it is the remote's branch that is counted.
	b.git("fetch", "-q", "origin")
	if taken := b.plan("next", "--json").Taken; len(taken) != 1 ||
		taken[0].Done != 1 || taken[0].Total != 2 {
		t.Errorf("the board says %+v; the branch the remote has is one of two ticked", taken)
	}
}

func TestClaimProgressUsesLocalHeadOnlyWhenItContainsRemote(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"ahead", "equal", "behind", "diverged"} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			a, _ := twoAgents(t)
			a.run("claim", "0007").ok()
			w := a.claimWorktree("work/0007")
			w.git("tag", "claim-base")
			feature(w, "0007", "widget", "A widget", "Draft", "- [x] Build it\n- [ ] Submit it")
			w.commitAllAt("0007: published progress", daysAgo(2))
			w.git("push", "-q", "origin", "work/0007")
			wantDone, wantDays := 1, 2
			switch state {
			case "behind":
				w.git("reset", "--hard", "claim-base")
			case "diverged":
				w.git("reset", "--hard", "claim-base")
				fallthrough
			case "ahead":
				feature(w, "0007", "widget", "A widget", "Draft", "- [x] Build it\n- [x] Submit it")
				w.commitAllAt("0007: local progress", daysAgo(1))
				if state == "ahead" {
					wantDone, wantDays = 2, 1
				}
			}
			// Dirty ticks in the held checkout are never another checkout's progress.
			feature(w, "0007", "widget", "A widget", "Draft", "- [ ] Build it")
			taken := a.plan("next", "--json").Taken
			if len(taken) != 1 || taken[0].Done != wantDone || taken[0].Total != 2 || taken[0].Moved == nil {
				t.Fatalf("%s: progress = %+v", state, taken)
			}
			if days := int(time.Since(*taken[0].Moved).Hours() / 24); days != wantDays {
				t.Errorf("moved = %d days, want %d", days, wantDays)
			}
		})
	}
}

func TestLocalSubmissionDoesNotReachTheReviewBoard(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	w := a.claimWorktree("work/0007")
	feature(w, "0007", "widget", "A widget", "Draft", "- [x] Build it")
	w.commitAll("0007: built")
	w.run("submit", "0007").ok()
	plan := a.plan("next", "--json")
	if len(plan.AwaitingReview) != 0 || len(plan.Taken) != 1 || plan.Taken[0].Done != 1 {
		t.Fatalf("unpushed submission = %+v", plan)
	}
}

func TestTakenEntryNamesOnlyThisClonesHoldingWorktree(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()
	w := a.claimWorktree("work/0007")
	// An external custom path must survive porcelain quoting and stay relative.
	path := filepath.Join(t.TempDir(), "held \"checkout\"\nwith spaces")
	a.git("worktree", "move", w.Root, path)
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(a.Root, path)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(rel)
	plan := a.plan("next", "--json")
	if len(plan.Taken) != 1 || plan.Taken[0].Worktree != want || plan.Taken[0].Mine || len(plan.Yours) != 0 {
		t.Fatalf("held claim = %+v", plan)
	}
	a.run("next").ok().has("checked out here: " + want)
	if !strings.Contains(plan.Guidance, "1 claim is checked out in this clone (0007)") ||
		!strings.Contains(plan.Guidance, "kit release <id>") ||
		strings.Contains(plan.Guidance, "Stopping here") {
		t.Fatalf("local claim guidance = %q", plan.Guidance)
	}
	if plan.Action == nil || plan.Action.Checkout != a.Root || len(plan.Action.Command) != 0 {
		t.Fatalf("taken claim routed the reader into another checkout: %+v", plan.Action)
	}
	remote := b.plan("next", "--json")
	if len(remote.Taken) != 1 || remote.Taken[0].Worktree != "" || strings.Contains(remote.Guidance, "checked out in this clone") {
		t.Fatalf("another clone's checkout leaked onto board: %+v", remote)
	}
	// A local branch alone does not establish that a worktree holds it.
	a.git("worktree", "remove", path)
	if taken := a.plan("next", "--json").Taken; len(taken) != 1 || taken[0].Worktree != "" {
		t.Fatalf("unoccupied claim = %+v", taken)
	}
}

func TestGuidanceCountsHeldClaimsWithoutOfferingTheirWork(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	feature(p, "0001", "one", "One", "Draft", "- [ ] Build one")
	feature(p, "0002", "two", "Two", "Draft", "- [ ] Build two")
	p.commitAll("plans")
	for _, branch := range []string{"0001-one", "0002-two"} {
		p.git("worktree", "add", "-b", branch, filepath.Join(t.TempDir(), branch))
	}
	plan := p.plan("next", "--json")
	if len(plan.Taken) != 2 || len(plan.Yours) != 0 || len(plan.Ready) != 0 ||
		!strings.Contains(plan.Guidance, "2 claims are checked out in this clone (0001, 0002)") ||
		!strings.Contains(plan.Guidance, "kit release <id>") {
		t.Fatalf("held claims = %+v", plan)
	}
	text := strings.Join(strings.Fields(p.run("next").ok().Stdout), " ")
	if !strings.Contains(text, "2 claims are checked out in this clone") || !strings.Contains(text, "kit release <id>") {
		t.Fatalf("text guidance = %s", text)
	}
}

func TestNextNamesTheLiveClaimAFreeFeatureWouldMeet(t *testing.T) {
	t.Parallel()
	// The two sides are different kinds of fact: 0002 names a file in prose, and 0001's
	// branch has actually changed it. The report compares one against the other.
	p := setUp(t)
	feature(p, "0001", "taken", "Taken", "Draft", "- [ ] Change the handler")
	feature(p, "0002", "free", "Free", "Draft", "- [ ] Also change `handler.go`")
	p.write("handler.go", "package main\n")
	p.commitAll("two plans and the file")
	p.run("claim", "0001").ok()

	// The claim only collides once it has changed something. A branch carrying nothing
	// is a claim with no work in it yet.
	p.run("next").ok().lacks("meets 0001")

	w := p.claimWorktree("work/0001")
	w.write("handler.go", "package main // taken\n")
	w.commitAll("work on the handler")

	p.run("next").ok().has("meets 0001", "handler.go")
}

func TestNextIsSilentWhenAFreeFeatureNamesNoFile(t *testing.T) {
	t.Parallel()
	// The operator's decision on 0083: silence over nagging. Most docs name no file, and
	// a finding on each of them is the noise a check must not make.
	p := setUp(t)
	feature(p, "0001", "taken", "Taken", "Draft", "- [ ] Change the handler")
	feature(p, "0002", "free", "Free", "Draft", "- [ ] Change the handler somehow")
	p.write("handler.go", "package main\n")
	p.commitAll("two plans and the file")
	p.run("claim", "0001").ok()
	w := p.claimWorktree("work/0001")
	w.write("handler.go", "package main // taken\n")
	w.commitAll("work on the handler")

	r := p.run("next").ok()
	r.lacks("meets")
	// And it is still offered as work — an overlap never refuses.
	r.has("0002")
}

func TestNextSeesAClaimThatExistsOnlyOnTheRemote(t *testing.T) {
	t.Parallel()
	// The case the report exists for: another agent's claim, which this clone has only
	// as refs/remotes/*. Diffing the branch name alone resolves to nothing here, so the
	// overlap went unreported exactly when somebody else was already in the file.
	a, b := twoAgents(t)
	feature(a, "0008", "second", "Second", "Draft", "- [ ] Also change `shared.go`")
	a.write("shared.go", "package main\n")
	a.commitAll("a second plan and the file")
	a.git("push", "-q", "origin", "HEAD:refs/heads/main")
	a.run("claim", "0007").ok()

	aw := a.claimWorktree("work/0007")
	aw.write("shared.go", "package main // a is here\n")
	aw.commitAll("change the shared file")
	aw.git("push", "-q", "origin", "HEAD:refs/heads/work/0007")

	b.git("fetch", "-q", "origin")
	b.git("merge", "-q", "--ff-only", "origin/main")
	if b.gitOut(nil, "branch", "--list", "work/0007") != "" {
		t.Fatal("b has a local copy of the claim, so this proves nothing")
	}

	b.run("next").ok().has("meets 0007", "shared.go")
}
