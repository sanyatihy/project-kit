---
id: 0071
title: Work that starts without a plan still reaches the board
status: Draft
created: 2026-08-30
shipped:
needs: 0070
---

## Problem

The protocol has one route from an idea to a shipped feature: a drafting round lands on
the trunk, then a builder claims from it. An operator who explores first has no route at
all. `claim` refuses until the round lands and `ship` needs a claim, so the session ends
as a plan branch carrying docs whose boxes are ticked and whose status is still `Draft` —
a state the board reports as wrong. BACKLOG.md has recorded this twice and set its trigger
at a third; the third was a whole project, documented in five docs after it was built,
whose board now blocks every feature on `every task is ticked — ship it or add the next
task`. `kit next` with nothing free also prints `Stopping here is the right outcome`,
which is true for an unattended agent and a wall for the operator at the keyboard.

The vision already takes this side. It measures the kit on a record that is "a byproduct
of the work rather than a stage before it", and calls a design essay written before the
work expensive and frequently wrong by the end. The protocol requires exactly that essay.

## Non-goals

- **Replacing the plan-first route.** Agents that never speak to each other need the doc
  before the work: `needs:` is what routes them, and it cannot be inferred after the fact.
  This is a second route, not a swap.
- **Writing `## Design` or `## Problem` from a diff.** A design recovered from code holds
  nothing the code does not, and reads as though it were decided rather than guessed. What
  earns a doc is the part the code cannot say.
- **Capturing anything mid-spike.** A note to file while exploring rebuilds the cost at
  the point it was removed from. The interview is at the end or it is not worth having.
- **A status for exploring.** There are three, and work already has none — the branch is
  the state, as it is for a claim.
- **A bypass of review.** Harvest writes a `Draft`. Rule 20 stands: the spike still ends
  in a pull request that someone else merges.

## Design

**A branch that is neither a claim nor a plan is a spike, and the board names it.** Kit
already tolerates these — `chore/` and `backlog-` branches merge through it and no check
looks at them. `kit next` on one says what it is and that the board is not routing it,
instead of printing a refusal aimed at an agent. No verb starts a spike and no id is
reserved, so starting costs nothing and a spike that dies leaks no number. A branch whose
name begins with an id stays a claim; `orphan-claim` already reports one naming a doc that
does not exist, so a mistyped claim does not silently become a spike.

**`kit harvest` does not read history. It creates it.** A locked-in session leaves a
working tree, not a commit log: the project that produced this feature had 2,090 lines, 82
tests and no commits at all. So harvest takes the id, writes the doc, and commits the tree
together with it. That single commit is where the work becomes reviewable, and it is what
makes `ship`'s clean-tree gate reachable from a spike at all.

**The operator names the boxes and the kit ticks them.** No order of work can be recovered
from a tree, and a commit message is not an outcome — the mechanical half is the list of
files harvest is about to commit, printed as raw material. Every box it writes is ticked,
because a doc written after the work has nothing unticked to offer. Rule 13 is answered
rather than broken: one commit carries the boxes and the code that makes them true, which
is what a reviewer needs in one place.

**The kit asks rather than invents.** `## Problem`, `## Design` and `## Log` are written as
the questions only the operator can answer — what this was for, what was chosen over what,
what was cut. `unfilled-template` is already HIGH on an empty `## Problem` or `## Design`,
so the doc cannot pass `audit --strict` until they are answered and nothing has to be
enforced twice. The kit writes no sentence about why.

**A spike that did not survive harvests to a BACKLOG line.** `kit harvest --nothing
"<what it showed>"` appends the entry and takes no id. Trying something and discarding it
stays as cheap as it has to be, and the one durable output is kept.

**Harvest promotes the parked entries it names.** The end of a spike is where the operator
knows what is worth doing next and the fleet has nothing to claim, so `kit harvest` offers
the BACKLOG entries as drafts. This is the handoff: attended work exits by giving
unattended work its round.

**`needs: 0070` because both rewrite `templates/AGENTS-section.md`.** 0070 drops two rules
and renumbers the list; this adds the second route to it. 0070's non-goals park this
question by name, so it is also the doc that says this comes after.

## Tasks

- [x] `kit next` recognises a branch that is neither a claim nor a plan, names it, and does
      not print a refusal — with a conformance case that the output contains no
      `Stopping here is the right outcome`
- [x] `kit harvest` takes the next id, writes a doc whose `## Tasks` are ticked and whose
      `## Problem`, `## Design` and `## Log` are the questions to answer, and commits the
      working tree with it — with a conformance case that runs on a repo with no commits
- [x] `kit harvest --nothing "<what it showed>"` appends a BACKLOG entry and reserves no id
- [x] `kit harvest` offers the BACKLOG entries as drafts, so the round the fleet claims
      comes out of the spike that ended
- [x] the protocol gains the second route, `kit help` carries it, and this project's
      `AGENTS.md` is regenerated in the same commit as the template
- [x] `docs/TECH.md` says a spike is a branch the board does not route, and that an id is
      taken at harvest
- [x] the command `kit next` prints under the spike guidance is `kit harvest`, not
      `kit new`

## Open questions

- **Is `harvest` a verb of its own, or a flag on `kit new`?** Leaning: its own. `new`
  writes a plan for work not done and `harvest` writes a record of work done; one flag
  would make the difference invisible in the protocol, and the two fill opposite halves
  of the same template.
- **What does harvest do with a spike that ran for days?** The interview assumes the
  session is still in the operator's head. Leaning: accept that it degrades and say so,
  because every fix for it is a thing to record while exploring, which is the cost this
  removes.

## Log

- 2026-08-30 — promoted from BACKLOG.md's "One agent that plans and builds in the same
  session has no path" on its own trigger, a third occurrence. The entry offered two
  repairs: name the case in the protocol, or teach `ship` a path with no claim branch.
  Neither is taken. Both treat the doc as owed before the work, and the third occurrence
  was a project where the code came first and the docs were better for it.
- 2026-08-30 — harvest no longer reads commits. It was drafted to tick boxes from a branch's
  log, and a locked-in session has no commits to read: the project that prompted this had
  none for its whole build. Harvest makes the commit instead, which also reaches `ship`'s
  clean-tree gate. The boxes come from the operator, since a tree carries no order of work.
- 2026-08-31 — harvest refuses on the trunk, which no box asked for. It commits the whole
  working tree, and on the trunk that is the one commit nobody could review. The refusal is
  skipped where there is no history: `Trunk` falls back to whatever HEAD names, and a
  repository with no commits is the state this command exists for.
- 2026-08-31 — "offers as drafts" is read as naming them with the command that drafts each,
  not as writing the drafts. Which parked entry is worth building is a product decision,
  and the rule against making one on the human's behalf outranks the convenience.
- 2026-08-31 — a seventh box, found by running the route by hand rather than by a test: the
  board printed `kit new` under the spike guidance, which names the route the operator did
  not take. Box 1 was satisfied — no refusal was printed — and the command beside it was
  still wrong.
