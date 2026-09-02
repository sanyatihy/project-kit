---
id: 0059
title: An agent is never misdirected or left waiting
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs:
---

## Problem

An agent does what the kit's last line of output told it to do. Three states break that,
and all three are on the path a new project walks.

`kit next` prints `kit claim <id>` for a feature whose plan has not reached the trunk, and
`kit claim` then exits 2 saying to land the round first. That is steps 9 and 10 of the
onboarding the protocol prescribes, so every new project stands there. `claim` learned the
trunk rule in 0027 and `next` did not.

Claiming from a worktree whose HEAD carries commits the trunk does not have pushes those
commits under the new feature's branch. Reproduced: in the worktree that had just shipped
0001, `kit claim 0002` exited 0 and the branch it created logged `0001: ship` and
`0001: add it` against the trunk. That pull request contains another feature's work, and
the kit instructed it.

`kit next --help` says the remote question is bounded at 15 seconds. Against a black-holed
remote it ran past 55. `exec.CommandContext` kills `git`, but the `git-remote-https`
grandchild holds the stdout pipe, so `cmd.Output()` blocks past the deadline. Agents run
unattended, so a hang costs more than a wrong answer.

## Non-goals

- Teaching `next` to refuse. The board reports; it does not gate. What it must not do
  is print a command that will be refused.
- Removing `--offline`. It stays, and stays explicit about what it cost.
- A second trunk check. `claim` already resolves the trunk; this is the same question
  asked of HEAD.

## Design

The board offers only what a claim would accept. `next` asks the question `claim`
already answers and reports the feature as blocked, naming the round that has to land,
rather than printing a command that exits 2. One condition, read by both.

A claim carries the trunk, not the checkout. `ClaimFeature` refuses when HEAD holds
commits absent from the trunk, which is the mirror of the check it already makes on the
doc, and its refusal already states the reason: only the branch's own author could merge
it. The fix line names the worktree to make, because that is what the agent was going to
be told next anyway.

A deadline that cannot be enforced is not a deadline. `cmd.WaitDelay` closes the pipes
the grandchild holds, so the bound the help text already promises becomes true. The
alternative — documenting the real behaviour — leaves an unattended agent stopped for ever
on a network the operator cannot see.

A feature appears once on the board. A shipped-but-unmerged feature is listed under
both `taken` and `awaiting review`; the second is the true one, because the work is
finished and the claim is only waiting for a merge.

## Tasks

- [x] `next` reports a feature whose plan has not landed as blocked, naming the round
- [x] `claim` refuses when HEAD carries commits the trunk does not have
- [x] The remote question comes back inside its stated bound, and a test drives a hang
- [x] A shipped-but-unmerged feature appears once on the board

## Log

- The blocked entry alone was not enough. A board where the round has not landed has
  nothing free, and the closing line then told the agent to wait for work only it could
  unblock. Landing the round is the reader's own next action, so it gets an arm of its own.
- `cmd.WaitDelay` is set at every call site with a deadline, not only the one measured.
  `claim` pushes under the same `exec.CommandContext`, so its deadline was equally
  unenforceable.
