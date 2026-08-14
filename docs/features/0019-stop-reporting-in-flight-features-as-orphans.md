---
id: 0019
title: Stop reporting in-flight features as orphans
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
check:               # optional — date to check whether this worked. Only for bets.
checked:             # date you checked; write the outcome in ## Log
needs: 0018          # shares docs/TECH.md and README.md with the branch it is cut from
related: docs/features/0017-read-the-board-from-real-git-state.md
---

## Problem

The protocol requires a draft to wait on a branch of its own, because nothing is
committed to the trunk directly. Audit the trunk while any of that is in flight and it
reports the protocol being followed as two defects:

```
MEDIUM  orph  (orphan-claim)
          branch "0001-a-feature" cites a feature that does not exist
          fix: delete the branch, or create the doc it claims with kit new
MEDIUM  orph  (no-plan)
          the project has commits but not one feature doc
```

Neither is true. The doc exists, on the branch that carries it, which is where the
protocol put it. The suggested fix — delete the branch — would destroy the draft.

This was found by auditing this repo while three of its own features were open pull
requests, so it fires on every in-flight feature of every project, permanently, from the
one checkout a human looks at. 0005 named the cost: a check that fires on work done right
trains everyone to stop reading the tool. `unfilled-template` and `stalled` were made
trustworthy in 0016 and 0017 and this undoes some of that in the same output.

## Non-goals

- **Fetching to find out.** 0017 settled that reads ask the remote one cheap question and
  never mutate refs. Inspecting an unfetched branch's tree is not that question.
- **Dropping `orphan-claim`.** A branch citing a feature nobody ever wrote is real: it
  means a doc was deleted or renamed and the branch was left behind, and the ID it holds
  is now unclaimable.
- **Reading pull-request state.** The kit knows docs and refs. Whether a branch is
  "in flight" is not a fact it can see, and it does not need to be — whether the branch
  carries the doc is the same answer and git already has it.

## Design

### A finding needs positive evidence

Both checks currently reason from absence: the doc is not *here*, so it does not exist.
Absence in one checkout says nothing about a branch, so both become evidence-based.

`orphan-claim` reports a branch only when the kit can **see** that branch's tree and
**confirm** it does not carry the feature doc named after it:

```
orphan(branch) = resolvable(branch) AND NOT exists(branch:<features>/<branch>.md)
```

`resolvable` tries the local ref then each remote-tracking one, the same widening
`trunkRef` uses. A branch that cannot be resolved at all — a remote-only branch nobody has
fetched — is not reported. That is a deliberate false negative: a genuine orphan existing
only on a remote goes unnoticed until somebody fetches, which is a far cheaper mistake
than telling an agent to delete a draft it cannot see.

`no-plan` stops firing when any claim-shaped branch exists. Its job is to catch adoption
failing — commits happening with nothing written down — and a branch named `0001-…` is
somebody writing something down. It does not need the doc's contents to know that.

### Why not compare against the trunk instead

Reading the trunk's tree for the doc would answer the same question for a merged feature
and the wrong one for an in-flight one, which is the case that matters. The branch is what
carries the claim, so the branch is what gets asked.

## Tasks

- [x] `BranchCarriesItsDoc` on `Repo`, resolving a branch locally then through each remote,
      and `OrphanClaims` reporting only branches it can resolve and confirm.
- [x] `no-plan` stands down when a claim-shaped branch exists.
- [x] Conformance: a draft on its own branch produces neither finding from the trunk, a
      branch citing a deleted doc still produces `orphan-claim`, and an unresolvable
      branch name produces nothing.

## Log

- 2026-08-14 — found by running `kit audit` on this repo with 0016, 0017 and 0018 open as
  pull requests. The kit reported its own correct workflow as two MEDIUM defects, one of
  them advising a fix that deletes work.
