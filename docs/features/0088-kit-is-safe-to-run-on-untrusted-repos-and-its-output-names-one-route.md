---
id: 0088
title: Kit is safe to run on untrusted repos and its output names one route
status: Draft
created: 2026-10-29
submitted:
needs:
---

## Problem

Before going public, an audit found that a hostile doc, branch or remote could make kit
print a shell payload as the next command, truncate a tracked file, or write outside the
repo. It also found that text output taught
retired routes while the JSON taught `start`/`finish`, and that anna's round paid a reopen
for every trunk merge.

## Non-goals

- A LICENSE. The operator chooses it.
- `kit audit` in generated CI. It changes every adopter; BACKLOG.
- Batched per-claim reads, the comment trim, fewer spike commits. Each is its own round.

## Design

- **Ids.** An id is `^\d{4,}$`, with no upper bound. Anything else is read as no id:
  HIGH `id-mismatch`, blocked on the board, never printed in a command. Every id regex
  widens to four or more digits. Cost: a long number in a branch name now reads as a
  claim, as four-digit runs already did.
- **Untrusted text.** Titles and harvest boxes are one line; control characters are
  stripped from titles and clipped text before printing.
- **Refs.** A branch or trunk name beginning with `-` is dropped where it is read
  (`safeRef`). Rejected: `--end-of-options` at each call site, which needs git 2.24 and
  every future caller to remember it.
- **Writes.** Docs and AGENTS.md are replaced by rename after an `Lstat` that refuses a
  symlink; a `.worktrees` that is not a directory gets no worktree.
- **Gate proof.** `finish` writes `Kit-Gate: <commit>` into its Submit commit; a skip
  carries it forward and only it permits one, after a reopen. Rejected: trusting the
  status line, which a bare `kit submit` writes without a gate.
- **Timeouts.** Commit, push and worktree creation get 10 minutes for hooks; other reads
  keep 30 s. `GIT_TERMINAL_PROMPT=0` on every git. The reservation fetch disables gc and
  maintenance through `-c`, which any git version accepts.
- **One route.** Every printed step comes from the same choice as `NextAction`:
  `kit start <id>`, `kit finish`, `kit start --plan` on an empty board, and `kit harvest`
  only for a spike `kit start --work` did not record (`spike_recorded`).
- **Trunk merges.** After Submit, commits that are only merges of the trunk need no
  reopen; finish reruns the gate and keeps the status.
- **Noise.** `question-stale` skips Parked plans.
- **Speed.** `audit` starts the reservation fetch beside the branch listing: 4.7 s → ~2.5 s.

## Tasks

- [x] Ids, titles and branch names from a repo cannot become commands, options or terminal escapes
- [x] No write goes through a symlink or adds .env, the gate is skipped only on Kit-Gate proof, and hooks get ten minutes
- [x] Every printed next step names start or finish, and help, exit codes and rule 14 match the code
- [x] Parked questions stay quiet, trunk merges need no reopen, and audit fetches reservations concurrently
- [x] README and TECH describe a public module, comments name their rule, and the backlog is current

## Log
