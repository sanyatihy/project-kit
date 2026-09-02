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
careful author does first — and that is the only finding.

One cause for both halves. `missing-section` tests whether a section is empty, and the
template fills every section with placeholder prose, two `- [ ] …` boxes and a fake dated
log entry, so the untouched doc looks complete and the tidied one looks broken. That is
0005's failure inverted, and worse: the lazy path is green.

Two smaller ones found in the same pass. `--fyi` does not mean FYI to `audit` —
`inbox-stale` ignores the marker, is HIGH, and at a 1-day threshold on a date-only line a
note filed at 23:00 fails `--strict` an hour later. And `BACKLOG.md` is not scanned for
secrets, though the protocol sends discovered work there.

## Non-goals

- A prose-quality heuristic. This answers "was it written at all", never "is it any good".
- Making `needs:` explicitly answered. The honest fix needs a `needs: none` token and puts
  fifteen shipped docs in breach.
- Repo-wide secret scanning. 0008's boundary holds; `BACKLOG.md` was missed.

## Design

The template teaches in comments, because comments are already stripped: `ParseSections`
and `forEachBox` never see them, so guidance inside `<!-- -->` reaches the author and no
count. The visible placeholders move inside them, so a fresh doc genuinely has zero boxes
and an unplanned feature reads as unplanned. That is what lets the check be exact rather
than fuzzy.

`unfilled-template` is HIGH on a non-Draft doc with an empty Problem, Design or Tasks. HIGH
because `claim` does not consult `audit`, so `--strict` is the only thing between an
unwritten doc and an agent acting on it. Draft is exempt, an unfinished draft being the
normal state of a draft.

`missing-section` gets one job — `## Log`, at Shipped only, since an Accepted feature has
nothing to log yet. `no-tasks` keeps the narrower case of prose with no boxes, so one defect
never produces two findings.

Staleness reads the marker, because one severity cannot serve both:

| check | fires on | after | severity |
|---|---|---|---|
| `inbox-stale` | a BLOCKING question | 2 days | HIGH |
| `fyi-stale` | an FYI note | 14 days | LOW |

Two days rather than one because the format carries a date and no time, so a one-day fuse
can elapse an hour after filing.

## Tasks

- [x] Every visible placeholder in `templates/features-TEMPLATE.md` moves into its HTML
      comment, and a fresh `kit new` doc reports zero boxes and empty sections
- [x] `unfilled-template` at HIGH over Problem/Design/Tasks for non-Draft docs, and
      `missing-section` narrowed to `## Log` at Shipped
- [x] `inbox-stale` (BLOCKING, 2 days, HIGH) split from `fyi-stale` (FYI, 14 days, LOW)
- [x] `BACKLOG.md` joins the secret scan, and `done-not-shipped` points at `kit ship`

## Log

- 2026-08-14 — dropped the "still matches the template's text" half of `unfilled-template`
  before writing it. With guidance in comments the template parses as empty too, so it could
  only restate the emptiness check, and a hand-copied template is caught by `id-mismatch`.
