---
id: 0033
title: Show how far a claimed feature has got
status: Shipped
created: 2026-08-17
shipped: 2026-08-17
needs: 0032
related: 0028-see-what-is-waiting-for-review.md
---

## Problem
The board counts boxes from the doc in **your** checkout, and the ticks live on the claim
branch until it merges. So while an agent works, the operator's board reads `0/3` however
far along it is:

```
  taken
    0002  0/3   Resolve a short link — 0002-resolve-a-short-link
```

Measured on a fixture with one box ticked: the trunk copy has 0 ticks, and
`origin/0002-resolve-a-short-link` has 1. The number is not stale, it is the wrong copy —
and it reads as "an agent that has done nothing", which is what `stalled` means.

The operator's only way to see progress is to open the pull request or run `git log` on
the branch, which is the worktree-reading the board exists to save. 0028 hit exactly this
for `status:` and fixed it by reading the branch; the counter was left behind.

## Non-goals
- **Estimating anything.** Boxes are not equal sizes. This reports what has been ticked
  and nothing more, which is the rule the counter already states.
- **Fetching.** Same bargain as everything else: one `ls-remote`, no objects, no writes.
- **A second command.** It is the same board; it just stops reading the wrong file.

## Design
`Survey` already resolves each feature's claim branch for the taken group. Where one
exists, count the boxes from that branch's copy of the doc instead of the checkout's —
`git show <ref>:<rel>`, parsed the way `statusOn` already parses status.

Unreadable answers the checkout: a branch whose objects were never fetched, or one that
does not carry the doc, keeps today's number rather than reporting nothing. `--offline`
and an unanswered remote already say the board is what the last fetch knew, so a stale
count is covered by a notice that is already there.

The same read gives the entry a second fact worth showing — **the last commit on the
branch** — so a claim that has not moved in a day is visible as such before `stalled`
fires at three days:

```
  taken
    0002  1/3   Resolve a short link — 0002-resolve-a-short-link, 20 minutes ago
```

## Tasks
- [x] Box counts for a claimed feature come from its claim branch, falling back to the
      checkout when the branch cannot be read; conformance for both
- [x] Each taken entry says when its branch last moved, in text and `--json`
- [x] README and `kit help next` say the counter follows the claim

## Log
- 2026-08-17 — the board reported 0/3 for two agents that were both mid-feature.

