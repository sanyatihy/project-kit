---
id: 0058
title: The kit hands out an id once
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs:
---

## Problem

An id is what a branch, a commit message and the board all cite, so it has to name one
feature for ever. `NextID` reads the highest id on disk and adds one, and disk is not a
record of what has been handed out.

Two ways it hands the same number out twice. Deleting the highest-numbered doc frees its id:
`0004-delta.md` was removed and the next `kit new` wrote `0004-epsilon.md`, after which every
branch and commit citing 0004 names a different feature. Deleting from the middle of the
range is safe, which is why this stayed quiet. And two agents drafting at once both read the
same highest id and both write it, after which the fix is a rename plus a branch rename on
work that has already started.

`NextID`'s own comment states this must not happen. The workaround in practice is to mint
ids one at a time by hand, which is what the round that drafted this feature did.

## Non-goals

- A counter file. A number in a tracked file is a line two agents edit at once, which is the
  collision this removes.
- Renumbering history. Ids already handed out stay, including ones a collision produced.
- Refusing to work offline. A project with no remote still drafts; it cannot be protected
  from a second agent it cannot see.

## Design

The id is taken by the same compare-and-swap that takes a feature. `kit new` pushes
`refs/kit/ids/<id>` before writing the doc, and a push that does not create the ref means the
id is gone and the next one is tried. Of any number of simultaneous drafters exactly one gets
a given number — the property `kit claim` relies on, applied one step earlier.

A reserved id is never reused, however the doc leaves. The ref outlives the file, so deleting
a doc no longer frees its number. `kit drop` keeps its own job: retiring a feature visibly,
with dependants named.

No remote means no protection, said out loud. With nothing to compare and swap against,
`kit new` falls back to the disk scan and says which guarantee it could not give. Silently
degrading is what makes the current behaviour hard to see.

The refs are cheap and the board never reads them. They cost one push per draft, so a stale
ref namespace cannot make the board wrong.

## Tasks

- [x] `kit new` reserves the id with a pushed ref, and retries when it loses
- [x] A deleted doc's id is never handed out again
- [x] Two agents drafting at once get different ids, driven across two clones
- [x] With no remote, `kit new` names the guarantee it cannot give

## Log

- 2026-08-21 — no `audit` check for a doc whose id was never reserved; `kit init` reserves the
  ids a project already holds instead. The backfill is what makes the second box true of a
  board that predates this.
- 2026-08-21 — a remote that exists and cannot be reached now fails the draft. The non-goal
  covers a project with no remote, and there was no reading of it that covered an unreachable
  one. In `BACKLOG.md`, triggered by an agent that could not draft.
