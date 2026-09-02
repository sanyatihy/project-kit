---
id: 0071
title: Work that starts without a plan still reaches the board
status: Shipped
created: 2026-08-30
shipped: 2026-09-01
needs: 0070
---

## Problem

The protocol has one route from idea to shipped feature: a round lands on the trunk, then a
builder claims from it. An operator who explores first has none — `claim` refuses until the
round lands and `ship` needs a claim, so the session ends as a plan branch of ticked boxes
still marked `Draft`, which the board reports as wrong. `kit next` with nothing free prints
`Stopping here is the right outcome`, true for an unattended agent and a wall for the
operator at the keyboard. BACKLOG.md recorded this twice and set its trigger at a third; the
third was a whole project, documented in five docs after it was built.

## Non-goals

- Replacing the plan-first route. `needs:` routes agents that never speak to each other and
  cannot be inferred after the fact. This is a second route.
- Writing `## Design` or `## Problem` from a diff. A design recovered from code holds
  nothing the code does not.
- Capturing anything mid-spike. The interview is at the end or it is not worth having.
- A status for exploring. The branch is the state, as it is for a claim.
- A bypass of review. Harvest writes a `Draft`, and the spike still ends in a pull request.

## Design

A branch that is neither a claim nor a plan round is a spike, and the board names it rather
than refusing it. No verb starts one and no id is reserved, so starting costs nothing and a
spike that dies leaks no number. Any four-digit id in the name keeps it a claim, which is
the conservative direction: a mistyped claim is `orphan-claim`'s, not a spike.

`kit harvest` does not read history, it creates it. A locked-in session leaves a working
tree and no commit log — the project that prompted this had 2,090 lines and none — so
harvest takes the id, writes the doc, and commits the tree with it. That single commit is
where the work becomes reviewable, and it is what leaves the clean tree `ship` requires.

The operator names the boxes and the kit ticks them. No order of work can be recovered from
a tree and a commit message is not an outcome; the mechanical half is the file list, which
the refusal prints when `--box` is missing. Every box arrives ticked, because a doc written
after the work has nothing unticked to offer.

`## Problem`, `## Design` and `## Log` arrive as questions. `unfilled-template` is already
HIGH on an empty one, so `audit --strict` refuses the doc until they are answered and
nothing is enforced twice. The kit writes no sentence about why.

`kit harvest --nothing "<what it showed>"` appends one BACKLOG entry, commits that file
alone and takes no id, so discarding work stays as cheap as it is.

Harvest names the parked entries at the end, with the command that drafts each. That is the
handoff: attended work exits by giving unattended work its round.

`needs: 0070` because both rewrite `templates/AGENTS-section.md`.

## Tasks

- [x] `kit next` names a branch that is neither a claim nor a plan, and prints no refusal
- [x] `kit harvest` takes an id, writes the doc with the boxes ticked and the questions
      open, and commits the working tree with it
- [x] `kit harvest --nothing "<what it showed>"` appends a BACKLOG entry and takes no id
- [x] Harvest names the parked entries with the command that drafts each
- [x] The protocol gains the second route and `kit help` carries it
- [x] `docs/TECH.md` says a spike is a branch the board does not route
- [x] The command under the spike guidance is `kit harvest`, not `kit new`

## Open questions

- Is `harvest` a verb of its own, or a flag on `kit new`? Leaning: its own. `new` plans work
  not done and `harvest` records work done; one flag hides that in the protocol.
- What does harvest do with a spike that ran for days? Leaning: accept that it degrades.
  Every fix for it is a thing to record while exploring, which is the cost this removes.

## Log

- 2026-08-30 — promoted from BACKLOG.md on its third occurrence. Both repairs the entry
  offered treat the doc as owed before the work; neither is taken.
- 2026-08-30 — harvest makes the commit instead of reading one. It was drafted to tick boxes
  from a branch's log, and a locked-in session has none.
- 2026-08-31 — harvest refuses on the trunk, where committing the whole tree is a change
  nobody could review. Skipped without history, since a repo with no commits is the case
  this is for.
- 2026-08-31 — "offers as drafts" means naming them with the command that drafts each.
  Which entry is worth building is a product decision.
- 2026-08-31 — a seventh box: the board printed `kit new` under the spike guidance, naming
  the route the operator did not take. Found by running it, not by a test.
