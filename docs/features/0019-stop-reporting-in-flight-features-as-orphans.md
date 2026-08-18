---
id: 0019
title: Stop reporting in-flight features as orphans
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
check:
checked:
needs: 0018
related: 0017-read-the-board-from-real-git-state.md
---

## Problem
A draft waits on its own branch, because nothing is committed to the trunk directly. Audit
the trunk while any of that is in flight and the protocol being followed is reported as two
defects: `orphan-claim` — *"branch 0001-a-feature cites a feature that does not exist, fix:
delete the branch"* — and `no-plan`. Neither is true, and the suggested fix would destroy the
draft. Found by auditing this repo with three of its own features open as pull requests, so
it fires on every in-flight feature of every project.

## Non-goals
- **Fetching to find out.** 0017 settled that reads ask one cheap question and never mutate refs.
- **Dropping `orphan-claim`.** A branch citing a doc nobody wrote is real: the ID it holds is unclaimable.
- **Reading pull-request state.** Whether the branch carries the doc is the same answer, and git already has it.

## Design
**A finding needs positive evidence.** Both checks reasoned from absence — the doc is not
*here*, so it does not exist — and absence in one checkout says nothing about a branch.

```
orphan(branch) = resolvable(branch) AND NOT exists(branch:<features>/<branch>.md)
```

`resolvable` widens from the local ref to each remote-tracking one. A branch nobody has
fetched is passed over: a deliberate false negative, because missing a real orphan until
someone fetches is far cheaper than telling an agent to delete a draft it cannot see.

`no-plan` stands down when any claim-shaped branch exists — its job is catching adoption
failing, and a branch named `0001-…` is somebody writing something down.

Comparing against the trunk instead would answer correctly for a merged feature and wrongly
for an in-flight one, which is the case that matters.

## Tasks
- [x] `BranchCarriesItsDoc` on `Repo`, resolving a branch locally then through each remote,
      and `OrphanClaims` reporting only branches it can resolve and confirm.
- [x] `no-plan` stands down when a claim-shaped branch exists.
- [x] Conformance: a draft on its own branch produces neither finding from the trunk, a
      branch citing a deleted doc still produces `orphan-claim`, and an unresolvable
      branch name produces nothing.

## Log
- 2026-08-14 — the checks reported this repo's own correct workflow as two defects, one
  advising a fix that deletes work.
