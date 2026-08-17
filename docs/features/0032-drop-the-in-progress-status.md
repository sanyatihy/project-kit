---
id: 0032
title: Drop the In progress status
status: Shipped
created: 2026-08-17
shipped: 2026-08-17
check:
checked:
needs: 0031
related: 0021-let-agents-work-from-drafts-and-make-accepting-a-signal.md
---

## Problem
Rule 1 forbids hand-editing `status:` — *"that is what `accept`, `ship` and `drop` are
for"*. `kit claim` prints *"set status: In progress in the doc"*. Rule 8 says to hand-edit
it back when releasing a claim. No command sets it. So the protocol forbids the only
available means for two transitions it requires, and an agent must break one rule to
follow another.

Measured this session: I hand-edited `status: In progress` on 0027, 0028, 0029 and 0030 —
in breach of rule 1 every time, and the reason each time was that `claim` told me to. The
backlog entry naming this said *"promote when someone hits the contradiction for real"*.

The field is also a second copy of a fact git already holds. A claim is a branch on the
remote; `In progress` is the doc asserting the same thing, by hand, where it can disagree.
`no-claim` exists only to report the disagreement.

## Non-goals
- **Removing `Accepted`.** That one records a human's judgement and nothing else holds it.
- **A new `kit start <id>` command.** It would be a second copy kept in step by a second
  command — the same contradiction with more machinery. "Enforce, do not instruct" says
  remove the need, not move it to a stronger tier.
- **Changing what `claim` does.** It still refuses, pushes, and prints the worktree; it
  simply stops instructing an edit no rule permits.

## Design
`In progress` goes, and `no-claim` goes with it — the check exists only to reconcile the
two copies, so with one copy there is nothing to reconcile.

What has to absorb the loss, and how:

| what used it | now |
|---|---|
| `next` ordering: finishing beats starting | a feature with a claim branch sorts ahead of one without |
| `next`'s WIP warning | counts claims held, not statuses |
| `stalled` | already keys off the claim; the status arm is redundant |
| `no-tasks` (Accepted or In progress) | Accepted alone |
| `claim`'s closing instruction | drops the line telling you to edit the doc |
| rule 8, releasing a claim | delete the branch, local and remote; no status to restore |

`AllStatuses` loses it, so `bad-status` reports a doc still carrying it — which is the
migration: every shipped doc on the trunk is `Shipped` already, and a doc left saying
`In progress` is one whose claim was released without deleting the branch, which is worth
a finding.

The status line in the docs becomes `Draft -> Accepted -> Shipped`, with `Dropped` and
`Superseded` as exits, and the claim branch is what says work is happening.

## Tasks
- [x] `In progress` leaves the vocabulary — `AllStatuses`, `claim`, `stalled`, `no-tasks`
      — and `next` orders by claim with the WIP warning counting claims; `no-claim` is
      retired from the registry and its conformance case; conformance that a claimed
      feature still sorts ahead of an unclaimed one
- [x] `claim` no longer instructs a status edit, and prints what to do instead
- [x] Protocol rules 1, 2, 5 and 8, the template, `kit help` and README drop the status
      and say the claim is what marks work as started
- [x] Conformance: a doc still saying `In progress` is reported as `bad-status` and
      nothing else breaks

## Log
- 2026-08-17 — promoted from BACKLOG.md after the contradiction was hit four times in one
  session. The third of its three options: drop the status rather than add a command.
- 2026-08-17 — the first two boxes merged into one, per rule 7. They cannot be separate
  commits that both compile: `ActiveFeatures` ranks by the status and `Plan.InProgress`
  counts it, so removing the constant and replacing the ordering are the same edit.
- 2026-08-17 — this doc is `In progress` while the feature that removes that status is
  built, which is the contradiction it exists to end. `ship` sets `Shipped` from whatever
  it finds, so the last edit clears it.
- 2026-08-17 — the design said `next` would sort a claimed feature ahead of an unclaimed
  one. Built, it changed nothing observable and was removed: the board already renders
  this checkout's claim as **yours** above everything and somebody else's under **taken**
  below what is free, so no ordering inside the list can move a feature anywhere the
  groups do not already put it. The conformance asserts the invariant the groups give.
- 2026-08-17 — `commitAllAt` stamped commits at noon, so a fixture aged N days sat
  N-and-a-half days from midnight today and floored to N-1. It never showed while
  `stalled` could fire on the status alone; keying it to the claim branch made every
  threshold test a day off. Commits are stamped at midnight now.
