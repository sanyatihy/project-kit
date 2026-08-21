---
id: 0058
title: The kit hands out an id once
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

An id is what a branch, a commit message and the board all cite, so it has to name one
feature for ever. `NextID` reads the highest id on disk and adds one, and disk is not a
record of what has been handed out.

Two ways it hands the same number out twice. Deleting the highest-numbered doc frees its
id: `0004-delta.md` was removed and the next `kit new` wrote `0004-epsilon.md`, after
which every branch and commit citing 0004 names a different feature. Deleting a doc from
the middle of the range is safe, which is why this has stayed quiet. And two agents
drafting at once both read the same highest id and both write it — the protocol carries a
whole section about this, `audit` reports `duplicate-id` at HIGH afterwards, and the fix
is a rename plus a branch rename on work that has already started.

`NextID`'s own comment states this must not happen. Nothing enforces it, and the
workaround in practice is to mint ids one at a time by hand, which is what the round that
drafted this feature did.

## Non-goals

- **A counter file.** A number in a tracked file is a line two agents edit at once, which
  is the collision this feature exists to remove.
- **Renumbering history.** Ids already handed out stay where they are, including the ones
  a previous collision produced.
- **Refusing to work offline.** A project with no remote still drafts; it just cannot be
  protected from a second agent it cannot see.

## Design

**The id is taken by the same compare-and-swap that takes a feature.** `kit new` pushes
`refs/kit/ids/<id>` before writing the doc, and a push that does not create the ref means
the id is gone and the next one is tried. A ref update is a compare-and-swap against the
value the server advertised, so of any number of simultaneous drafters exactly one gets a
given number — the property `kit claim` already relies on, applied one step earlier.
`BACKLOG.md` proposes this and it is promoted here because the workaround has now been
used.

**A reserved id is never reused, however the doc leaves.** The ref outlives the file, so
deleting a doc no longer frees its number and `kit drop` stops being the only thing
standing between a deleted file and a duplicated id. `kit drop` keeps its own job:
retiring a feature visibly, with dependants named.

**No remote means no protection, said out loud.** With no remote there is nothing to
compare and swap against, so `kit new` falls back to the disk scan and says which
guarantee it could not give. Silently degrading is what makes the current behaviour hard
to see.

**The refs are cheap and the board never reads them.** They cost one push per draft and
answer exactly one question. `next` and `audit` keep reading the docs, so a stale ref
namespace cannot make the board wrong.

## Tasks

- [x] `kit new` reserves the id with a pushed ref, and retries when it loses
- [x] A deleted doc's id is never handed out again, and a test drives it
- [ ] Two agents drafting at once get different ids, and a test drives both clones
- [ ] With no remote, `kit new` names the guarantee it cannot give

## Open questions

- **Should `audit` report a doc whose id was never reserved?** Every doc written before
  this ships is in that state, so the check would fire on every existing project until
  the refs are backfilled. The leaning is no check, and a one-time backfill in `kit init`.

## Log
