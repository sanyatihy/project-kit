---
id: 0060
title: Every finding is a problem someone can fix
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

`audit` is believed because it is quiet when nothing is wrong, and the vocabulary has
grown past what it protects. Thirty-four checks, of which four have ever fired across the
382 commits in this repo's history. `audit` is 969 lines of Go and 1,449 lines of test —
15.7% of the non-test kit and 21.6% of its tests.

The worst of it is not size. A doc retired with `kit drop` before its plan was written
leaves three HIGH `unfilled-template` findings, whose offered fixes are "write it, or set
status: Draft until you can" — the first is work nobody wants and the second reopens
retired history, which rule 21 forbids. `audit --strict` is project-wide and the protocol
runs it before every ship, so **one dropped stub refuses every agent's ship in that
project, permanently.**

Three more checks have that shape. `unticked-shipped` and `shipped-early` fire only on
Shipped docs and both prescribe an edit to one. `kit ship` refuses to create either state.

VISION states the standard directly: a check that fires on healthy projects is worse than
no check.

## Non-goals

- **Cutting a check a shipped doc protects.** 0019's non-goals forbid dropping
  `orphan-claim`; it stays, at 143 lines and zero firings, and the number is recorded
  rather than acted on.
- **Re-litigating 0048 and 0049.** Five checks were cut then. This applies the standard
  those rounds set to the ones that survived, and adds no new opinion about documents.
- **Weakening what the loop guarantees.** Every check about who owns what, and about a doc
  saying something untrue, stays.

## Design

**A terminal doc is asked nothing.** `unfilledTemplate` tests whether a doc is tracked and
not whether its status is terminal. Every check that can only be satisfied by editing a
Shipped or Dropped doc is either scoped away from terminal docs or deleted — the same
argument 0049 used to delete `missing-section`, applied to the checks that outlived it.
`unticked-shipped` and `shipped-early` go, because `kit ship` refuses to produce either
and both fixes are forbidden edits.

**A check that reports a state nobody may fix is deleted, not tuned.** Four more go on
their own evidence. `needs-cycle` costs 58 lines for a state reachable only by writing two
`needs:` fields at each other, never once observed. `no-plan` prints the message and the
fix string `kit next` already prints. `dead-link` has fired zero times and its parser once
panicked inside the command that gates every ship. `fyi-stale` reports a note nobody is
required to act on, and takes `block --fyi` with it.

**One defect is one finding.** Seven pairs are one problem with one fix wearing two names:
`no-tasks` into `unfilled-template`, `bad-status` into `no-status`, `no-tech` into
`tech-drift`, `no-inbox` and `no-vision` into `no-protocol`, `inbox-unknown` into
`inbox-unparsed`, `shipped-bad-date` into `shipped-no-date`. `docs/TECH.md` already
describes the last pair as one decision with three outcomes. Each pairing is checked
before it is merged, and one that turns out to be two defects keeps its name.

**Severity is the damage, not the age of the check.** `split-claim` is MEDIUM, so two
agents believing they own one feature does not fail `--strict`, while `unticked-shipped` —
a state `ship` cannot produce — is HIGH and does. Raising it is a change to when work is
refused, so it is asked rather than assumed.

The registry goes from 34 to 21 and HIGH from 11 to 9, which is two fewer ways for an
unrelated doc to refuse every agent's ship.

## Tasks

- [ ] No check asks for an edit to a terminal doc, and a dropped stub audits clean
- [ ] `unticked-shipped` and `shipped-early` are deleted, with their tests
- [ ] `needs-cycle`, `no-plan`, `dead-link` and `fyi-stale` are deleted, and `--fyi` with them
- [ ] The seven duplicate findings are merged into one name each
- [ ] `kit help checks` and `docs/TECH.md` describe the vocabulary that remains

## Open questions

- **Should `split-claim` be HIGH?** It names the worst failure the kit has and currently
  gates nothing. Raising it means a project with a stray claim branch cannot ship until it
  is cleaned up, which is the intent, but it will refuse work that ships today. The
  leaning is HIGH.

## Log
