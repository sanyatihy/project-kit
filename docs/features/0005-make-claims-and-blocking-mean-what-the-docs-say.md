---
id: 0005
title: Make claims and blocking mean what the docs say
status: Shipped
created: 2026-08-11
shipped: 2026-08-12
related: ../../cmd/kit/help.go
---

## Problem

An outside review run against `kit help` and the subcommands, without reading this repo,
found three bugs and four fair misreadings. `templates/INBOX.md` is not a help topic;
`--strict` is absent from the Commands block; "a branch named after the doc" never says
local, remote and worktree branches all count; the tie-break among Accepted features is
stated nowhere. The kit promises an agent never needs this repository, and those are four
measured breaches of it.

The bugs: a question blocks any feature whose id it mentions in passing; a doc claiming
`In progress` with no branch audits clean; a feature with no boxes reports "every task is
ticked".

## Non-goals

- Auditing task quality. No heuristic separates a vague box from a good one, and a noisy
  check destroys audit's only asset.
- Feature-size limits by task count. The threshold would be arbitrary, and `stalled` covers
  the dimension that matters.
- Secret detection. Entropy scanning is noisy on prose.

## Design

Blocking reads the id field, not the line: the format fixes the position, so the parser uses
it. `kit block` ships in the same feature, because a strict parse over hand-typed lines
trades a false positive for a silent false negative, which is the worse failure.

New checks go in the existing registry, so `kit help checks` cannot drift and each is forced
a conformance case.

Protocol gaps stay prose: releasing a claim, rewriting a box, inserting urgent work, pushing
before working.

Help comes last, and success is not a longer help text — it is that the same review, run
again against the binary alone, cannot reach those four conclusions.

## Tasks

- [x] Blocking reads the feature-ID field of an INBOX line, not the whole line
- [x] `kit block <id> "<question>"` appends a canonical INBOX line
- [x] Audit: `inbox-unparsed` — a line outside the documented format
- [x] Audit: claim consistency — `In progress` with no branch, branch with no doc, two
      branches for one doc, a branch on a `Draft`
- [x] A feature with no task boxes reports that, in `next` and in `audit`
- [x] Audit: an INBOX question naming a feature ID that does not exist
- [x] Protocol: releasing a claim, rewriting a box, where urgent work is inserted, push
      before working
- [x] `kit ship <id>` — verify boxes, set status and `shipped:`, prompt the as-built migration
- [x] A `TECH.md` template, and `no-tech` pointing at it
- [x] `kit help` answers the four things the review got wrong

## Log

- 2026-08-11 — the four wrong findings are help defects rather than reader error. Treating
  them as reader error discards the only outside measurement of the kit's promise.
- 2026-08-12 — the claim checks read branch names, and this repo produced a false positive
  no fixture had: two merged branches on a Shipped feature read as two agents competing.
  Scoped to features still being worked.
