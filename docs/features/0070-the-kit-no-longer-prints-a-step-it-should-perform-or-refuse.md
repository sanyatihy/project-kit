---
id: 0070
title: The kit no longer prints a step it should perform or refuse
status: Shipped
created: 2026-08-29
shipped: 2026-08-31
needs: 0069
---

## Problem

A printed command ranks with the refusal beside it, and a printed `git push --delete` has
already handed a loser the winner's branch. Six leftovers of that class: `claim` prints
`git worktree add` instead of making the worktree, and its refusals name `../<repo>-trunk`;
a missing `status:` is refused with a hand-edit; `ship` never says the trunk already holds
the claim; `docs/features/TEMPLATE.md` pre-answers `needs:`; protocol rules 9 and 10
restate refusals `claim` already makes; a doc with no `id:` is invisible to `id-mismatch`.

## Non-goals

- Preventing a self-merge. Branch protection is the only tier an agent cannot route around.
- Creating a trunk worktree on a refused claim. A refused claim must not touch the disk.
- An authorship check. Agents share the operator's identity here, so it fires on every merge.
- The single-operator plan-and-build path. A protocol fork rather than a printed command.
- Making the worktree optional. After `claim` the current checkout is still the trunk.
- A second code for a missing `id:`. Same guarantee as `id-mismatch`, same finding.

## Design

A won claim creates `.worktrees/<branch>`. The add runs after the push and a failed add
does not unclaim — the split 0063 gave the exclude. A path already holding the claim branch
is success; anything else there is a warning beside the claim. `release` removes it, or the
branch could never be deleted afterwards.

Every checkout the kit names lives under `.worktrees/`, so the two refusals that send an
agent to the trunk name `.worktrees/<trunk>`, detached — a trunk checkout is not a claim
and has no feature branch to be named for.

`claim`'s no-status refusal names `kit reset`, which is the command that owns the field.

`ship` reports containment, not authorship: the trunk holding the claim branch before the
stamp is the merge-early case, and it is readable without asking who pushed.

Rules 9 and 10 go, restating refusals `claim` and `new` already make. The worktree sentence
that remains is a command's output rather than a rule.

`docs/features/TEMPLATE.md` is deleted. `kit help template` prints the embedded copy and
`kit init` never writes this file, so the placed copy only drifts.

`id-mismatch` covers a missing `id:` at HIGH — the id every command takes is not a key that
finds the file, the same defect as two that disagree. `no-status` then names the file,
because `kit reset  "<why>"` is untypeable.

`needs: 0069` because both rewrite `templates/AGENTS-section.md`. The code could start
today; the protocol file cannot.

## Tasks

- [x] `kit claim` creates `.worktrees/<branch>` after a won push, and a failed add does not
      unclaim
- [x] The two trunk refusals name `.worktrees/<trunk>`
- [x] `claim`'s missing-`status:` refusal names `kit reset`
- [x] `kit ship` says when the trunk already contains the claim branch
- [x] `kit help`, the protocol and `docs/TECH.md` follow, and rules 9 and 10 are gone
- [x] `docs/features/TEMPLATE.md` is gone
- [x] `id-mismatch` reports a missing `id:` at HIGH, and `no-status`'s fix names the file

## Open questions

- Does merge-after-ship belong in `next` as well as `ship`? Leaning: no. `AwaitingReview`
  already drops a branch the trunk contains, and the protectable half is the forge.

## Log

- 2026-08-30 — three leftovers joined the round: delete `docs/features/TEMPLATE.md`, drop
  rules 9 and 10, extend `id-mismatch` to a missing `id:`.
- 2026-08-31 — `release` removes the worktree, inside the box that made `claim` create one.
  Otherwise every release finds the branch checked out and leaves it behind.
