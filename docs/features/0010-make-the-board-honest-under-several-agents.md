---
id: 0010
title: Make the board honest under several agents
status: Shipped
created: 2026-08-12
shipped: 2026-08-12
related: 0006-make-the-kit-safe-for-concurrent-agents.md
---

## Problem
With several features in flight the board says two untrue things.

A branch holding an unaccepted `Draft` is reported as a claim. Agents draft and leave
the doc for a human, and nothing is committed to `main`, so the draft sits on a branch —
which `audit` calls `draft-claimed`, "building on a design nobody settled". It fires on the
first step of the documented loop.

Nothing says how far along anything is. `next` names the next unticked box and never how
many are done. With four agents the operator's question is which of them are moving.

## Non-goals
- Running agents. `herdr` owns the runtime — worktrees, panes, lifecycle, prompting. The kit owns the work: what to build, who holds it, how far along, what drifted.
- Estimating. Boxes are not equal; `3/6` says what has been ticked and nothing more.
- Progress for a feature nobody is on. Only interesting where work is happening.

## Design
A `Draft` branch is a claim only when it carries work. The distinguishing fact is
already in git — a branch holding nothing but the feature doc is a proposal, one that
touched anything else is somebody building. No new state, no new field.

Progress is ticked-over-total, in the text output and in `--json`, because a runtime
watching panes reads the JSON to know about the plan.

## Tasks
- [x] `draft-claimed` fires only when a Draft's branch carries work beyond the doc itself
- [x] `kit next` shows ticked-over-total per feature, in text and in `--json`

## Log
- 2026-08-12 — pivoted in place from "Run several agents from the kit itself", a `kit agent`
  subcommand over tmux: `herdr` ships every box of it, including the two the fan-out test had
  just shown were hardest. Third time an outside tool redirected a design here, after `beads`
  on atomic claims; each time what survived was durable state about the work.
- 2026-08-12 — kept `draft-claimed` rather than deleting it. The check is right about the
  case it was written for; it never knew about drafting, because agents did not draft then.
