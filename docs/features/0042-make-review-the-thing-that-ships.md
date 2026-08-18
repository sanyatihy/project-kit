---
id: 0042
title: Make review the thing that ships a feature
status: Draft
created: 2026-08-18
shipped:
check:
checked:
review:
needs:
related: docs/features/0028-see-what-is-waiting-for-review.md, docs/features/0032-drop-the-in-progress-status.md
---

## Problem

The builder sets `Shipped` and then asks for review, so the status means *the builder
believes this is done*. Nothing in the docs or the branches records that anybody else
agreed, which is the fact a loop running unattended most needs to carry.

Review is also the only instruction on the board with no command behind it: `next` tells an
idle agent to review one somebody else built, then leaves it to invent which branch, what to
diff against, and what to check. What keeps the two agents apart is prose, at the tier
`docs/TECH.md` says prose fails — measured there: the rule against merging your own work was
broken twice by an agent that had read it, while `ship`'s coded refusal was obeyed the first
time it fired.

## Non-goals

- **Running anything.** Not tests, not builds. The kit is not a test runner, and a verdict it
  produced would be one nobody could audit. It says what to establish; the reviewer does it.
- **Merging.** The forge owns that and branch protection is the floor. `review` names the
  merge command for the forge `init` detected, the way `claim` names the worktree command.
- **A status for waiting.** 0032 removed the last one. Finished-and-waiting is derivable —
  every box ticked, on a pushed branch, not yet Shipped — and a second copy could only
  disagree with the branches.

## Design

**`Shipped` is the reviewer's to set, in a commit of its own.** That commit is the record:
its author is somebody other than the builder, and it lives in git as long as the branch
does. No new field and no new state. The `shipped:` date becomes the date the work was
accepted rather than the date its author stopped typing.

**A claim is a branch name, so the reviewer must not be on the branch.** A checkout on
`0042-…` *is* a claim on 0042, and from one the kit cannot tell reviewer from builder.
`kit review` prints a detached worktree — `git worktree add --detach <path> origin/<branch>`
— which keeps the claim single-holder while a second party writes the status commit. The
refusal rests on that mechanism rather than on etiquette.

**`kit ship` inverts its refusal.** It refuses the checkout holding the claim, where it used
to require it, and that refusal carries the builder's next instruction: push and stop, a
reviewer ships this. The builder still runs the command they would have run and is told what
happens next, so the ending closes rather than opening a vacuum. Its other verifications —
boxes ticked, needs met — stay exactly as they are, now serving the reviewer.

**`kit review [<id>]`** takes the feature that has waited longest and that this checkout may
review, or the one named. It fetches the branch, prints `<trunk>...<branch>` so the diff is
the branch's own work rather than the trunk moving underneath it, states what a review must
establish, and names the merge command. `review: human` refuses to print one and says a
person takes this: any non-empty value counts, 0029's fail-safe direction.

**The board reads the new shape.** `AwaitingReview` stops meaning *Shipped on the branch* and
starts meaning *every box ticked on a pushed branch, not Shipped, not on the trunk*.
`done-not-shipped` must stay quiet for exactly that state, or it reports MEDIUM on every
feature correctly waiting its turn.

**What this guarantees, and what it does not.** It enforces *not from the claim*, which is
unambiguous. It cannot enforce *a different agent*: agents here share one credential and four
agents' commits all carried the operator's name. A lone operator is not locked out, since a
detached checkout is one command. The honest claim is that the loop becomes legible and the
friction points the right way, while branch protection stays the only real enforcement.

## Tasks
- [ ] `kit ship` refuses the claim holder, the refusal naming what happens next; its other
      verifications unchanged; conformance both ways
- [ ] `kit review [<id>]`: the detached review worktree, the diff range, what a review must
      establish, the merge command, and refusal when this checkout built it
- [ ] `AwaitingReview` reads ticked-and-pushed rather than Shipped, and `done-not-shipped`
      stays quiet while a feature waits for review
- [ ] `next`'s review guidance names `kit review`; protocol, `docs/TECH.md`, README and
      `kit help` say `Shipped` is the reviewer's act and why the commit is the record
- [ ] One conformance run of the whole loop: build, push, review from a detached checkout,
      ship, merge, and the refusal at each end

## Open questions
- Should a rejected review be visible on the board? Rejecting is cheap here — nothing to
  undo, since `Shipped` was never set — but a branch that was read and turned down looks
  exactly like one nobody has opened, so agents will keep picking it up. Leaning: leave it to
  the forge, and watch for a feature being reviewed twice before building anything.

## Log
- 2026-08-18 — the kit is not a test runner, so `review` sets a review up and refuses the
  wrong reviewer rather than producing a verdict.
- 2026-08-18 — ordering decided before any code: `Shipped` set by the builder records only
  that the builder stopped, which is the one thing the branches already show.
