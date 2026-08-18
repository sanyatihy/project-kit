---
id: 0040
title: Stop reading one branch as two claims
status: Shipped
created: 2026-08-18
shipped: 2026-08-18
check:
checked:
review:
needs:
related: docs/features/0010-make-the-board-honest-under-several-agents.md
---

## Problem

A branch whose name has a slash in it is counted twice, so one agent's worktree reads as two
agents holding the same feature. On a fresh clone, with `agents/0039-work` the only branch and
the only worktree:

```
0039 Cut what a fleet board read costs: claimed by 0039-work and agents/0039-work
fix: one agent per feature — delete the branch that is not doing the work
```

`split-claim` at MEDIUM on healthy state, and a fix line that deletes the work if it is
followed. The worktree parse keeps the last path segment of `branch refs/heads/agents/0039-work`
while `git branch --format=%(refname:short)` keeps the whole name, so `seen` holds both. Found
while drafting 0039 on a plan branch that happened to name it.

## Non-goals

- **Not a change to what counts as a claim.** A branch citing an id is a claim wherever the id
  sits in the name — that is deliberate and 0010 settled it. This makes one branch count once.

## Design

**The worktree's branch is read the way every other ref is: strip `refs/heads/`, keep the
rest.** Splitting on `/` and taking the last segment was only ever right for unnamespaced
branches, which is what the kit's own claims are, so nothing noticed. `branchName` already
strips a remote prefix and nothing else; the worktree path is the one place that invented its
own rule.

**The convention says *mentions*, not *named after*.** `docs/TECH.md` tells drafters a plan
branch must never be "named after a feature, since a leading four-digit ID is a claim", which
reads as a rule about the first segment. A plan branch called `plan/fold-the-misfire-into-0039`
is a claim on 0039 by the same rule, and it was written by an agent that had read the sentence.
One word, at the tier where prose is all there is.

## Tasks
- [x] The worktree branch parsed by stripping `refs/heads/`, with conformance that a namespaced
      branch in a worktree is one claim and that two real branches still report `split-claim`
- [x] `docs/TECH.md` and the plan-branch guidance say a plan branch must not *mention* a
      feature id anywhere in its name

## Log
- 2026-08-18 — the plan branch drafting 0039 was named after it, and `kit audit` reported
  the checkout as fighting itself.
- 2026-08-18 — the protocol text changed, so every project carrying the old wording reports
  `protocol-stale` until `kit init` refreshes it. This repo's own AGENTS.md included.
