---
id: 0070
title: The kit no longer prints a step it should perform or refuse
status: Shipped
created: 2026-08-29
shipped: 2026-08-31
needs: 0069
---

## Problem

TECH.md ranks a printed command with the refusal it sits next to, and a printed
`git push --delete` has already handed a loser the winner's branch. Three steps are
still that tier: `claim` prints `git worktree add` instead of creating the isolation
unit, and its refusals still name `../<repo>-trunk`; a missing `status:` is refused
with a hand-edit of the field 0067 exists to own; `ship` says merging is not yours
and does not say when the trunk already contains the claim. The same class of leftover
sits in three other places: `docs/features/TEMPLATE.md` still pre-answers `needs:`,
protocol rules 9 and 10 restate refusals `claim` already makes, and a doc with no
`id:` is invisible to `id-mismatch`.

## Non-goals

- **Preventing a self-merge.** Branch protection is the only tier an agent cannot route
  around. This reports. An authorship check would fire on every merge while agents share
  the operator's identity, which is this repo.
- **Creating a trunk worktree on a refused claim.** A lost or refused claim must not
  mutate the disk. The path is named; the add stays with the agent that has to stand
  there.
- **An authorship check, or another check that cannot tell a dependency from a
  comparison.** `needs-unstated` already spent "a finding is always a problem". A
  missing `id:` is not that: both halves are absent or they agree, and there is
  nothing to guess.
- **The single-operator plan-and-build path.** That is a protocol fork, not a printed
  command. It stays in BACKLOG.md until that question is answered.
- **Making the worktree optional.** 0063 permitted a clone that is already the feature's
  checkout. After `claim` the current checkout is still the trunk, so the worktree is
  always the place to stand.
- **A second code for a missing `id:`.** Same guarantee as `id-mismatch`, same finding.

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

**Rules 9 and 10 go.** They restate refusals `claim` and `new` already make. The
worktree sentence that remains is the output of a won claim, not a rule; it lives in
`kit help` and the command. The rest of the list is renumbered. Citations stay by name.

**`docs/features/TEMPLATE.md` is deleted.** `kit help template` prints the embedded
copy, `kit init` must not write this file, and `notAFeature` already keeps the name off
the board. The copy in the features directory still carries the `needs:` sentence 0068
removed, so a hand-copy is the bug that round closed.

**`id-mismatch` reports a missing `id:`.** The filename prefix with no frontmatter id is
the same defect as the two disagreeing: the id the commands take is not a key that
finds this file. HIGH, as it already is. `no-status`'s fix names the file when `id:` is
empty, because `kit reset  "<why>"` is a command nobody can type.

**`needs: 0069` because both rewrite `templates/AGENTS-section.md`.** This round changes
the worktree sentence and drops two rules; 0069 changes the feature definition. The
code could start today. The protocol file cannot.

## Tasks

- [x] `kit claim` creates `.worktrees/<branch>` after a won push, prints that path, and
      does not unclaim if the add fails — with a conformance case that the success
      output contains no `git worktree add`
- [x] the two `ClaimFeature` refusals that send an agent to the trunk name
      `.worktrees/<trunk>`, not `../<repo>-trunk`
- [x] `claim`'s missing-`status:` refusal names `kit reset`, not a hand-edit
- [x] `kit ship` says when the trunk already contains the claim branch
- [x] `kit help`, the protocol, and `docs/TECH.md` say claim creates the worktree;
      rules 9 and 10 are gone; this project's `AGENTS.md` is regenerated in the same
      commit as the template
- [x] `docs/features/TEMPLATE.md` is gone
- [x] `id-mismatch` reports a missing `id:` at HIGH, with a conformance case, and
      `no-status`'s fix names the file when `id:` is empty

## Open questions

- **Does merge-after-ship belong in `next` as well as `ship`?** `ship` cannot see a merge
  that happens after it returns. Leaning: no. `AwaitingReview` already drops a branch the
  trunk contains, and an authorship line would be wrong on this repo. The protectable
  half is the forge.

## Log

- 2026-08-29 — dropped the template's `needs:` HTML comment. It is scaffolding `kit new`
  writes; a filled doc does not keep it. 0065 moved that guidance off the field, and 0069
  was filled without the copy.
- 2026-08-30 — three leftovers join the round: delete `docs/features/TEMPLATE.md`, drop
  protocol rules 9 and 10, and extend `id-mismatch` to a missing `id:`. The non-goal on
  new checks narrows to authorship and comparison-vs-dependency; a missing id is neither.
- 2026-08-31 — `kit release` removes the worktree inside the box that made `claim` create
  one. Every release would otherwise find the branch checked out and leave it behind, which
  is the leftover this round exists to stop. A worktree an operator put elsewhere is still
  theirs and is reported, not removed.
