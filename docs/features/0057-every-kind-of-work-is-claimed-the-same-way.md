---
id: 0057
title: Every kind of work is claimed the same way
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs: 0058
---

## Problem

`kit claim` is the only atomic operation the kit has, and it works on one kind of work.
The protocol sends everything else — a fix to something shipped, a chore, a data update —
to "a branch whose name contains no four-digit id", which nothing pushes, nothing records
and nothing decides. Two agents given the same bug both start it, both branch, and neither
finds out.

The gap is not hypothetical. Every merge-conflict resolution in the round that shipped
0050 through 0056 went on an unclaimed branch, named by whoever got there first. Nothing
would have stopped a second agent doing the same work beside it.

Two smaller holes in the same command have the same shape. `ClaimFeature` never asks
whether this checkout already holds a claim, so `kit claim 0008` from inside 0007's
worktree exits 0 — protocol rule 12 says one feature per checkout and no refusal enforces
it. And `Claims` matches any branch whose name mentions an id, so `plan/fold-in-0002-fix`
makes the board report 0002 as taken while `kit audit` exits 0 saying the docs and the
repo agree.

## Non-goals

- **A second coordination mechanism.** The pushed ref stays the claim. A chore is claimed
  the way a feature is, or the guarantee is two guarantees.
- **A doc for every chore.** What a chore needs is a name nobody else can take, not a
  design someone must read.
- **Retiring `plan/` branches.** A drafting round is still one branch merged once.

## Design

**A chore is claimed by the same push that claims a feature.** `kit chore "<what>"` mints
a claim from a title, pushes the branch that takes it, and records one line — id, title,
who, when — in a file the board reads. No `## Problem`, no `## Design`, no boxes: a chore
that needs any of those is a feature, and the operator is told so rather than guessed at.
The alternative — leaving chores uncoordinated and documenting the risk — is what the kit
does today, and it is the one class of work where the kit's headline promise does not hold.

**A chore takes an id, so the record can be cited.** Branch names, commits and the board
all key on the id, and a chore with no id cannot appear on the board at all. This is why
this feature waits on 0058: handing out an id twice is worse for chores than for features,
because there is no doc whose contents disambiguate them.

**One checkout holds one claim, and `claim` says so.** The condition is already computed —
`next` prints "take one of those in a worktree of its own, never here" — and never
enforced. `ClaimFeature` refuses when this checkout already holds a claim, which collapses
protocol rules 11 and 12 into a refusal and one line of prose.

**A claim is a branch that claims, not a branch that mentions.** What counts as a claim is
`Claims`, and it currently matches a substring. Narrowing it to the branch a claim actually
creates would close the `plan/`-prefix hole, and would also change a guarantee the kit
states out loud — see `## Open questions`.

## Tasks

- [x] `kit chore "<what>"` claims a named unit of work and records one line
- [x] The board lists claimed chores, and `kit release` ends a chore claim
- [x] `claim` and `new` refuse when this checkout already holds a claim
- [x] The protocol routes non-feature work to `kit chore` instead of to an unclaimed branch

## Open questions

- **Does narrowing `Claims` belong here or nowhere?** `docs/TECH.md` and `kit help` both
  state that local branches, remote branches and worktrees all count as claims, and a
  conformance case pins that a namespaced branch in a worktree is one claim. Narrowing the
  match closes the `plan/fold-in-0002-fix` hole and weakens that. The leaning is to keep
  the broad match and refuse the name at `kit new` instead, so nothing that exists stops
  counting; the alternative is a `plan-branch-claims` finding at HIGH.
- **Where does the chore record live?** One line appended to `docs/CHORES.md` is the
  cheapest thing the board can read. A chore with no file at all — the pushed branch as
  the whole record — is cheaper still and cannot drift, but then a finished chore leaves
  nothing behind. The leaning is the file.

## Log

- Both open questions went their stated way. `Claims` keeps its broad match and the
  `plan/fold-in-0002-fix` hole is closed at `kit new`; the record is a file,
  `docs/CHORES.md`, one line per chore.
- The record is committed onto the claim branch rather than written into the checkout.
  `kit chore` runs from the trunk checkout, so a line left uncommitted there would never
  reach the worktree the agent is sent to, and a lost push would leave it standing.
- Two refusals the design did not name: a chore title carrying a four-digit number, and a
  chore under a name somebody already holds. Sequential ids cannot collide, so without the
  second nothing would have stopped two agents doing one job.
- The third box grew `new`, because closing the `plan/` hole there is the same rule the box
  already states: one checkout, one claim. `new` exempts a feature's own claim branch, so
  work discovered mid-build is still written down where it was found.
