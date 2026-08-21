---
id: 0056
title: Every command reads the branch that exists
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

A claim lives on the remote, and the kit answers questions about it from whichever copy the
machine happens to hold. One reproduced consequence is a deleted commit: an agent ran
`kit release`, the guard asked a stale local ref, and another checkout's pushed work became
unreachable at exit 0 with no `--force`.

## Non-goals

- **Fetching inside `next`.** The board already asks `ls-remote`; this is about which answer
  it believes.
- **Refusing to work offline.** `--offline` stays, and stays explicit about what it cost.

## Design

**The ref about to be deleted is the ref that is asked.** `Release` resolves the branch
locally first, so a stale local copy answers "carries no work" while the remote carries
commits. `BranchBuildsOn` is asked of the remote ref when one exists, and `release` refuses
outright when the two copies disagree — an unknown answer must not be the permissive one in
a command that deletes.

**Progress comes from the branch the remote has.** `next --help` says a taken feature's
count "comes from its claim branch, where the boxes are actually ticked". It comes from
whichever copy resolves first, so two checkouts print different progress for one feature in
the same instant, and `stalled` measures its age from the same stale ref.

**Every remote is a remote.** `branchName` strips only `origin/`, so on a clone whose remote
is named `upstream` the trunk resolves to `upstream/main`, `trunkRef` resolves nothing,
`stalled` can never fire and `release` refuses everything. Strip any configured remote prefix.

**An answer that cannot change inside one command is asked once.** `Trunk()` uses an empty
string as its "not asked yet" sentinel — the one memo on `Repo` with no companion flag — so
when no trunk resolves it re-runs `findTrunk` on every call. Measured at 2N+1 git processes,
linear in the size of the board, which is the one shape `gitbudget_test.go` exists to catch.
It misses it because its fixture always resolves a trunk.

Reproduced:

| state | what the kit did |
| --- | --- |
| stale local ref, remote has commits | `release` deleted them, exit 0 |
| two checkouts, one feature | `0/2` here, `1/2` there, same instant |
| remote named `upstream` | no trunk resolves; `stalled` can never fire |
| no trunk resolvable, 40 features | 89 git processes, growing with the board |

## Tasks

- [x] `release` asks the remote ref, and refuses when local and remote disagree
- [ ] Progress and `stalled` read the authoritative ref, and a test drives two checkouts
- [ ] Any remote prefix is stripped, not just `origin/`
- [ ] `Trunk()` is memoised on a flag, and the budget fixture covers a repo with no trunk

## Log
