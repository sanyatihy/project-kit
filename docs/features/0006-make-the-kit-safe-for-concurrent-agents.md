---
id: 0006
title: Make the kit safe for concurrent agents
status: Draft
created: 2026-08-12
shipped:
check:
checked:
related: 0005-make-claims-and-blocking-mean-what-the-docs-say.md
---

## Problem
The kit's rules were written for one operator shipping weekly, and they hold at that
scale. Running two agents at once breaks three of them, and two break silently.

Measured, not assumed:

- **Two agents both get the same ID.** `kit new` takes the next free ID from the
  directory listing, so two branches off the same base both produce `0002`. The
  merge is **clean** — different filenames, nothing for git to conflict on — and
  `kit audit` reports "docs and repo agree". The result is two features with
  `id: 0002`, and the ID is the kit's primary key: one branch named `0002-…` then
  claims *both*, and one INBOX line blocks *both*.
- **The claim is not atomic.** Rule 3 makes a branch the claim, but creating it and
  pushing it are two steps with a network round trip between them. Two agents can
  both pass "is it taken?" before either is visible. The rule describes an intent
  the tool never enforces.
- **`kit next` routes agents into each other.** Standing in a checkout whose HEAD
  claims 0001, `next` reports 0001 as "taken" — by you — and offers a *different*
  feature under "start here". Followed literally, that starts a second feature in
  the first one's checkout.

The cadence is off by an order of magnitude in the same direction. `InboxStaleDays`
is 7 and `StalledDays` is 30, both tuned for a human shipping weekly. Measured:
`kit next` shows a blocked feature **instantly**, and `kit audit` stays silent about
it for a week. An unanswered question costs an idle agent in minutes now, so the
board is already the right instrument and only `audit`'s escalation lags.

0005 deferred `kit claim` and retuning the thresholds on the grounds that the signal
to build them is real usage rather than anticipation. That signal has arrived.

## Non-goals
- **Hash IDs.** They would fix collisions by construction, which is what `beads`
  does, but sequential IDs are load-bearing here: rule 1 orders by lowest ID, and
  IDs are cited in commit messages and branch names. See `## Open questions`.
- **A database.** The alternative to convention is not a state store. Git already
  has an atomic primitive — creating a ref either succeeds or fails — and the whole
  claim protocol can be built on it without anything to keep in sync.
- **Cross-feature dependencies.** Still deferred, still for the same reason. Nothing
  measured here needed them.
- **Choosing how many agents run at once.** That number is a function of agent
  reliability, verification strength, and review bandwidth. Two of those move on
  their own, so the kit should report the state and never encode a count.
- **Worktree management.** Creating, listing, and removing worktrees is git's job
  and a launcher script's job. The kit's business is only refusing to hand out work
  in a checkout that is already occupied.

## Design
**Uniqueness is auditable, so audit it.** `duplicate-id` is the check that catches
what the merge cannot, and it goes first because everything else assumes the ID
identifies one feature. It is also the honest fallback for allocation: whatever
`kit new` does, two agents offline on two branches can always collide, and a check
that fires is better than a scheme that pretends it cannot happen.

**The push is the claim.** `kit claim <id>` creates the branch and pushes it, and a
rejected push means someone else got there first — reported as a non-zero exit, not
a warning. This is the same guarantee `beads` gets from an atomic database write,
taken from the ref store instead, and it needs no new state. It also makes rule 1
enforceable rather than advisory: an agent that cannot push has not claimed anything.

**`next` reports the checkout it is standing in.** The claim on HEAD is *yours* —
reporting it as taken is the same class of defect 0005 existed to fix, a tool saying
something untrue. And a checkout already on a claim must not get "start here" for a
different feature; what it gets instead is the fact, and the protocol says what to
do about it.

**Thresholds follow the cadence of the thing being watched**, not the operator's
calendar. `next` is already instant and needs no change; `audit` is the lagging half.

**Protocol last**, because the boxes above change what it has to say.

## Tasks
- [ ] Audit: `duplicate-id` — two feature docs sharing one `id:`, which merges clean and makes claims and blocking ambiguous
- [ ] `kit claim <id>` — create and push the branch; a rejected push is a lost race and a non-zero exit
- [ ] `kit next` reports the claim on the current HEAD as yours, not as taken
- [ ] `kit next` will not headline a different feature in a checkout that is already claimed
- [ ] `StalledDays` and `InboxStaleDays` retuned from weekly-human to agent cadence
- [ ] Protocol: the worktree is the isolation, the push is the claim, and where the ID race can still bite

## Open questions
- Should `kit new` try to avoid the collision at all, or only let `duplicate-id`
  catch it? Fetching and reading remote branches' docs would narrow the window
  without closing it, and it makes `new` a network operation. Leaning: no — keep
  `new` offline and cheap, and let the check carry it.
- Hash IDs would end the race outright but cost the ordering rule and stable
  citation. Leaning: not worth it, but this is the one place `beads` is
  straightforwardly right and it should be settled deliberately rather than by
  default.

## Log
- 2026-08-12 — the ID collision was found by running it rather than reasoning about
  it. The severity is in what the test showed and not in the collision itself: the
  merge is clean and `audit` reports agreement, so nothing in the system notices.
  That moved it from an annoyance to the first box.
- 2026-08-12 — rejected a state store. `beads` makes the opposite bet and its own
  README prices it: a database to install, two operating modes, a sync step separate
  from git, and an export that is explicitly not the source of truth. Git's ref
  creation is already atomic, which is the only property that was actually missing.
- 2026-08-12 — measured the board before retuning it. `kit next` surfaces a block
  immediately and `audit` waits seven days, so the fix is `audit`'s threshold alone.
  Worth recording because the assumption going in was that the board was slow.
- 2026-08-12 — tested the terminal-to-editor handoff to see whether the kit owed it
  anything. It does not: sessions and editor windows both bind by working directory,
  so a worktree gets its own without being told. That is why worktree management
  stays a non-goal — the isolation is git's and the tooling around it is a launcher
  script's, and the only thing left for the kit is refusing to hand out work in a
  checkout that is already occupied.
