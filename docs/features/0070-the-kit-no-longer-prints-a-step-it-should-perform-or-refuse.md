---
id: 0070
title: The kit no longer prints a step it should perform or refuse
status: Draft
created: 2026-08-29
shipped:
needs: 0069
---

## Problem

TECH.md ranks a printed command with the refusal it sits next to, and a printed
`git push --delete` has already handed a loser the winner's branch. Three steps are
still that tier: `claim` prints `git worktree add` instead of creating the isolation
unit, and its refusals still name `../<repo>-trunk`; a missing `status:` is refused
with a hand-edit of the field 0067 exists to own; `ship` says merging is not yours
and does not say when the trunk already contains the claim.

## Non-goals

- **Preventing a self-merge.** Branch protection is the only tier an agent cannot route
  around. This reports. An authorship check would fire on every merge while agents share
  the operator's identity, which is this repo.
- **Creating a trunk worktree on a refused claim.** A lost or refused claim must not
  mutate the disk. The path is named; the add stays with the agent that has to stand
  there.
- **A new audit check.** `needs-unstated` already spent "a finding is always a problem".
  A check that cannot tell a coordinator merge from a builder merge would spend it again.
- **The single-operator plan-and-build path.** That is a protocol fork, not a printed
  command. It stays in BACKLOG.md until that question is answered.
- **Making the worktree optional.** 0063 permitted a clone that is already the feature's
  checkout. After `claim` the current checkout is still the trunk, so the worktree is
  always the place to stand.

## Design

**A won claim creates `.worktrees/<branch>`.** The push remains the claim: the add runs
after it, and a failed add does not unclaim — the same split 0063 gave the exclude. The
output names the path to work in, not a git command to copy. If that path is already a
worktree of the claim branch, that is success. Anything else at the path is a warning
beside the claim.

**Every checkout the kit names lives under `.worktrees/`.** The two refusals that still
print `git worktree add --detach ../<repo>-trunk` name
`<main worktree>/.worktrees/<trunk>` instead, detached, because a trunk checkout is not a
claim and has no feature branch to be named for. One directory, two kinds of checkout.

**`claim`'s no-status refusal names `kit reset`.** 0067 covered `no-status` and `next`.
The leftover is the `Then` on a missing `status:` in `ClaimFeature`, which is the
instruction a chain will follow when it hits `claim` first.

**`ship` reports when the trunk already contains the claim branch.** Containment, not
authorship: the trunk having the work before the stamp is the merge-early case, and it
is readable without asking who pushed. Merge-after-ship stays a prose line and a
protected trunk. `AwaitingReview` already going quiet is the board's half of that; this
is `ship`'s.

**`needs: 0069` because both rewrite `templates/AGENTS-section.md`.** This round changes
the worktree sentence; 0069 changes the feature definition. The code could start today.
The protocol file cannot.

## Tasks

- [ ] `kit claim` creates `.worktrees/<branch>` after a won push, prints that path, and
      does not unclaim if the add fails — with a conformance case that the success
      output contains no `git worktree add`
- [ ] the two `ClaimFeature` refusals that send an agent to the trunk name
      `.worktrees/<trunk>`, not `../<repo>-trunk`
- [ ] `claim`'s missing-`status:` refusal names `kit reset`, not a hand-edit
- [ ] `kit ship` says when the trunk already contains the claim branch
- [ ] `kit help`, the protocol's worktree sentence, and `docs/TECH.md` say claim
      creates the worktree, and this project's `AGENTS.md` is regenerated in the same
      commit as the template

## Open questions

- **Does merge-after-ship belong in `next` as well as `ship`?** `ship` cannot see a merge
  that happens after it returns. Leaning: no. `AwaitingReview` already drops a branch the
  trunk contains, and an authorship line would be wrong on this repo. The protectable
  half is the forge.

## Log

- 2026-08-29 — dropped the template's `needs:` HTML comment. It is scaffolding `kit new`
  writes; a filled doc does not keep it. 0065 moved that guidance off the field, and 0069
  was filled without the copy.
