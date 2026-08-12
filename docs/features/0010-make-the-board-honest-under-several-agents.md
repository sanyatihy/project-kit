---
id: 0010
title: Make the board honest under several agents
status: Accepted
created: 2026-08-12
shipped:
check:
checked:
related: 0006-make-the-kit-safe-for-concurrent-agents.md
---

## Problem
With one feature in flight the board is fine. With several it says two things that
are not true.

**A branch holding an unaccepted `Draft` is reported as a claim.** The protocol has
agents draft features and leave them for a human to accept, and nothing is ever
committed to `main`, so the draft sits on a branch until it is accepted. `audit` calls
that `draft-claimed` — *"someone is building on a design nobody settled"* — which is
the documented flow working, described as a defect. It is firing on this doc right
now. A finding that fires on correct behaviour is the failure mode `audit` cannot
afford, and this one fires on the first step of the loop.

**Nothing says how far along anything is.** `kit next` names each feature's next
unticked box and never how many are done. With one agent that is enough, because the
next box is the whole state. With four, the question the operator actually has is
which of them are moving, and the board cannot answer it — neither in the text output
nor in `--json`.

## Non-goals
- **Running agents.** `herdr` owns the runtime — worktrees, panes, agent lifecycle,
  prompting, and knowing when an agent is genuinely blocked. It does all of what this
  doc previously proposed and two things that were going to be hard. The kit owns the
  work: what to build, who holds it, how far along, what drifted. Neither needs to
  know about the other.
- **Estimating.** Boxes are not equal and the count is not a forecast. `3/6` says what
  has been ticked, nothing more; anything that implies time would be invented.
- **Progress for a feature nobody is on.** The number is only interesting where work
  is happening, and putting it everywhere makes the board noisier for no gain.
- **Changing what `draft-claimed` means for a real claim.** A branch on a `Draft`
  someone is *building* on is still worth reporting. Only the review case is new.

## Design
**A `Draft` branch is a claim only when it carries work.** The distinguishing fact is
already in git: a branch that holds nothing but the feature doc itself is a draft
waiting for a human, and a branch that has touched anything else is somebody building
on an unsettled design. That is a question `audit` can answer from the commits it
already reads, with no new state and no new field in the doc.

Reporting turns on the same fact rather than going silent: the doc-only case is not a
finding at all, because there is nothing wrong with it.

**Progress is ticked-over-total, next to the feature.** `TaskCount` already computes
it and nothing surfaces it. It goes in the text output where the eye lands and in
`--json`, because with a runtime like `herdr` watching panes, `kit next --json` is
what anything else has to read to know about the plan.

## Tasks
- [x] `draft-claimed` fires only when a Draft's branch carries work beyond the doc itself
- [x] `kit next` shows ticked-over-total per feature, in text and in `--json`

## Open questions
<!-- none -->

## Log
- 2026-08-12 — this doc was drafted as "Run several agents from the kit itself", a
  `kit agent` subcommand over tmux. Pivoted rather than dropped: it was still `Draft`,
  and `herdr` turned out to ship every box of it — `herdr worktree create/open/remove`
  and `herdr agent start/prompt/wait/read/attach`, including the two parts the fan-out
  test had just proved were the hard ones. Building it would have been a worse copy of
  something already installed.
- 2026-08-12 — the reversal is the third time an outside tool redirected a design here,
  after `beads` on atomic claims and the terminal-substrate reading. Each time what
  survived was the part nobody else does: durable state about the work. That is worth
  applying as a test before the next feature rather than rediscovering.
- 2026-08-12 — kept `draft-claimed` rather than deleting it. The check is right about
  the case it was written for; it just never knew about drafting, because when it was
  written agents did not draft.
