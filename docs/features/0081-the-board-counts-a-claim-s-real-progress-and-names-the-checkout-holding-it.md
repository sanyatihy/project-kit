---
id: 0081
title: The board counts a claim's real progress and names the checkout holding it
status: Draft
created: 2026-09-08
submitted:
needs:
---

## Problem

`next` counts a claim's boxes on the tip the remote advertises, so commits made in a
worktree of this clone and not yet pushed are invisible: three claims mid-flight read
0/6, 0/5 and 0/6 from the trunk checkout beside them while their branches carried 0, 3
and 1 ticked boxes. The board never says a claim is checked out here either, so the
reader was told every feature is somebody else's and waiting is the right outcome, with
the work sitting in `.worktrees/` a directory away.

## Non-goals

Reading a claim's working tree. A box is ticked in its implementing commit, so the
branch tip is the record and an uncommitted tick is not yet a fact.

A group of their own for locally held claims. They are still not this checkout's to
work, and a group beside `taken` would say they were.

Moving the anchor for the other `claimRev` callers. Being on the review board means
pushed, and `audit` reads a claim the same way from every clone.

## Design

Progress and last-moved come from the freshest revision this machine can prove for a
claim: the local head when it exists and contains the tip the remote advertises, and
that tip otherwise. Diverged or behind stays remote, so a stale local ref never
overwrites the shared answer. A clone that has never fetched the branch still reports
no count, which is the existing answer for a fact that cannot be read.

`claimRev` keeps the remote anchor. Submission is what `AwaitingReview` reads through
it, and an unpushed submit is on nobody's review board.

An item carries the worktree holding its claim, relative to the repo root and empty
when no worktree of this clone has it. The taken entry prints it, and the guidance says
how many claims are checked out here and that `kit release <id>` takes back one whose
agent is gone. Neither may tell a reader to work in a checkout that is not theirs —
one feature per checkout is what the path makes easy to break.

## Tasks

- [x] Count a claim's boxes and last-moved date from the local head when it contains the advertised tip
- [ ] Name the worktree holding a claim on its taken entry, in text and JSON
- [ ] Say in the guidance which claims this clone is holding and how to take one back

## Log
