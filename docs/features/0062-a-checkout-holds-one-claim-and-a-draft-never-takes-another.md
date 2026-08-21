---
id: 0062
title: A checkout holds one claim, and a draft never takes another
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

Two rules say one checkout does one feature, and no refusal enforces either. `kit claim
0008` from inside 0007's worktree exits 0, so one checkout ends up holding two claims and
the second branch is built on the first one's work. `next` already computes the condition
and prints "take one of those in a worktree of its own, never here"; nothing acts on it.

The same gap lets a branch claim a feature nobody is working. A doc drafted while HEAD is
`plan/fold-in-0002-fix` makes the board report 0002 as `taken` and `kit audit` exit 0
saying the docs and the repo agree. `Claims` matches any branch whose name mentions an id,
which is deliberate — worktrees and namespaced branches must count — so the name is what
has to be refused, at the one moment the kit is present while it is chosen.

Both were built and verified on 0057's branch before it was dropped: commit `0684b67`,
103 insertions across `claim.go`, `newdoc.go`, `help.go` and three test files.

## Non-goals

- **Changing what counts as a claim.** `Claims` stays broad. `docs/TECH.md` and `kit help`
  both state that local branches, remote branches and worktrees all count, and a
  conformance case pins it.
- **A second kind of work.** 0057 proposed one and was dropped; the protocol sentence that
  made it look necessary goes in 0061.

## Design

**A checkout that holds a claim takes no other.** `ClaimFeature` refuses when `Mine` is
true of any other feature, which turns protocol rules 11 and 12 into a refusal and one
line of prose. The fix line names the worktree to make, because that is what the agent was
going to be told next.

**A draft written on somebody else's claim is refused, not reported.** `kit new` refuses
while HEAD's branch name cites a four-digit id that is not this checkout's own claim — its
own worktree may still draft a discovered doc, which is the case that has to keep working.
This is the kit's stated preference for a refusal over a rule, applied to the one rule
that fails silently: a `plan/` branch carrying an id corrupts the board and audits clean.

**The work is a cherry-pick, not a rewrite.** `0684b67` is self-contained and touches
nothing 0057 added; it lands first and the boxes below verify it against a trunk that has
moved since.

## Tasks

- [x] `0684b67` lands on this branch, and `make check` passes against current main
- [x] `claim` refuses a checkout that already holds a claim, named in `kit claim --help`
- [x] `kit new` refuses to draft on a branch citing an id that is not this checkout's claim
- [x] 0057's claim branch is released, and the board shows no work nobody is doing

## Log
