---
id: 0017
title: Read the board from real git state
status: In progress  # Draft | Accepted | In progress | Shipped | Dropped | Superseded
created: 2026-08-14
shipped:             # fill the date when status flips to Shipped
check:               # optional — date to check whether this worked. Only for bets.
checked:             # date you checked; write the outcome in ## Log
needs: 0016          # both edit audit.go's check registry and README's check list
related: docs/features/0010-make-the-board-honest-under-several-agents.md
---

## Problem

The claim mechanism is correct about the only thing it treats as atomic — the push is a
compare-and-swap and exactly one agent wins it. Everything the board says *around* that
push is read from local git state that can be wrong, and three of those readings break
the loop this kit exists to run: features are proposed and shipped through pull
requests, `main` is protected, and the branch is deleted when the PR merges.

**A deleted branch becomes a permanent claim nobody holds.** `Claims` reads
`git branch -r`, and git never prunes remote-tracking refs on `fetch` or `pull`. No kit
command fetches. So the moment a PR merges and the branch is deleted:

```
kit next   → taken. "nothing is free… Stopping here is the right outcome: say so and wait."
kit claim  → exit 1, "already claimed by 0001-first-feature"    (it exists nowhere)
kit audit  → "docs and repo agree"
```

Three answers, all wrong, and the middle one is phrased as a lost race — which the
protocol tells the agent is normal and final. The remedy the message prints fails twice:
`git push origin --delete` errors because the remote ref is already gone, and
`git branch -d` errors because there is no local branch. `git fetch --prune` is the fix
and appears nowhere. This fires on every feature of every project, so the autonomous
loop halts itself after the first merge.

**A claim can be won on a branch that does not contain the plan.** `ClaimFeature` pushes
`HEAD:refs/heads/<branch>` without checking that `HEAD` contains the doc it is claiming.
Against a repo created the normal way — GitHub initialises `main` with a README — the
scaffold and the drafts are untracked until somebody opens a bootstrap PR, and nothing
in `kit init`'s output says to:

```
kit claim 0001
  claimed  0001-do-the-thing
  pushed   origin/0001-do-the-thing — the push is the claim, and you won it
next:
  git worktree add ../proj-0001 0001-do-the-thing

$ ls ../proj-0001 && cd ../proj-0001 && kit next
README.md
  not set up yet — run:  kit init …/proj-0001
```

Every step exits 0. The agent is sent to a worktree containing nothing, and then told to
scaffold a second kit inside it.

**`stalled` cannot fire where anyone is looking.** It is the HIGH check for the failure
that matters most under autonomy — an agent died holding a feature — and it keys on
`status: In progress`. In a pull-request flow that status only ever exists on an
unmerged branch, so from the checkout a human or a supervising agent audits, the doc
reads `Accepted` and `kit audit` says "docs and repo agree". `LastMentioned` compounds
it by reading `git log` for HEAD alone, so commits on the feature branch are not
activity either. A dead agent and a busy one are indistinguishable, permanently, and the
symptom is the same as the first defect: "taken", with no liveness signal and no expiry.

## Non-goals

- **Making the kit fetch.** A read command that mutates refs is a read command that can
  fail, hang, or rewrite state behind an agent's back. The remote is asked one question
  and told nothing; `--offline` skips even that.
- **Enforcing that work happens in a worktree.** Rule 3 states it and git provides the
  isolation. This adds no policing.
- **Ageing a claim that carries no commits.** A branch created at trunk's tip is
  byte-identical to one whose PR merged, and neither carries a date the kit can read. A
  claim with no work is therefore reported by `next` as taken and never as stalled —
  guessing would invent findings.
- **Reserving feature IDs over the network.** Real, and a different mechanism from
  claiming. BACKLOG.md.

## Design

### Ask the remote, do not trust the mirror

`git ls-remote --heads` is one round trip, transfers no objects, and answers exactly the
question `Claims` is trying to answer: which branches exist for other agents to see.
Where a remote exists, that answer replaces `refs/remotes/*` entirely — those refs are
a cache of a past fetch and are the thing that goes stale. Local branches and worktree
branches are still claims: they are this machine's own state and cannot be stale.

```
Claims(id) = local branches ∪ worktree branches ∪ (ls-remote heads, if reachable)
```

Cached once per `Repo` and given a 5-second timeout, because `kit next ~/code/*` fans
out over every project and an unreachable remote must not hang the board. When the call
fails or `--offline` is passed, it falls back to `refs/remotes/*` as today — and only
then can a phantom claim happen, which is why that is the one case whose message names
`git fetch --prune`.

`--offline` is added to `next` and `audit` together. `claim` never takes it: a claim
decided against a cache is not a claim.

### A claim must carry the plan

`ClaimFeature` refuses unless the feature doc exists in the tree it is about to push:
`git cat-file -e HEAD:<doc>`. This runs before the push, so a refusal costs nothing and
leaves nothing behind.

It is an ordinary error and not a `NotYours` — nobody else has the feature, the agent
simply has uncommitted work that has to land first. The message says so, and says how,
because with `main` protected the answer is a pull request and not a commit:

```
kit claim: docs/features/0001-x.md is not committed on main, so the branch this would
push carries no plan. Open a PR that lands the plan on main first, then claim.
```

`kit init`'s closing instructions gain the same step, since that is where a bootstrap
run reads what to do next and where the omission started.

### Liveness comes from the ref, not from a status on another branch

Two changes, and together they make `stalled` mean what it says:

1. `loadGit` reads `git log --all`, so a commit citing a feature counts as activity
   wherever it was made. `LastTouched` likewise. Today the kit is blind to every commit
   on an unmerged branch, which is where all work happens.
2. `stalled` fires for a feature that **has a claim carrying work** and is not
   `Shipped`, `Dropped` or `Superseded` — instead of one whose local doc says
   `In progress`. The claim is the fact that survives branch isolation; the status does
   not. A claim carrying no work beyond the doc is skipped, per the non-goal above.

`no-claim` is unchanged and stays keyed on `In progress`: it fires inside the worktree
doing the work, which is where a missing branch is a real and fixable disagreement.

### Trunk is whatever the remote says it is

`isTrunk` knows only `main` and `master`, and `BranchBuildsOn` picks the first of those
that resolves. A project whose default branch is anything else has every branch read as
a claim. Trunk becomes: the branch `origin/HEAD` points at, then `main`, then `master`.
Needed here anyway — `BranchBuildsOn` decides whether a claim carries work, and the
`stalled` change above now depends on that answer.

## Tasks

- [ ] Trunk detection from `origin/HEAD` with the `main`/`master` fallback, used by both
      `isTrunk` and `BranchBuildsOn`; test a project whose default branch is neither.
- [ ] `Claims`/`OrphanClaims` read remote heads via a cached, timeout-bounded
      `ls-remote` and ignore `refs/remotes/*` when it succeeds; `--offline` on `next` and
      `audit` keeps the old path. Conformance: merge a claim branch, delete it on the
      remote, and assert `next` offers the feature and `claim` wins it.
- [ ] `ClaimFeature` refuses when the doc is not in `HEAD`'s tree, with the bootstrap-PR
      message; `kit init` says to land the plan on `main` first. Conformance: claiming an
      uncommitted plan exits non-zero and pushes nothing.
- [ ] `stalled` keys on a claim carrying work rather than local `In progress` status, and
      `loadGit`/`LastTouched` read `--all`. Conformance: a feature worked on an unmerged
      branch is not stalled from the trunk checkout, and an abandoned one is.

## Log

<!-- Dated, append-only conclusions. The argument lives in the PR. -->
