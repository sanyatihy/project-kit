---
id: 0028
title: See what is waiting for review
status: Draft
created: 2026-08-16
shipped:
check:
checked:
needs: 0027
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem
`kit next` answers *what do I build*. Nothing answers *what needs review*, so the far
side of `ship` is unobserved: a shipped feature leaves the board and the kit never
mentions it again.

Both failures this session live in that blind spot. Four features were shipped and
never pushed, with `audit --strict` green throughout — rule 13 says push and nothing
checks it. And in `observer`, five drafts sat on unmerged branches while the board
reported *"nothing is planned"*. Same gap, reached from opposite ends: work that has
left the builder and not arrived anywhere.

There is also no role for a coordinator to hold. Rule 13 says *"reviewing and merging
are not yours"*, which reads as human-only — but the invariant it protects is that
**the builder does not approve its own work**. A second agent can hold that role; a
human never being involved is a separate decision.

## Non-goals
- **A new status.** `Shipped` already means the builder is finished, which is the
  builder's own statement about its own work. Whether it landed is not its call and
  git already answers it — a `Reviewed` status would be a field an agent could set
  about a decision it does not own.
- **Merging, or reading approvals.** No forge API, unchanged from 0018 and 0020. The
  kit reports what is waiting; a coordinator merges with its own tools. This is what
  keeps it identical on GitHub and on a self-hosted GitLab.
- **Judging whether work is any good.** `audit` reports what is checkable.
- **Assigning reviewers.** The kit does not know who exists.

## Design
Three git facts already available decide the state, and no new one is stored:

| `status:` | branch on remote | on trunk | state |
|---|---|---|---|
| In progress | yes | no | being built |
| Shipped | no | no | **shipped, never pushed** |
| Shipped | yes | no | **awaiting review** |
| Shipped | — | yes | landed — nothing to say |

**`kit next` gains an `awaiting review` section**, after the board and before the
guidance. It lists the feature, its branch, and how long it has waited. It is the
coordinator's board and it is the same command everyone else runs, so there is one
place to look rather than a second tool to remember.

The closing guidance accounts for it, per 0013: a project with nothing free and three
features awaiting review must not say *"stop"* — the work to do is reviewing.

**`shipped-not-pushed`**, HIGH. A feature whose doc says Shipped, whose branch is not
on the remote, and whose work is not on the trunk exists only on one machine. HIGH
because it is silent, and because the fix is one push — the same reasoning that makes
`stalled` HIGH.

Both read `remoteHeads`, which `Claims` already calls, and both degrade the way the
rest of the kit does: with `--offline`, or a remote that cannot be reached, neither
reports rather than guessing. A claim read from a stale mirror is already handled by
`RefsAreAuthoritative`, and this uses the same answer.

**Rule 13 is reworded** from "reviewing and merging are not yours" to the invariant it
means: *you do not merge your own work; a different checkout — another agent, or a
human — reviews it.* The protocol gains the three roles and the barrier that separates
them.

## Tasks
- [ ] `Repo.AwaitingReview()` — Shipped, branch on the remote, not reachable from the
      trunk — reusing `remoteHeads` and `trunkRef`
- [ ] `kit next` prints an `awaiting review` section, in text and `--json`
- [ ] Closing guidance names reviewing as the work when that is all there is
- [ ] `shipped-not-pushed` at HIGH, quiet offline and quiet when the work is on the
      trunk; conformance for each
- [ ] Rule 13 reworded around separation of duties, with the three roles and the
      landing barrier, in `templates/AGENTS-section.md`
- [ ] README and `kit help next` describe the review board and who reads it

## Log
- 2026-08-16 — scoped after shipping four features without pushing any of them, with
  every check green the whole way.
