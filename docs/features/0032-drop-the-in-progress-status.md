---
id: 0032
title: Drop the In progress status
status: Shipped
created: 2026-08-17
shipped: 2026-08-17
needs: 0031
related: 0021-let-agents-work-from-drafts-and-make-accepting-a-signal.md
---

## Problem

Rule 1 forbids hand-editing `status:`. `kit claim` prints "set status: In progress in the
doc". Rule 8 says to hand-edit it back when releasing a claim. No command sets it. So the
protocol forbids the only available means for two transitions it requires, and an agent must
break one rule to follow another. Measured: `status: In progress` was hand-edited on 0027,
0028, 0029 and 0030, in breach of rule 1 every time and each time because `claim` said to.

The field is also a second copy of a fact git already holds. A claim is a branch on the
remote; `In progress` is the doc asserting the same thing, by hand, where it can disagree.
`no-claim` exists only to report that disagreement.

## Non-goals

- Removing `Accepted`. That one records a human's judgement and nothing else holds it.
- A new `kit start <id>` command. It would be a second copy kept in step by a second
  command — the same contradiction with more machinery.
- Changing what `claim` does. It still refuses, pushes and prints the worktree; it stops
  instructing an edit no rule permits.

## Design

`In progress` goes, and `no-claim` with it: the check exists only to reconcile the two
copies, so with one copy there is nothing to reconcile.

| what used it | now |
|---|---|
| `next` ordering: finishing beats starting | a feature with a claim branch sorts ahead of one without |
| `next`'s WIP warning | counts claims held, not statuses |
| `stalled` | already keys off the claim; the status arm is redundant |
| `no-tasks` (Accepted or In progress) | Accepted alone |
| `claim`'s closing instruction | drops the line telling you to edit the doc |
| rule 8, releasing a claim | delete the branch, local and remote; no status to restore |

`AllStatuses` loses it, so `bad-status` reports a doc still carrying it. That is the
migration: every shipped doc on the trunk is `Shipped` already, and a doc left saying
`In progress` is one whose claim was released without deleting the branch, which is worth a
finding.

The status line becomes `Draft -> Accepted -> Shipped`, with `Dropped` and `Superseded` as
exits, and the claim branch is what says work is happening.

## Tasks

- [x] `In progress` leaves `AllStatuses`, `claim`, `stalled` and `no-tasks`; `next` orders by
      claim with the WIP warning counting claims; `no-claim` is retired from the registry
- [x] `claim` no longer instructs a status edit, and prints what to do instead
- [x] Protocol rules 1, 2, 5 and 8, the template, `kit help` and README drop the status and
      say the claim is what marks work as started
- [x] A doc still saying `In progress` is reported as `bad-status` and nothing else breaks

## Log

- 2026-08-17 — promoted from BACKLOG.md after the contradiction was hit four times. The third
  of its three options: drop the status rather than add a command.
- 2026-08-17 — the design said `next` would sort a claimed feature ahead of an unclaimed one.
  Built, it changed nothing observable and was removed: the board already renders this
  checkout's claim as yours above everything and somebody else's under taken below what is
  free, so no ordering inside the list can move a feature the groups do not already place.
- 2026-08-17 — test fixtures stamp commits at midnight, not noon. A fixture aged N days sat
  N-and-a-half from midnight today and floored to N-1, which made every threshold test a day
  off once `stalled` keyed on the claim branch.
