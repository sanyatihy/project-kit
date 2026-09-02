---
id: 0041
title: Write about the system, not the session
status: Shipped
created: 2026-08-18
shipped: 2026-08-18
needs:
related: 0038-keep-docs-short-after-the-trim.md, 0037-rules-that-produce-concise-docs.md
---

## Problem

The docs spend their words on the wrong sections. Measured across this repo's 40 feature
docs against the 14 in `localcode`, a project built with the kit: `## Problem` averages 12.9
lines here against 7.2 there, `## Design` 17.9 against 35.6, `## Log` 5.0 against 17.1. The
kit's own docs argue the work and skimp on what was decided and what the doing taught, which
are the two things a later reader cannot get from the code.

0038's thresholds cannot see this and are calibrated in the wrong direction: taken from this
repo immediately after 0037 cut it, they fire on 7 of `localcode`'s 14 docs and 13 of its 56
`## Log` entries.

## Non-goals

- Per-section budgets, or a `## Problem` ≥ `## Design` check. The sharper signal, and it
  needs its own calibration rather than riding along.
- Trimming the corpus again. 0037 did that once and rule 11 freezes what shipped.
- Judging quality. A check counts; whether a doc is good is review's job.

## Design

Write about the system, not the session. A sentence about the work — which session it
happened in, what an earlier try did, who got a rule wrong — is not about the system and
stops being true the moment it is fixed. The incident lives in the pull request and anything
durable in `docs/TECH.md`. Eight docs here narrate a session outright, which is where the
12.9-line `## Problem` average comes from.

`session-narration` at LOW backstops it on a small marker set, following `secret-in-doc`'s
precedent and inheriting its honesty: narration written without the markers matches nothing,
so the check is a backstop and the rule is the rule.

The `## Log` carries what the doing taught, and the entry cap goes. 0038 capped an entry at
five lines, measured over a corpus whose Logs had just been cut, so the calibration only
confirmed the trim. The reference corpus puts 13 of 56 entries past that cap and they hold
its best content — a measurement, a falsified premise, what a dependency's outcome changed
downstream. Entry length was never the signal; the diary is.

`DocLineLimit` is recalibrated against the reference corpus instead of this one, or it
cancels the change on the way out: 7 of 14 docs there are already over 100 lines, and a Log
that grows is what pushes a doc past it.

A feature records a decision. A change whose whole story fits in its commit message is a
BACKLOG line and a commit, not six sections padded to fill. Nothing can read a doc and rule
on whether it decided anything, so this stays prose, in rule 1 beside the too-big/too-small
guidance that draws the feature's other boundary.

## Tasks

- [x] Rule in `templates/AGENTS-section.md`: write about the system, not the session, with
      the keep/cut test; and rule 1 says what earns a doc rather than a BACKLOG line
- [x] `## Log` template guidance rewritten to what the doing taught, and `## Problem` given
      back its stated budget
- [x] `session-narration` at LOW over the marker set
- [x] `doc-bloat` loses its `## Log` entry arm, and `DocLineLimit` is retaken from the
      reference corpus with the sample and date in the comment
- [x] README and `kit help checks` carry the change

## Log

- 2026-08-18 — the protocol ceiling goes 115 to 121. Both rules delete a class of writing
  from every doc after them, which is the trade that buys protocol growth; a rule that only
  adds something to remember does not.
- 2026-08-18 — the marker set has poor recall and keeps it. Nine candidate phrases catch 4
  docs of 40 and misfire on none, and the thirty-line arguments carry no stock phrase at
  all. That is `secret-in-doc`'s shape: certain when it fires, blind to prose.
- 2026-08-18 — scoped to docs that are not Shipped. It fired on four frozen docs and rule 11
  forbids the fix, so on the corpus it was aimed at the check could only ever be noise.
- 2026-08-18 — `DocLineLimit` 100 to 150, from the reference corpus: median 104, p90 146,
  longest 211 over 14 docs. One doc there is still over it, which is the wanted behaviour —
  a limit above the longest thing ever written is not an alarm.
