---
id: 0005
title: Make claims and blocking mean what the docs say
status: Shipped
created: 2026-08-11
shipped: 2026-08-12
check:
checked:
related: ../../cmd/kit/help.go
---

## Problem
An outside review, conducted against `kit help` and the subcommands without reading this
repo, found three real bugs and four things it got wrong — and every one of the four is a
fair reading of what the binary prints. `templates/INBOX.md` is not a help topic, `--strict`
is absent from the Commands block, "a branch named after the doc" never says local, remote
and worktree branches all count, and the tie-break among Accepted features is stated
nowhere. The kit's promise is that an agent never needs this repository; those are four
measured breaches of it.

The bugs: a question blocks any feature whose ID it mentions in passing; a doc claiming
`In progress` with no branch audits clean; a feature with no boxes reports "every task is
ticked".

## Non-goals
- **Auditing task quality.** No heuristic separates a vague box from a good one, and a noisy check destroys audit's only asset.
- **Feature-size limits by task count.** The threshold would be arbitrary; `stalled` covers the dimension that matters.
- **Secret detection.** Entropy scanning is noisy on prose — if ever, a handful of unambiguous patterns.

## Design
**Blocking reads the ID field, not the line.** The format already fixes the position; the
parser should use it. `kit block` ships in the same feature rather than after it — a strict
parse over hand-typed lines trades a false positive for a silent false negative, which is
the worse failure.

**New checks go in the existing registry**, so `kit help checks` cannot drift and every one
is forced a conformance case.

**Protocol gaps are prose**, not machinery: releasing a claim, rewriting a box, inserting
urgent work, pushing before working.

**Help comes last**, and success is not a longer help text — it is that the same review, run
again against the binary alone, cannot reach those four conclusions.

## Tasks
- [x] Blocking reads the feature-ID field of an INBOX line, not the whole line
- [x] `kit block <id> "<question>"` appends a canonical INBOX line
- [x] Audit: `inbox-unparsed` — a line outside the documented format, found while fixing the parser
- [x] Audit: claim consistency — `In progress` with no branch, branch with no doc, two branches for one doc, a branch on a `Draft`
- [x] A feature with no task boxes reports that, in `next` and in `audit`, instead of "every task is ticked"
- [x] Audit: an INBOX question naming a feature ID that does not exist
- [x] Protocol: releasing a claim, rewriting a box, where urgent work is inserted, push before working
- [x] `kit ship <id>` — verify boxes, set status and `shipped:`, prompt the as-built migration
- [x] A `TECH.md` template, and `no-tech` pointing at it
- [x] Tighten `kit help` so the four things this review got wrong are answerable from the binary

## Log
- 2026-08-11 — the four wrong findings were reclassified as help defects. Treating them as
  reader error would have discarded the only outside measurement of the kit's promise.
- 2026-08-12 — the claim checks read branch names, and running them against this repo — the
  first with real branches — produced a false positive no fixture had: two merged branches
  on a Shipped feature read as two agents competing. Scoped to features still being worked.
