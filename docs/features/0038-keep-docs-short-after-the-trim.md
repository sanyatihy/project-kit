---
id: 0038
title: Keep docs short after the trim
status: Shipped
created: 2026-08-17
shipped: 2026-08-18
needs: 0037
related: 0037-rules-that-produce-concise-docs.md
---

## Problem
0037 trims this repo once. Nothing stops the same bloat returning here or arriving in any
other project: the guidance that failed to prevent it is prose in a template, and a
template only reaches an existing project when somebody re-runs `kit init`. A check in the
binary reaches every project at the next upgrade.

## Non-goals
- Blocking a ship. Length is hygiene, not drift. LOW, never HIGH, so `--strict` stays
  about things that are untrue.
- Judging quality. The check counts lines. Whether a doc is *good* is review's job.

## Design
`doc-bloat`, LOW. Fires on a feature doc past N lines, or a `## Log` entry past two.
The fix line offers both ways out — trim it, or say in the `## Log` why it earns the
length — so a genuinely dense doc is a sentence away from quiet rather than a lie away.

N is measured, not guessed. After 0037, N is the trimmed corpus's 90th percentile
rounded up, recorded in the constant's comment with the date and the sample. A number
picked before the trim would only encode what the docs happen to weigh today.

`kit ship` prints one line, at the moment a doc stops being a working document. The
wording leads with what to keep, because the risk is an agent reading "compact" as
"delete":

```
  keep every decision, constraint and measurement — cut only what the code now says
```

That is the whole instruction. It names no target and no number: a doc that is already
short passes it untouched, and a doc that is long because it decided a lot stays long.

## Tasks
- [x] `doc-bloat` at LOW over a feature doc's length and its `## Log` entries, with N taken
      from the trimmed corpus and the sample in the comment; conformance both ways
- [x] `kit ship` prints the compaction line, and `kit help ship` says what it means
- [x] README and `kit help checks` carry the new code

## Log
- 2026-08-17 — split from 0037 so the threshold can be measured from trimmed docs rather
  than guessed from bloated ones.
