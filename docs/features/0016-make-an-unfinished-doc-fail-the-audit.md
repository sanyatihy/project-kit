---
id: 0016
title: Make an unfinished doc fail the audit
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
needs:
related: 0009-fix-what-the-first-real-run-found.md
---

## Problem
The audit rewards not doing the work. `kit new`, change `status: Draft` to `Accepted`, touch
nothing else: `next` hands an agent `…` as its instruction beside the command to claim it,
and `audit` says the docs and repo agree. Delete the placeholder `## Log` line — what a
careful author does first — and *that* is the only finding.

Both halves have one cause: `missing-section` tests whether a section is empty, and the
template fills every section with placeholder prose, two `- [ ] …` boxes and a fake dated
log entry. The untouched doc looks complete and the tidied one looks broken. That is 0005's
failure inverted, which is worse: the lazy path is green.

Two smaller versions found in the same pass: **`--fyi` does not mean FYI to `audit`** —
`inbox-stale` ignores the marker, is HIGH, and at a 1-day threshold on a date-only line a
note filed at 23:00 fails `--strict` an hour later. And **`BACKLOG.md` is not scanned for
secrets**, though rule 6 sends discovered work there.

## Non-goals
- **A prose-quality heuristic.** This answers "was it written at all", never "is it any good". `TBD` passes.
- **Making `needs:` explicitly answered.** The honest fix needs a `needs: none` token and puts fifteen shipped docs in breach — BACKLOG.
- **Repo-wide secret scanning.** 0008's boundary holds; `BACKLOG.md` was simply missed.

## Design
**The template teaches in comments, because comments are already stripped.**
`ParseSections` and `forEachBox` never see them, so guidance inside `<!-- -->` reaches the
author and no count. The visible placeholders move inside them — `## Tasks` loses its two
`- [ ] …` boxes, so a fresh doc genuinely has zero and an unplanned feature reads as
unplanned. This is what lets the check below be exact rather than fuzzy.

**`unfilled-template`, HIGH**: a non-Draft doc with an empty Problem, Design or Tasks.
HIGH because `claim` does not consult `audit`, so `--strict` is the only thing between an
unwritten doc and an agent acting on it. It cannot misfire once guidance is in comments, and
Draft is exempt because an unfinished draft is the normal state of a draft.

**`missing-section` gets one job** — `## Log`, and only at Shipped, since an Accepted
feature has nothing to log yet. `no-tasks` keeps the narrower case of prose with no boxes,
so one defect never produces two findings.

**Staleness reads the marker**, because one severity cannot serve both:

| check | fires on | after | severity |
|---|---|---|---|
| `inbox-stale` | a BLOCKING question | 2 days | HIGH |
| `fyi-stale` | an FYI note | 14 days | LOW |

Two days rather than one because the format carries a date and no time, so a one-day fuse
can elapse an hour after filing.

## Tasks
- [x] Move every visible placeholder in `templates/features-TEMPLATE.md` into its HTML
      comment, and assert a fresh `kit new` doc reports `0` task boxes and empty
      Problem/Design/Log sections.
- [x] Add `unfilled-template` at HIGH over Problem/Design/Tasks for non-Draft docs;
      narrow `missing-section` to `## Log` at `Shipped`. Conformance: fires on an
      accepted untouched doc, quiet on a filled one, quiet on a Draft.
- [x] Split `inbox-stale` (BLOCKING, 2 days, HIGH) from `fyi-stale` (FYI, 14 days, LOW),
      with a test that an FYI line does not fail `--strict`.
- [x] Add `BACKLOG.md` to the secret scan and point `done-not-shipped` at `kit ship`;
      update the check list in README.md.

## Log
- 2026-08-14 — dropped the "still matches the template's text" half of `unfilled-template`
  before writing it: with guidance in comments the template parses as empty too, so it could
  only restate the emptiness check. A hand-copied template is caught by `id-mismatch`.
