---
id: 0043
title: Set the reviewer up and record a turn-down
status: Dropped
created: 2026-08-18
shipped:
needs:
related: docs/features/0013-never-leave-an-agent-without-an-instruction.md
---

## Problem

Review is the only instruction on the board with no command behind it. `next` tells an idle
agent to review one somebody else built, then leaves it to work out which branch, what to
diff against, what to check, and how to set up a checkout that is not the claim. Every other
step of the loop is a command that refuses what it should refuse.

A turn-down is invisible in the same way. A branch that was read and rejected looks exactly
like one nobody has opened, so the board offers it again to the next idle agent, and the
work of reviewing it is repeated until somebody happens to approve.

## Non-goals

- Holding a review open. No lock ref, no "being reviewed" state: two agents reviewing
  one branch waste a reading, and 0042's ref creation already stops the second from
  shipping it twice. A lock would be state to expire and clean up for a duplicate that
  costs one diff.
- Carrying the reason. The turn-down records *that* and *at which commit*. Why belongs
  in the pull request, where the argument already lives; the kit is not an issue tracker.

## Design

`kit review [<id>]` takes the feature that has waited longest and that this checkout may
review, or the one named. It prints the fetch, the detached worktree 0042 requires, the
range `<trunk>...<branch>` so the diff is the branch's own work rather than the trunk moving
underneath it, what a review has to establish, and the merge command for the forge. It
refuses the checkout holding the claim, since that is the one that may not ship it.

The merge command means consulting the forge at review time. `docs/TECH.md` says the
forge is detected at `init` and never consulted again; `DetectForge` reads a directory and a
remote and keeps no state, so calling it here costs nothing and changes no answer. The
sentence changes with this feature rather than quietly becoming untrue.

`kit review <id> --changes` creates `refs/kit/changes/<id>` at the commit turned down.
Symmetric with 0042's reviewed ref and the same primitive. The board then shows the feature
as *changes requested* rather than awaiting review, and stops offering it, until the branch
moves past that commit — at which point the ref names an ancestor and the entry is waiting
again. Nothing to clear by hand: new work is what clears it.

`review: human` refuses to print a merge command and says a person takes this one. Any
non-empty value counts, which is 0029's fail-safe direction.

## Tasks
- [ ] `kit review [<id>]`: pick, fetch, detached worktree, diff range, what to establish,
      merge command; refusal when this checkout holds the claim
- [ ] `--changes` creates `refs/kit/changes/<id>` at the commit read, and `review: human`
      withholds the merge command
- [ ] The board reads the changes ref: *changes requested* while it names the branch's tip
      or an ancestor of nothing newer, waiting again once the branch moves past it
- [ ] `next`'s review guidance names `kit review`; `kit help review`, README and the
      `docs/TECH.md` sentence about the forge

## Log
- 2026-08-18 — split from 0042 so the protocol lands without waiting on the ergonomics, and
  each is one pull request.
- 2026-08-20 — dropped: Parked with 0042, which it needs: the reviewer's command is only worth building once review by a second agent is being run.
