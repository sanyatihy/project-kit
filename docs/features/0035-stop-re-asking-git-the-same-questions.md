---
id: 0035
title: Stop re-asking git the same questions
status: Draft
created: 2026-08-17
shipped:
check:
checked:
needs:
related: 0028-see-what-is-waiting-for-review.md
---

## Problem
One `kit next --offline` on a 30-feature project with 10 claims spawns **116 git
processes**:

```
32 rev-parse   31 symbolic-ref   31 remote   10 log   9 show   2 branch   1 worktree
```

`HeadBranch()` and `remotes()` shell out on every call and are called once per feature, so
62 of those 116 are the same two questions asked over and over. Neither answer can change
inside one command.

This is the shape that already cost once: `remoteHeads` was unmemoised and took `kit audit`
on this repo from under a second to 1m57s. The pattern survived in two more places, and it
compounds with the fleet board — `kit next ~/code/*` pays it per project.

## Non-goals
- **Caching between runs.** Nothing on disk, no state the kit has to invalidate. These
  answers live for one command and die with it.
- **Batching the reads that genuinely differ**, such as one `git show` per claim branch to
  count its boxes. Those ask different questions; only the repeated ones are waste.
- **A general query layer.** Three memoised accessors, in the shape `Trunk()` already uses.

## Design
`HeadBranch()` and `remotes()` memoise, exactly as `Trunk()` and `remoteHeads()` already
do: an `asked` flag beside the value, set on first call.

Measured with a counting wrapper on `$PATH`, same 30-feature project:

| | git processes | wall clock |
|---|---|---|
| today | 116 | 0.56s |
| memoised | **56** | **0.30s** |

**Then the 32 `rev-parse`.** `resolveBranch` tries `refs/heads/<branch>` and then each
`refs/remotes/<remote>/<branch>`, so a claimed feature costs one or two processes and the
trunk lookup costs more. A single `git show-ref` lists every ref in the repository at once,
which answers all of them from one map. This is the larger change of the two and it is
worth measuring before committing to: if it does not beat 56 calls by a clear margin on the
same fixture, the box says so in `## Log` and the map does not land.

The counting wrapper is worth keeping in the repo as a make target, because "how many
subprocesses does a board read cost" is the question this feature answers and the next
regression will be invisible without it.

## Tasks
- [x] `HeadBranch()` and `remotes()` memoise per `Repo`, with a unit test that the second
      call runs no git
- [x] `make gitcount` runs a board read against a generated fixture and prints the
      subprocess count, so the number is checkable rather than remembered
- [x] `resolveBranch` answers from one `git show-ref` — or the box is deleted with the
      measurement that showed it was not worth it
- [ ] `docs/TECH.md` records the rule: an answer that cannot change inside one command is
      asked once

## Log
- 2026-08-17 — measured during a code audit, with a `git` wrapper on `$PATH` counting
  invocations.
- 2026-08-17 — 116 → 58 processes from memoising the two, then → **28** from one
  `git show-ref` behind `hasRef`. The remaining 28 are questions that genuinely differ per
  claim: each branch's copy of its doc, and each branch's tip date.
- 2026-08-17 — the unexpected half: `cmd/kit`'s suite went from 18.2s to **9.2s** at the
  same `-parallel 4`. Every conformance test drives the binary several times, so a board
  read costing a quarter of the processes shortens the gate as well. `make check` is now
  ~11s, from 35s before 0034.
