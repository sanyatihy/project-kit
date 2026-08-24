---
id: 0068
title: A doc declares the dependency its own prose names
status: Draft
created: 2026-08-24
shipped:
needs: 0066
---

## Problem

The guidance above `needs:` answers the question it asks: *"Leave it empty if this could be
built today against what already exists — that is the common case."* A chain planning its
first round left `needs:` empty on all three docs while two of them named the dependency a
paragraph above the field — *"without this, 0003's charts have nothing to aggregate"*, *"the
numbers exist after 0001+0002"*. One of those features was then claimed off a trunk that did
not carry the parser it read, and the session that took it spent its budget merging the
branch it needed. The same chain declared `needs:` correctly on its next round, once it had
paid for the first: the nudge misleads a planner who has no such round behind them.

## Non-goals

- **Judging whether the declared ids are the right ones.** 0065's line, unchanged: no
  heuristic has that judgement.
- **Reading anything but `## Problem` and `## Design`.** `## Log` and `## Non-goals` cite
  neighbouring features constantly, and a citation there is history rather than a plan.
- **Reporting a mention of a feature that has shipped or been dropped.** Measured over the
  65 docs here: 41 name an id their `needs:` omits and every one is a reference to settled
  work, so a check without that gate is noise at 63%. With it, no doc on the trunk fires.

## Design

**The comment states the test and stops.** "Could this start before that ships?" is the
whole judgement; "that is the common case" is the part that pre-answered it, and the parallel
work it argues for is already argued for two lines below. `kit help template` prints the
template file itself, so the one edit covers the printed topic too.

**`needs-unstated` reports a non-terminal feature id named in `## Problem` or `## Design`
that `needs:` omits.** It is a grep against ids that resolve to a doc, so a four-digit run
that is a port number or a year names no feature and reports nothing. It skips terminal docs
for `needs-unparsed`'s reason: a frozen doc's `needs:` cannot be corrected, so a finding on
one never clears.

**LOW, and the first check to use that level.** The check cannot tell a dependency from a
comparison, so a draft contrasting itself with a sibling draft trips it, and the fix there is
a reworded sentence rather than a field — that is too weak for MEDIUM and worth printing
anyway, because the misread it catches costs a session. `--strict` gates on HIGH, so nothing
is blocked by it.

## Tasks

- [x] the template's `needs:` guidance states the test without pre-answering it
- [x] `audit` reports a non-terminal id named in `## Problem` or `## Design` that `needs:`
      omits, skipping terminal docs, with a conformance case
- [x] the docs on the trunk pass it

## Log

- 2026-08-24 — no doc on the trunk fires the check, which the terminal gate predicts:
  every doc there but this one is Shipped or Dropped. What the gate also hides is that a
  four-digit run in prose is resolved against *this* project's features, so a doc quoting
  another project's numbering can be reported for a feature it never meant. This doc does
  exactly that and is silent only because the ids it quotes have shipped here.
