---
id: 0038
title: Trim every feature doc to its durable core
status: Draft
created: 2026-08-17
shipped:
check:
checked:
needs: 0037
related: 0037-rules-that-produce-concise-docs.md
---

## Problem
37 feature docs, 3,551 lines, 31k words — roughly 40k tokens of agent context, most of it
narration written before the rules in 0037 existed. The worst are 186, 176 and 171 lines
for features whose durable content is a paragraph.

## Non-goals
- **Reversing any decision.** Compaction removes retelling; every decision, constraint and
  measurement that shaped the code stays, in fewer words.
- **Deleting docs.** The ID is a primary key and history stays addressable.

## Design
One pass over `docs/features/*.md`, oldest first, each doc cut to what a later agent needs:
why it exists, what was decided, what it changed. Rule of thumb per doc — `## Problem` in
under five lines, `## Design` holding only decisions a reader cannot derive from the code,
`## Log` one line per entry that changed the plan and none for anything else.

What comes out, in the order it is worth the most:

- **Log diaries.** Process notes, mistakes made and fixed, compliance narration.
- **Walkthroughs in `## Design`** of code that now exists and can be read.
- **Incident retellings** already recorded in `docs/TECH.md` or in the check's own message.

`## Tasks` boxes stay as they are: they are the record of what was actually done, and their
ticks are load-bearing for `unticked-shipped`.

Target: under 1,500 lines total, with no doc over 80 so `doc-bloat` is quiet. The number is
an alarm and not the goal — a doc that needs 90 lines keeps them and says why in its Log.

## Tasks
- [ ] 0001–0012 trimmed
- [ ] 0013–0024 trimmed
- [ ] 0025–0036 trimmed
- [ ] `kit audit --strict` green, `doc-bloat` quiet, and the totals recorded in `## Log`

## Log
- 2026-08-17 — split from 0037 so the rules land before anything is trimmed against them.
