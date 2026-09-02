---
id: 0023
title: Stop the commands doing quietly what the board forbids
status: Shipped
created: 2026-08-16
shipped: 2026-08-16
needs:
related: 0021-let-agents-work-from-drafts-and-make-accepting-a-signal.md
---

## Problem

`next` and the commands disagree about what may be worked, and the commands win. Measured on
a fresh project with `0002` declaring `needs: 0001`:

```
$ kit next          0002 blocked — waiting on: needs 0001 to ship first
$ kit claim 0002    claimed                                    (exit 0)
$ kit ship  0002    shipped  (was Accepted)                    (exit 0)
$ kit audit         MEDIUM  shipped-early
```

`Survey` consults `UnmetNeeds`; `ClaimFeature` and `Ship` never do. So an agent that claims
by id without reading the board builds on a foundation that is not there, and `ship` mints a
`shipped-early` finding on a doc the protocol then forbids anyone to edit. 0021 made it
worse: `status:` no longer gates anything, so `needs:` is the only ordering constraint left
and it is unenforced at the two moments that matter.

`ship` has a worse version. From a checkout whose own board says taken, `kit ship` exits 0
and freezes another agent's feature mid-flight — `Shipped` is history nobody may edit, so
the agent holding the branch can no longer touch its own doc, and `audit` reports nothing.

`kit drop` has two silent failures. Dropping a feature others declare in `needs:` strands
them without a word, and `next` goes on promising "needs 0001 to ship first" about a feature
that can never ship. And on a doc with no `## Log` section the reason is discarded:
`AppendToSection` reports the heading was absent and `Drop` ignores the answer.

## Non-goals

- Making `needs:` mandatory. An unanswered field is a real problem and a separate one.
- Blocking `accept`. Accepting is a statement about the design, not a start, and blessing
  something that waits on another feature is normal.
- A `--force`. An escape hatch on the only remaining ordering constraint makes it advisory.

## Design

`claim` and `ship` reuse `UnmetNeeds`, the same call `Survey` makes, so the board and the
commands cannot disagree.

`claim` refuses as `NotYours`, exit 1: the agent is meant to take something else, which is
the instruction a lost race already carries. `ship` refuses as an ordinary error, exit 2 —
the work is done and the plan is wrong, so the operator has to look, and the message names
which ids are unshipped.

`ship` also refuses a feature claimed by a branch that is not this checkout's. `Survey`
already computes exactly this as `Mine`. Shipping from the worktree holding the claim is the
documented path; shipping from anywhere else is one agent freezing another's work. The
message names the way out, because refusing without one is how an agent gets stuck.

`drop` reports rather than refuses. A feature that will not be built is exactly when you
want to know who was waiting on it, and refusing would leave the dropped doc stuck open.

`next` stops promising a wait that will never end. Where a `needs:` id names a doc that is
Dropped or Superseded, the line says so instead of "to ship first".

## Tasks

- [x] `ClaimFeature` refuses unmet `needs:` as `NotYours`, naming the ids
- [x] `Ship` refuses unmet `needs:`, exit 2, naming the ids
- [x] `Ship` refuses a feature claimed by another checkout's branch, naming the way out
- [x] `Drop` reports features whose `needs:` name the dropped id
- [x] `Drop` creates `## Log` when it is absent rather than discarding the reason
- [x] `Survey` distinguishes a dependency that can never ship from one not yet shipped
- [x] README and `kit help` for claim, ship and drop describe the new refusals

## Log

- 2026-08-16 — the board refused 0003 and `kit claim 0003` took it anyway, on a project
  bootstrapped from an empty directory.
