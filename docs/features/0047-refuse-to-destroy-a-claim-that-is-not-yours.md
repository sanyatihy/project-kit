---
id: 0047
title: Refuse to destroy a claim that is not yours
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs:
---

## Problem

Two reproduced paths lead from a healthy board to one feature held by two agents, or to a
deleted branch carrying work. Both are reachable by following instructions the kit itself
prints, and the test written to catch the second one passes by accident.

## Non-goals

- A permission model. Agents share one credential, so nothing here tells identities
  apart. The unit of ownership stays the checkout, which is what `Repo.Mine` already reads.
- Recovering a branch already deleted. That is the reflog's job; this stops the delete.
- A new audit finding. `split-claim` already reports two branches on one feature. A
  refusal in the command is the tier above a report, and the tier this belongs in.

## Design

`Release` requires the claim to be this checkout's, exactly as `Ship` does. Both
commands act on somebody's work, and `Repo.Mine` is the one answer to "is this yours".
`Ship` calls it, `Release` does not, and that asymmetry is the defect. A foreign claim
stays releasable behind `--force`, because clearing up after a dead agent is real work —
the flag is what makes destroying another agent's claim deliberate rather than incidental.

An unknown trunk refuses instead of permitting. `BranchBuildsOn` answers "no work"
when `trunkRef()` is empty, and "no work" is the answer that deletes. The ladder gains
`refs/remotes/<remote>/{main,master}`, and `Release` refuses outright when no trunk
resolves. In the one command that destroys, silence must not be the permissive answer.

The claim refusal names `kit release <id>`, never `git push --delete`. The command
carries the work guard and now the ownership check; the raw git line carries neither, and
a losing agent reading it is one step from taking the winner's branch. A hedging sentence
does not stop a printed command from being run.

**The conformance harness gives the binary the same fixed git environment it gives
itself.** `gitEnv()` exists for this and is applied to `p.git` but not `p.run`, so every
git call the binary makes inherits the developer's `~/.gitconfig`.

Reproduced against the built binary:

| path | result |
| --- | --- |
| loser follows the printed `git push --delete`, then claims | two checkouts both told the feature is theirs |
| `kit release` where the default branch is `trunk`, branch carrying a commit | remote branch deleted, exit 0, no `--force` |
| `GIT_CONFIG_GLOBAL` setting `init.defaultBranch = trunk` | three conformance tests fail |

## Tasks

- [x] `Release` refuses a claim this checkout does not hold; `--force` is the way past
- [x] `findTrunk` consults the remote-tracking trunk, and `Release` refuses when none resolves
- [x] The taken-feature refusal in `claim` names `kit release`, and prints no raw git delete
- [x] `project.run` uses `gitEnv()`, and the suite passes under a foreign `init.defaultBranch`
- [x] Tests drive a foreign checkout releasing a claim, and a release on a non-`main` trunk

## Log
- 2026-08-21 — the ownership test is a local head, not `Repo.Mine`. `Mine` asks whether HEAD
  is the claim branch, so an agent that had just claimed a feature and not yet moved to the
  worktree could not release what it had taken. A local ref answers "this clone created the
  branch", which holds for every worktree sharing it.
