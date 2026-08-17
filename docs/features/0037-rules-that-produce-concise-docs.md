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
The kit's writing guidance says how *much* to write and not what to leave out, so agents
following it produce padding that passes every check. Measured here against
`../localcode`, built with the same protocol:

| | this repo | localcode |
|---|---|---|
| `docs/TECH.md` | 254 lines | 77 |
| typical `## Log` entry | 4–6 lines | 1–3 |

The three failures, in what they cost:

- **`## Log` reads as a diary.** "Committed once with `make check` red", "the box said
  ≤200 and lands at 247", "paid for by folding it into the conformance paragraph". None
  changes a later decision; all of it is read by every agent that opens the doc.
- **TECH retells incidents where a fact would do.** localcode states the parameter and
  links the feature doc for the argument; this repo carries the argument twice.
- **Ceilings became targets.** Three features in a row ended by compressing content to fit
  a number set in an earlier feature — the metric optimising itself, at the cost of the
  entries that made a rule decidable.

## Non-goals
- **Editing shipped docs.** Frozen history stays frozen; this changes what gets written
  next. A cleanup pass is a separate decision.
- **A verbosity check.** Nothing separates a long necessary entry from a long unnecessary
  one, and a check that fires on healthy docs is worse than no check.

## Design
The enforcement point is the template and the protocol, because they are what an agent
reads at the moment it writes. All three edits say what to leave out.

**`## Log` is decisions, not a diary.** What earns an entry: a box rewritten, a premise
falsified, an open question settled, a measurement that changes what to do next. What
never does: mistakes made and fixed, compliance with a rule, restatement of the diff.
One or two lines each, in the doc whose plan changed.

**TECH states facts.** Parameters as tables, the argument left in the feature doc and
linked. An incident belongs in that feature's `## Problem`, not retold here.

**Ceilings are alarms.** `budget_test` already says to raise the number and give a reason;
the protocol gains the other half — never compress an entry that earns its place to fit
one. Growth should be visible, not prevented.

The kit's own `docs/TECH.md` is the worked example: it goes back through the same rule.

## Tasks
- [ ] `templates/features-TEMPLATE.md`: `## Log` says what earns an entry and what never
      does; `## Design` and `## Problem` say the same in one line each
- [ ] `templates/TECH.md` and `kit help as-built`: facts and tables, argument in the
      feature doc, no incident retelling
- [ ] Protocol's writing rule: ceilings are alarms, raised with a reason, never met by
      compressing what earns its place
- [ ] `docs/TECH.md` rewritten to that rule, and the ceiling set from what it lands at

## Log
- 2026-08-17 — asked for after comparing this repo's docs with `../localcode`.
