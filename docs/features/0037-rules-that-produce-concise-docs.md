---
id: 0037
title: Trim the feature docs to their durable core
status: Draft
created: 2026-08-17
shipped:
check:
checked:
needs:
related: 0031-cut-the-kit-s-own-docs-to-what-gets-read.md
---

## Problem
Feature docs are agent context: 37 of them, 3,551 lines, 31k words, carried by every
session that reads the board. Most of the bulk is narration written alongside the work —
log diaries, walkthroughs of code that now exists, incidents already recorded in
`docs/TECH.md`. The decisions inside are worth keeping; the retelling around them is not.

## Non-goals
- **Losing anything.** Every decision, constraint, measurement and rejected alternative
  stays. If a sentence is the only record of why something is the way it is, it stays.
- **Hitting a number.** Each doc ends where its content ends.
- **Rewriting the protocol.** The one-off edit of shipped docs is authorised directly, not
  by a rule change.

## Design
One pass, oldest first, cutting only what a later agent gains nothing from:

| out | stays |
|---|---|
| `## Log` diaries — mistakes fixed, rules complied with | log lines where the plan changed |
| `## Design` walkthroughs of code that exists | decisions and constraints a reader cannot derive |
| incidents retold from `docs/TECH.md` or a check's message | the measurement that settled a choice |
| restatement between `## Problem` and `## Design` | the signal that justified the work |

`## Tasks` boxes stay verbatim: their ticks are what `unticked-shipped` reads, and they are
the record of what was actually done.

Two template edits stop the same padding being written again — `## Log` says one line per
entry and only when the plan changed, `## Design` says decisions rather than walkthrough.
Nothing else in the protocol changes.

## Tasks
- [x] Template guidance for `## Log` and `## Design`
- [x] 0001–0012 trimmed
- [x] 0013–0024 trimmed
- [ ] 0025–0036 trimmed
- [ ] `kit audit --strict` green, and a spot check that a trimmed doc still answers why the
      feature exists and what it decided

## Log
- 2026-08-17 — one-off, authorised directly: shipped docs are agent context rather than an
  archive, and rule 11 stands unchanged for everything else.
