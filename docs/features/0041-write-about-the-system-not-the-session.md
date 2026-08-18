---
id: 0041
title: Write about the system, not the session
status: Draft
created: 2026-08-18
shipped:
check:
checked:
review:
needs:
related: 0038-keep-docs-short-after-the-trim.md, 0037-rules-that-produce-concise-docs.md
---

## Problem
The docs spend their words on the wrong sections. Measured across this repo's 40 feature
docs against the 14 in `localcode`, a project built with the kit: `## Problem` averages
12.9 lines here against 7.2 there, `## Design` 17.9 against 35.6, `## Log` 5.0 against 17.1.
The kit's own docs argue the work and skimp on what was decided and what the doing taught —
the two things a later reader cannot get from the code.

0038's thresholds cannot see this, and are calibrated in the wrong direction: taken from
this repo immediately after 0037 cut it, they fire on 7 of `localcode`'s 14 docs and 13 of
its 56 `## Log` entries.

## Non-goals
- **Per-section budgets, or a `## Problem` ≥ `## Design` check.** The sharper signal — it
  separates the two corpora exactly, 10 docs here and 0 there — and it needs its own
  calibration rather than riding along. BACKLOG.
- **Trimming the corpus again.** 0037 did that once and rule 11 freezes what shipped. This
  changes what gets written next.
- **Judging quality.** A check counts; whether a doc is good is review's job.

## Design
**Write about the system, not the session.** The rule an agent can apply: a sentence about
*the work* — which session it happened in, what the previous attempt did, who got a rule
wrong — is not about the system and stops being true the moment it is fixed. State what
holds. The incident lives in the pull request, and anything durable in `docs/TECH.md`.
This is where the 12.9-line `## Problem` average comes from; eight docs here narrate a
session outright.

`session-narration` at LOW backstops it on a small marker set. It follows `secret-in-doc`'s
precedent and inherits its honesty: narration written without the markers matches nothing,
so the check is a backstop and the rule is the rule.

**The `## Log` carries what the doing taught, and the entry cap goes.** 0038 capped an entry
at five lines and the template narrowed it to "only when the plan changed" — both measured
over a corpus whose Logs had just been cut, so the calibration only confirmed the trim. The
reference corpus puts 13 of 56 entries past that cap, and they hold its best content: a
measurement, a falsified premise, what a dependency's outcome changed downstream. Entry
length was never the signal — the diary is, and 0037's guidance already excludes it — so the
arm is removed rather than raised.

`DocLineLimit` is recalibrated against the reference corpus instead of this one, or it
cancels the change on the way out: 7 of 14 docs there are already over 100 lines, and a
Log that grows is what pushes a doc past it. Same convention as 0038 — the number, its
sample and its date in the constant's comment.

**A feature records a decision.** 40 docs here against 14 there, for the smaller subject of
the two. A change whose whole story fits in its commit message is a BACKLOG line and a
commit, not six sections padded to fill. Nothing can read a doc and rule on whether it
decided anything, so this one is prose that stays prose — it goes in rule 1, beside the
too-big/too-small guidance that already draws the feature's other boundary.

## Tasks
- [x] Rule in `templates/AGENTS-section.md`: write about the system, not the session, with
      the keep/cut test; and rule 1 says what earns a doc rather than a BACKLOG line
- [ ] `## Log` template guidance rewritten to what the doing taught, and `## Problem` given
      back its stated budget
- [ ] `session-narration` at LOW over the marker set, with conformance that it fires and
      that a doc without the markers stays quiet
- [ ] `doc-bloat` loses its `## Log` entry arm, and `DocLineLimit` is retaken from the
      reference corpus with the sample and date in the comment; conformance updated
- [ ] README and `kit help checks` carry the change, and this doc's own sections are
      measured against the rule it adds

## Log
- 2026-08-18 — measured before drafting: `## Design` 17.9 lines here against 35.6 in the
  reference corpus, `## Log` 5.0 against 17.1, `## Problem` 12.9 against 7.2. 0038's two
  thresholds fire on 7 of 14 docs and 13 of 56 log entries there.
- 2026-08-18 — the protocol ceiling goes 115 to 121. Both rules delete a class of writing
  from every doc after them, which is the trade that buys protocol growth; a rule that only
  adds something to remember does not, and the number is there to force that question.
