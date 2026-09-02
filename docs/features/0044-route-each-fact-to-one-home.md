---
id: 0044
title: Route each fact to one home
status: Shipped
created: 2026-08-18
shipped: 2026-08-18
needs:
related: 0041-write-about-the-system-not-the-session.md, 0037-rules-that-produce-concise-docs.md
---

## Problem

Both documents a later reader depends on carry the transcript of the work rather than its
result. `docs/TECH.md` is 293 lines, of which roughly 120 are rules a newcomer acts on; the
rest is the incident each rule came from, welded into the same sentence, so it can be neither
skipped nor cut. 0041 read the same defect in `## Log` and calibrated against `localcode`,
whose logs are R&D working notes — fourteen entries in one feature, each revising the last —
then removed the entry cap on that reading.

## Non-goals

- Shorter docs. Length is the symptom; what a line is for is the rule.
- The `## Problem` ≥ `## Design` composition check. Still the sharper instrument, and it
  needs a corpus written under this rule before it can be calibrated.
- Re-trimming shipped feature docs. Rule 11 freezes them, and they are where the incidents
  belong.
- Ruling on whether a fact is durable. No check can; one shape gets a detector and the rest
  is prose.

## Design

Three destinations, chosen as the sentence is written.

| the sentence is | it goes to |
| --- | --- |
| true after this feature ships | `docs/TECH.md`, one line |
| what changed this doc's plan | one `## Log` entry naming the change |
| neither | the pull request |

What makes it mechanical is "name what changed". An entry that cannot name a box rewritten,
a premise falsified, or a number something downstream reads is not an entry. That is what
"what the doing taught" fails to exclude: under R&D the doing teaches on every run, so every
run qualifies and the doc becomes the run log with the conclusions mixed in.

Rule and evidence are separate sentences, and only the rule is mandatory. Evidence earns a
trailing clause where it stops the rule being reversed — the constraint or the measurement,
never how it was discovered. The budget stays (`116 git processes against 29`); the session
that measured it goes.

0041's mechanism stands and its calibration does not. The per-entry cap stays removed, but
the reference corpus is withdrawn: `localcode`'s logs are notes taken during a sweep, which
is the shape being excluded, so a target taken from them aims at the defect.

Detection is one marker set, with the bargain `secret-in-doc` makes. `session-narration`
grows to the revision shapes an R&D corpus produces, keeping only markers that misfire on
neither corpus. `kit ship` is where routing is actually decided, so it names the three
destinations rather than saying "move durable facts".

`docs/TECH.md` is rewritten to the rule it now states. It is read before every change and it
is the worked example; left as it is, it teaches the old shape to everything that reads it.

## Tasks

- [x] The protocol states the three destinations and the name-the-change test, replacing
      "Write the record while you work"
- [x] `## Log` guidance in the feature template says name the change and routes what is not
      one
- [x] `templates/TECH.md` carries the rule-then-clause shape
- [x] `session-narration` covers the revision shapes, markers kept only where they misfire
      on neither corpus
- [x] README's account of `## Log` states the routing rule
- [x] `kit ship` names the three destinations where it said "move durable facts"
- [x] `docs/TECH.md` is rewritten to rule-then-clause, and `techBudget` retaken from the
      result

## Log

- 2026-08-18 — the box grew a fix: every marker is several words and the scan was
  line-at-a-time, so any marker the wrap split matched nothing. `Narration` now scans a
  two-line window.
- 2026-08-18 — the incident narration was clauses, not the bulk. Rewriting `docs/TECH.md` to
  rule-then-clause rewrote 84 of 293 lines and left the file 4 shorter, where the Problem
  implied roughly half was archaeology. `techBudget` 300 to 295 rather than the cut.
