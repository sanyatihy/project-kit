---
id: 0060
title: Every finding is a problem someone can fix
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs:
---

## Problem

`audit` is believed because it is quiet when nothing is wrong, and its vocabulary has grown
past what it protects: 34 checks, of which four have ever fired across 382 commits, in 969
lines of Go and 1,449 of test — 15.7% of the non-test kit.

Size is not the worst of it. A doc retired with `kit drop` before its plan was written
leaves three HIGH `unfilled-template` findings, whose offered fixes are "write it, or set
status: Draft until you can" — the first is work nobody wants and the second reopens
retired history, which the protocol forbids. `audit --strict` is project-wide and runs
before every ship, so one dropped stub refuses every agent's ship in that project,
permanently. `unticked-shipped` and `shipped-early` have the same shape: both fire only on
Shipped docs, both prescribe an edit to one, and `kit ship` refuses to create either state.

## Non-goals

- Cutting a check a shipped doc protects. 0019 forbids dropping `orphan-claim`; it stays at
  143 lines and zero firings.
- Re-litigating 0048 and 0049. This applies the standard those rounds set to what survived.
- Weakening what the loop guarantees. Every check about who owns what, and about a doc
  saying something untrue, stays.

## Design

A terminal doc is asked nothing. `unfilledTemplate` tests whether a doc is tracked and not
whether its status is terminal, so every check satisfiable only by editing a Shipped or
Dropped doc is scoped away from terminal docs or deleted. `unticked-shipped` and
`shipped-early` go.

A check that reports a state nobody may fix is deleted rather than tuned. `needs-cycle`
costs 58 lines for a state reachable only by writing two `needs:` fields at each other,
never observed. `no-plan` prints what `kit next` already prints. `dead-link` has fired zero
times and its parser once panicked inside the command that gates every ship. `fyi-stale`
reports a note nobody is required to act on.

One defect is one finding. Seven pairs are one problem with one fix wearing two names:
`no-tasks` into `unfilled-template`, `bad-status` into `no-status`, `no-tech` into
`tech-drift`, `no-inbox` and `no-vision` into `no-protocol`, `inbox-unknown` into
`inbox-unparsed`, `shipped-bad-date` into `shipped-no-date`. Each pairing is checked before
it is merged, and one that turns out to be two defects keeps its name.

Severity is the damage, not the age of the check. `split-claim` is MEDIUM, so two agents
believing they own one feature does not fail `--strict`, while `unticked-shipped` — a state
`ship` cannot produce — is HIGH and does.

The registry goes from 34 checks to 21, and HIGH from 11 to 9: two fewer ways for an
unrelated doc to refuse every agent's ship.

## Tasks

- [x] No check asks for an edit to a terminal doc, and a dropped stub audits clean
- [x] `unticked-shipped` and `shipped-early` are deleted, with their tests
- [x] `needs-cycle`, `no-plan`, `dead-link` and `fyi-stale` are deleted; `--fyi` stays
- [x] The seven duplicate findings are merged into one name each
- [x] `split-claim` is HIGH, and `kit help checks` and `docs/TECH.md` describe what remains

## Open questions

- Should `split-claim` be HIGH? It names the worst failure the kit has and gates nothing.
  Raising it refuses work that ships today. Leaning: HIGH.

## Log

- 2026-08-21 — `--fyi` stays, against the design. The check was a fuse on a note nobody owes
  an answer; the marker keeps that note from blocking `next` and `claim`, which is a
  separate promise.
- 2026-08-21 — `needs-cycle` deleted, but a doc whose `needs:` cites its own id stays
  visible: `needs-unknown` reports it as the need nothing can satisfy.
- 2026-08-21 — merging `no-tasks` widened `Unwritten()`, which `claim` refuses over, so a
  `## Tasks` of prose with no boxes is now refused as well as reported. Three LOW checks
  merged into MEDIUM ones, which empties that level.
- 2026-08-21 — `split-claim` is HIGH, taking the open question's leaning, so a project
  holding a stray claim cannot ship until `kit release` takes it back. HIGH is 10 rather
  than the design's 9 because of it.
