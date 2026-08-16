---
id: 0027
title: Refuse a claim on a plan that has not landed
status: In progress
created: 2026-08-16
shipped:
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

The consequence is not a bad branch, it is a broken review. Every claim taken this way
is a child of the plan branch rather than a sibling off the trunk, so the only way to
integrate any of it is to merge into that branch — and whoever does that is merging
work they coordinated. It happened here: four features were built this way, three merge
commits landed under the builder's name, and no branch was ever pushed for review.

Rule 13 was violated downstream of this, not independently of it. The tool enforced the
weaker half of a rule whose whole purpose is to keep implementation branches
independent.

## Non-goals
- **Checking the plan is merged on the *remote* trunk.** The local trunk ref is what
  `Trunk()` and `trunkRef()` already resolve, and requiring a network call would make
  `claim` fail offline for a reason that has nothing to do with claiming.
- **Refusing when there is no trunk at all.** A fresh project with one branch and no
  remote is normal, and `claim` already says the claim is invisible to everyone.
- **Blocking `new` or `next`.** Drafting and reading the board off a plan branch are
  correct; it is taking work that must wait.

## Design
`ClaimFeature` replaces `TrackedAtHEAD(target.Rel)` with the same test against the
trunk: the doc must exist in `trunkRef()`'s tree. `trunkRef` already widens from the
local trunk to each remote-tracking one, so a fresh clone with only `origin/main`
resolves.

Refused as an ordinary error, exit 2, not `NotYours`. This is not "somebody else has
it, take another" — nothing is claimable until the plan lands, so an agent that moves
on to the next id hits the same wall. The message says to land the round first.

When no trunk resolves at all, the check is skipped and the existing behaviour stands:
a project with one branch and no remote has no trunk to be absent from, and refusing
there would break the first claim of every new project.

`TrackedAtHEAD` stays for what it was for — a doc uncommitted in the working tree does
not travel with the pushed branch — because the trunk check does not cover it: a doc
can be on the trunk and edited-but-uncommitted here.

## Tasks
- [x] `ClaimFeature` requires the doc on `trunkRef()`, exit 2, naming the round to land
- [x] The no-trunk case is skipped, with a conformance case claiming in a fresh
      single-branch project
- [x] Conformance: claiming from an unmerged plan branch is refused; claiming the same
      feature after the round merges succeeds
- [ ] `kit help claim` and README state the barrier and why it exists

## Log
- 2026-08-16 — found while reviewing how four features came to be merged by their own
  builder: the ordering rule was documented and only half enforced.
- 2026-08-16 — a fresh single-branch project turns out never to reach the skip:
  `Trunk()` falls back to the branch the checkout is on, so the doc is on the trunk
  and the check passes outright. The skip covers the states with no trunk at all — a
  detached HEAD, or a checkout whose only branch is claim-shaped.
