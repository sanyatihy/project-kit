---
id: 0027
title: Refuse a claim on a plan that has not landed
status: Shipped
created: 2026-08-16
shipped: 2026-08-16
check:
checked:
needs:
related: 0021-let-agents-work-from-drafts-and-make-accepting-a-signal.md
---

## Problem
The protocol says *"The plan must reach the default branch before anything can be
claimed"*, and `TrackedAtHEAD` checks the doc is in **HEAD's** tree — which on a plan
branch it is. So claiming from an unmerged plan branch works:

```
$ git ls-tree origin/main docs/features/ | grep -c 0001    0
$ kit claim 0001    claimed  0001-planned      (exit 0)
```

The consequence is a broken review, not a bad branch: every claim taken this way is a
child of the plan branch, so the only way to integrate it is a merge into the branch its
own author coordinated. It happened here — four features built this way, three merge
commits under the builder's name, nothing pushed for review.

## Non-goals
- **Checking the plan is merged on the *remote* trunk.** The local trunk ref is what
  `Trunk()` and `trunkRef()` already resolve, and requiring a network call would make
  `claim` fail offline for a reason that has nothing to do with claiming.
- **Refusing when there is no trunk at all.** A fresh project with one branch and no
  remote is normal, and `claim` already says the claim is invisible to everyone.
- **Blocking `new` or `next`.** Drafting and reading the board off a plan branch are
  correct; it is taking work that must wait.

## Design
`ClaimFeature` also requires the doc in `trunkRef()`'s tree, which widens from the local
trunk to each remote-tracking one. An ordinary error at exit 2, not `NotYours`: nothing on
an unlanded round is claimable, so an agent that takes the next id hits the same wall.

`TrackedAtHEAD` stays for the case the trunk check does not cover — a doc on the trunk but
edited and uncommitted here. Where no trunk resolves the check is skipped, or the first
claim of every new project would be refused.

## Tasks
- [x] `ClaimFeature` requires the doc on `trunkRef()`, exit 2, naming the round to land
- [x] The no-trunk case is skipped, with a conformance case claiming in a fresh
      single-branch project
- [x] Conformance: claiming from an unmerged plan branch is refused; claiming the same
      feature after the round merges succeeds
- [x] `kit help claim` and README state the barrier and why it exists

## Log
- 2026-08-16 — the ordering rule was documented and only half enforced: four features
  reached the trunk merged by their own builder.
- 2026-08-16 — a fresh single-branch project turns out never to reach the skip:
  `Trunk()` falls back to the branch the checkout is on, so the doc is on the trunk
  and the check passes outright. The skip covers the states with no trunk at all — a
  detached HEAD, or a checkout whose only branch is claim-shaped.
