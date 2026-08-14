package main

// Conformance: `kit next`. This is the command an agent runs first in every
// session, so what it hands back — and what it refuses to hand back — is the whole
// contract.

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// feature writes a minimal but valid feature doc. The template's full shape is
// covered by the `new` suite; what matters here is status, ID, and task list.
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

// plan runs `kit next --json` and returns this project's plan, for the assertions
// that are about the structure rather than the printed page.
func (p *project) plan(args ...string) kit.Plan {
	p.t.Helper()
	r := p.run(args...).ok()
	var payload map[string]kit.Plan
	if err := json.Unmarshal([]byte(r.Stdout), &payload); err != nil {
		p.t.Fatalf("next --json emitted invalid JSON: %v\n%s", err, r.Stdout)
	}
	for _, plan := range payload {
		return plan
	}
	p.t.Fatalf("next --json reported no project at all:\n%s", r.Stdout)
	return kit.Plan{}
}

func TestNextHandsBackTheTopmostUntickedBox(t *testing.T) {
	// List order is the order of work, which is what lets ordering need no syntax.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted",
		"- [x] Schema\n- [ ] The POST handler\n- [ ] The redirect")

	r := p.run("next").ok()
	r.has("start here", "0001", "The POST handler")
	r.lacks("The redirect")
}

func TestNextOffersInProgressBeforeAccepted(t *testing.T) {
	// Finishing beats starting, so a half-done feature outranks a fresh one even
	// though its ID is higher.
	p := setUp(t)
	feature(p, "0001", "first", "First", "Accepted", "- [ ] Do the first thing")
	feature(p, "0002", "second", "Second", "In progress", "- [ ] Do the second thing")

	out := p.run("next").ok().Stdout
	inProgress := strings.Index(out, "0002")
	accepted := strings.Index(out, "0001")
	if inProgress == -1 || accepted == -1 {
		t.Fatalf("both features should be listed:\n%s", out)
	}
	if inProgress > accepted {
		t.Errorf("Accepted 0001 was offered before In progress 0002:\n%s", out)
	}
}

func TestNextTreatsABranchAsAClaim(t *testing.T) {
	// The claim is a branch name and nothing else — no lock file, no state to sync.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
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
	// Measured before this existed: standing in the checkout that claims 0001,
	// `next` reported 0001 as taken — by you — which is the tool stating a fact
	// about the reader back to them as somebody else's.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
	p.commitAll("add the feature doc")
	p.git("checkout", "-q", "-b", "0001-shorten")

	r := p.run("next").ok()
	r.has("yours", "0001", "The POST handler")
	if strings.Contains(r.Stdout, "taken") {
		t.Errorf("the checkout's own claim was reported as taken:\n%s", r.Stdout)
	}
}

func TestNextJSONSeparatesYoursFromTaken(t *testing.T) {
	// --json is the interface agents actually read, so the distinction has to exist
	// there and not only in the prose above it.
	p := setUp(t)
	feature(p, "0001", "mine", "Mine", "Accepted", "- [ ] Do it")
	feature(p, "0002", "theirs", "Theirs", "Accepted", "- [ ] Do that")
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
	// With one feature the next box is the whole state. With several the question is
	// which are moving, and only a count answers it. In --json too, because that is
	// what anything watching the plan from outside has to read.
	p := setUp(t)
	feature(p, "0001", "part", "Partly done", "Accepted",
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
	// The whole point of the field: what is ready is the answer to "can these be
	// worked at once", so it has to be right in both directions — refusing work that
	// cannot start, and releasing it the moment the dependency ships.
	p := setUp(t)
	feature(p, "0001", "base", "Base", "Accepted", "- [ ] Do it")
	p.write("docs/features/0002-store.md",
		doc("0002", "Store", "Accepted", "needs: 0001", "- [ ] Do it"))
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
	p.run("next").ok().has("needs 0001 to ship first")

	// Shipped is the bar, not merged.
	p.write("docs/features/0001-base.md",
		doc("0001", "Base", "Shipped", "shipped: 2026-01-02", "- [x] Do it"))
	p.commitAll("ship 0001")

	plan = p.plan("next", "--json")
	if len(plan.Blocked) != 0 {
		t.Errorf("blocked = %+v, want nothing once the dependency shipped", plan.Blocked)
	}
	if len(plan.Ready) != 1 || plan.Ready[0].ID != "0002" {
		t.Errorf("ready = %+v, want 0002 released", plan.Ready)
	}
}

func TestNextAlwaysEndsWithAnInstruction(t *testing.T) {
	// The invariant, not any particular sentence. A state the tool can reach and
	// does not close is a state something else decides — which is how an agent
	// facing a board of taken and blocked features started working on one of them.
	tell := []string{"kit claim", "kit new", "nothing is free", "waiting on a human"}
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
	t.Run("only drafts", func(t *testing.T) {
		p := setUp(t)
		feature(p, "0001", "idea", "Idea", "Draft", "- [ ] Do it")
		out := p.run("next").ok().Stdout
		if !ends(out) {
			t.Errorf("no instruction:\n%s", out)
		}
		// Accepting is a human's call, so the tool must not tell an agent to do it.
		if strings.Contains(out, "set status: Accepted") {
			t.Errorf("next told an agent to take the human's gate:\n%s", out)
		}
	})
	t.Run("something ready", func(t *testing.T) {
		p := setUp(t)
		feature(p, "0001", "go", "Go", "Accepted", "- [ ] Do it")
		if out := p.run("next").ok().Stdout; !ends(out) {
			t.Errorf("no instruction:\n%s", out)
		}
	})
	t.Run("everything taken or blocked", func(t *testing.T) {
		// The state that actually failed: one feature claimed by somebody else, the
		// rest waiting on it. Nothing for this agent, and it must be told so.
		p := setUp(t)
		feature(p, "0001", "taken", "Taken", "Accepted", "- [ ] Do it")
		p.write("docs/features/0002-waiting.md",
			doc("0002", "Waiting", "Accepted", "needs: 0001", "- [ ] Do it"))
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
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
	p.write("docs/INBOX.md",
		"# Inbox\n\n- [ ] 2026-01-01 · 0001 · BLOCKING · Should codes expire?\n")

	r := p.run("next").ok()
	r.has("blocked", "waiting on", "Should codes expire?")
	if strings.Contains(r.Stdout, "start here") {
		t.Errorf("a blocked feature was offered as work:\n%s", r.Stdout)
	}
}

func TestNextIsNotBlockedByAnFYIQuestion(t *testing.T) {
	// FYI is recorded for a human without stopping the work, or every note would
	// halt a feature.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
	p.write("docs/INBOX.md",
		"# Inbox\n\n- [ ] 2026-01-01 · 0001 · FYI · Using base62 for slugs.\n")

	p.run("next").ok().has("start here", "The POST handler")
}

func TestNextBlocksAFeatureWithEveryBoxTicked(t *testing.T) {
	// Nothing left to do is not the same as ready to work: it needs shipping or a
	// next task, and silently offering nothing would hide that.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [x] Done\n- [x] Also done")

	p.run("next").ok().has("blocked", "every task is ticked")
}

func TestNextExplainsDraftsRatherThanSayingNothingIsOpen(t *testing.T) {
	// `next` never offers a Draft — that is where a human settles the design — but
	// silence would look like there is no work at all.
	//
	// This used to assert the output said "Accepted", which was the tool telling an
	// agent to take the human's gate. Naming who the drafts are waiting on is the
	// same information without the instruction to overstep it.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")

	r := p.run("next").ok()
	r.has("drafts", "0001", "waiting on a human")
	r.lacks("start here")
	r.lacks("set status: Accepted")
}

func TestNextWarnsOnlyAboveTheWIPLimit(t *testing.T) {
	const warning = "in progress at once"

	atLimit := setUp(t)
	feature(atLimit, "0001", "a", "A", "In progress", "- [ ] Task a")
	feature(atLimit, "0002", "b", "B", "In progress", "- [ ] Task b")
	atLimit.run("next").ok().lacks(warning)

	over := setUp(t)
	feature(over, "0001", "a", "A", "In progress", "- [ ] Task a")
	feature(over, "0002", "b", "B", "In progress", "- [ ] Task b")
	feature(over, "0003", "c", "C", "In progress", "- [ ] Task c")
	over.run("next").ok().has(warning)
}

func TestNextPrintsTheCommandThatStartsTheWork(t *testing.T) {
	// The output is meant to be actionable without reading the docs first.
	//
	// The command is the claim, not the worktree. It used to print `git worktree add
	// -b`, which builds somewhere to work before knowing the work is yours — and
	// skips the only step that settles who gets it. `kit claim` prints the worktree
	// command itself, once it has won.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")

	r := p.run("next").ok()
	r.has("kit claim 0001", "docs/features/0001-shorten.md")
	if strings.Contains(r.Stdout, "worktree add") {
		t.Errorf("next offered a worktree before the feature was claimed:\n%s", r.Stdout)
	}
}

func TestNextWillNotHeadlineAnotherFeatureInAClaimedCheckout(t *testing.T) {
	// Measured before this existed: standing in 0001's checkout, `next` offered 0002
	// under "start here". Followed literally that begins a second feature in the
	// first one's branch, and the two ship as one.
	p := setUp(t)
	feature(p, "0001", "mine", "Mine", "In progress", "- [ ] Finish this")
	feature(p, "0002", "other", "Other", "Accepted", "- [ ] A different thing")
	p.commitAll("two features")
	p.git("checkout", "-q", "-b", "0001-mine")

	r := p.run("next").ok()
	if strings.Contains(r.Stdout, "start here") {
		t.Errorf("a claimed checkout was told to start another feature:\n%s", r.Stdout)
	}
	// The free work is still reported — the point is where to do it, not secrecy.
	r.has("0002", "not in this checkout", "kit claim 0002")
}

func TestAClaimedCheckoutIsNotOfferedOtherWorkEvenWhenItsOwnIsBlocked(t *testing.T) {
	// The suppression keys off what the checkout claims, not off whether that
	// feature happens to be workable. A blocked feature is still the reason this
	// branch exists, and starting a second one here would still be wrong.
	p := setUp(t)
	feature(p, "0001", "mine", "Mine", "In progress", "- [ ] Finish this")
	feature(p, "0002", "other", "Other", "Accepted", "- [ ] A different thing")
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

func TestNextOnAnUnsetUpProjectSaysHowToFixIt(t *testing.T) {
	p := newProject(t)
	p.run("next").ok().has("not set up yet", "kit init")
}

func TestNextIsNotBlockedByAQuestionAboutAnotherFeature(t *testing.T) {
	// The ID lives in a field of the line. Matching the whole line blocks any
	// feature a question happens to name, and the cause is invisible in the output.
	p := setUp(t)
	feature(p, "0007", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
	p.write("docs/INBOX.md",
		"# Inbox\n\n- [ ] 2026-08-11 · 0012 · BLOCKING · Should this match what we did in 0007?\n")

	p.run("next").ok().has("start here", "The POST handler")
}

func TestNextStillBlocksOnAMalformedLineThatNamesTheFeature(t *testing.T) {
	// A line outside the format falls back to scanning the whole thing, which
	// over-blocks. That is the safe direction: missing a blocker sends an agent
	// into work a human has not cleared. `inbox-unparsed` makes the line visible.
	p := setUp(t)
	feature(p, "0007", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
	p.write("docs/INBOX.md", "# Inbox\n\n- [ ] is 0007 really the right approach\n")

	p.run("next").ok().has("blocked")
	p.run("audit").ok().has("inbox-unparsed")
}

func TestNextTellsAnEmptyPlanApartFromAFinishedOne(t *testing.T) {
	// Both leave nothing to do, and the fixes are opposite: one needs the plan
	// written, the other needs shipping.
	never := setUp(t)
	feature(never, "0001", "empty", "Empty", "Accepted", "")
	never.run("next").ok().has("no task boxes yet")

	finished := setUp(t)
	feature(finished, "0001", "done", "Done", "Accepted", "- [x] All of it")
	finished.run("next").ok().has("every task is ticked")
}
