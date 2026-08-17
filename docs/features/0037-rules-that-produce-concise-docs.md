---
id: 0037
title: Rules that produce concise docs
status: Draft
created: 2026-08-17
shipped:
check:
checked:
needs:
related: 0031-cut-the-kit-s-own-docs-to-what-gets-read.md
---

## Problem
Feature docs are agent context, and this repo's are 3,551 lines / 31k words across 37
docs. The guidance says how much to write, never what to leave out, so every doc is padded
and still passes every check.

| | here | target |
|---|---|---|
| feature doc | 96 lines avg, 186 worst | ≤60, most far less |
| `## Log` entry | 4–6 lines | 1 |
| `docs/TECH.md` | 254 | ≤150 |

## Non-goals
- **Cutting decisions or evidence.** Compression removes narration, never a constraint or
  the measurement behind a choice.
- **Trimming the existing docs.** That is 0038, once the rules exist to trim against.

## Design
Budgets per section, stated where they are read — in the template's own guidance — and
small enough to force a choice:

| section | budget | rule |
|---|---|---|
| `## Problem` | 5 lines | what is broken and the signal it matters now |
| `## Non-goals` | 1 line each | the boundary and where it lives instead |
| `## Design` | 20 lines | decisions a builder cannot derive; no walkthrough |
| `## Tasks` | 1 line per box | the outcome, not the activity |
| `## Log` | **1 line per entry** | only: a box rewritten, a premise falsified, a question settled |

**`## Log` is not a diary.** Mistakes made and fixed, compliance with a rule, and anything
the diff shows earn no entry at all.

**Shipping compacts the doc.** `kit ship` prints the instruction: cut what the code now
says, keep the decisions. Rule 11 changes from *never edit a Shipped doc* to *never change
what a Shipped doc decided* — compaction is allowed, reversal is not.

**`doc-bloat`, LOW.** A feature doc over 80 lines. Low because length is hygiene, not
drift; it exists so growth is visible without blocking a ship.

## Tasks
- [ ] Template: per-section budgets and what earns a `## Log` entry
- [ ] Protocol: rule 11 allows compaction; the writing rule says ceilings are alarms,
      raised with a reason, never met by compressing what earns its place
- [ ] `doc-bloat` at LOW, with conformance both ways
- [ ] `templates/TECH.md` and `kit help as-built`: facts and tables, argument left in the
      feature doc

## Log
- 2026-08-17 — asked for after comparing with `../localcode`, whose docs are shorter than
  this repo's and still too long.
