---
id: 0017
title: Read the board from real git state
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
needs: 0016
related: 0010-make-the-board-honest-under-several-agents.md
---

## Problem
The push is atomic and correct. Everything the board says *around* it is read from local
state that can be wrong, and three readings break the pull-request loop the kit exists to
run.

**A deleted branch becomes a permanent claim nobody holds.** `Claims` reads `git branch -r`,
git prunes those refs on neither `fetch` nor `pull`, and no kit command fetches. The moment
a PR merges and its branch is deleted, `next` says taken and stop, `claim` reports a lost
race against a branch that exists nowhere, and `audit` says the docs and repo agree — three
wrong answers, the middle one phrased as final. The printed remedy fails twice:
`push --delete` and `branch -d` both error, and `git fetch --prune` appears nowhere. This
fires on every feature of every project, so the loop halts after the first merge.

**A claim can be won on a branch that does not contain the plan.** `ClaimFeature` pushes
`HEAD:refs/heads/<branch>` without checking `HEAD` carries the doc. Against a repo whose
`main` starts with a README, the drafts are untracked, every step exits 0, and the agent is
sent to a worktree containing nothing and then told to scaffold a second kit inside it.

**`stalled` cannot fire where anyone is looking.** It keys on `status: In progress`, which
in a PR flow only ever exists on an unmerged branch — so from the trunk checkout the doc
reads `Accepted` and audit agrees. `LastMentioned` compounds it by reading HEAD's log alone,
so commits on the feature branch are not activity either. A dead agent and a busy one are
indistinguishable.

## Non-goals
- **Making the kit fetch.** A read command that mutates refs can fail, hang, or rewrite state behind an agent's back.
- **Enforcing that work happens in a worktree.** Rule 3 states it and git provides the isolation.
- **Ageing a claim that carries no commits.** A branch at trunk's tip is identical to one whose PR merged; guessing would invent findings.
- **Reserving IDs over the network.** A different mechanism from claiming — BACKLOG.

## Design
**Ask the remote, do not trust the mirror.** `git ls-remote --heads` is one round trip,
transfers no objects, and answers exactly what `Claims` needs. Where a remote exists it
replaces `refs/remotes/*` entirely; local and worktree branches still count, being this
machine's own state.

Cached per `Repo` and timeout-bounded, because `kit next ~/code/*` fans out over every
project. On failure or `--offline` it falls back to the mirror — the one case where a
phantom claim is possible, which is why that message names `git fetch --prune`. `claim`
never takes `--offline`: a claim decided against a cache is not a claim.

**A claim must carry the plan.** `git cat-file -e HEAD:<doc>` before the push, so a refusal
costs nothing. An ordinary error rather than `NotYours` — nobody else has it, the work
simply has to land first, and with `main` protected that means a pull request.

**Liveness comes from the ref, not a status on another branch.** `loadGit` and `LastTouched`
read `--all`, so a commit citing a feature counts wherever it was made; `stalled` fires for a
feature whose claim carries work and is not terminal. The claim survives branch isolation;
the status does not.

**Trunk is whatever the remote says it is** — `origin/HEAD`, then `main`, then `master`.
A project whose default is neither had every branch read as a claim.

## Tasks
- [x] Trunk detection from `origin/HEAD` with the `main`/`master` fallback, used by both
      `isTrunk` and `BranchBuildsOn`; test a project whose default branch is neither.
- [x] `Claims`/`OrphanClaims` read remote heads via a cached, timeout-bounded
      `ls-remote` and ignore `refs/remotes/*` when it succeeds; `--offline` on `next` and
      `audit` keeps the old path. Conformance: merge a claim branch, delete it on the
      remote, and assert `next` offers the feature and `claim` wins it.
- [x] `ClaimFeature` refuses when the doc is not in `HEAD`'s tree, with the bootstrap-PR
      message; `kit init` says to land the plan on `main` first. Conformance: claiming an
      uncommitted plan exits non-zero and pushes nothing.
- [x] `stalled` keys on a claim carrying work rather than local `In progress` status, and
      `loadGit`/`LastTouched` read `--all`. Conformance: a feature worked on an unmerged
      branch is not stalled from the trunk checkout, and an abandoned one is.

## Log
- 2026-08-14 — dropped the doc's mtime from the `stalled` signals: for a tracked file it
  records when git last wrote it out, so a merge resets it to now and a freshly merged draft
  masked a claim dead for a week. `Feature.Modified` went with it, its only reader.
- 2026-08-14 — trunk needed a fourth fallback: a project with no remote and no `main`/`master`
  still has a default branch, taken from HEAD but only when HEAD is not claim-shaped.
- 2026-08-14 — rejected detecting the deleted branch locally by testing ancestry against the
  trunk: exact for merge commits, wrong for squash merges, where the tip is an ancestor of
  nothing. The remote answers both.
