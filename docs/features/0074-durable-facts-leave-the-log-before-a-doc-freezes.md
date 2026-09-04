---
id: 0074
title: Durable facts leave the Log before a doc freezes
status: Shipped
created: 2026-09-03
shipped: 2026-09-04
needs:
---

## Problem

A feature in a sibling project shipped at 486 lines with a `## Log` of 4,599 words — 79% of
the document. The entries state how a currency is decided, what a commission is written as,
and the two constants its guards compare against: facts true after the feature ships, which
the routing rule sends to `docs/TECH.md` one line each. None reached that file. The doc is
Shipped, so the rule can no longer be applied to it.

The Log grew about 100 words per commit across 45 commits, and every entry passes every rule
the kit has, because each of them tests one entry. The routing rule is the one that would
catch it, and `kit ship` prints "move durable facts into docs/TECH.md, one line each" while
verifying nothing. Two cleanups by hand halved the doc and neither held; the second edited a
Shipped doc.

0038 shipped `doc-bloat` for this and 0049 deleted it, for an escape hatch that existed in no
line of code rather than because length stopped mattering. Nothing replaced it.

## Non-goals

- Judging an entry. The test is the section's weight, not whether a sentence earns its place.
- An absolute word count. Measured over three corpora, 400 words fires on 13 of one
  project's 50 docs, and a check that fires on healthy projects is worse than no check.
- Reporting a terminal doc. 0049 deleted five checks for firing where rule 17 forbids the fix.
- A `--force`. The escape is moving the facts, which is the action wanted.
- Bounding entries per commit. One box, one commit, one thing learned is the honest rate;
  what is wrong is where the learning is filed.
- A check on `## Design`. Measured over the same 140 docs, Design runs 45 words per box here
  at the median and 185 at another project's worst, so no threshold separates a doc that
  restates its Tasks from one whose single box needs the room.
- A rule against a doc that is a bucket of unrelated fixes. Rule 2 already refuses it by
  name, and a second statement is the defect this round exists to remove.

## Design

The `## Log` may not outweigh the rest of the doc. Measured 2026-09-03 over 140 docs in three
projects: the Log is 12% of a doc at the median here and 26% at its worst, 22% and 53% in
`localcode`, 18% and 54% in `financials`. A ceiling at half fires on 0 of 73 here, 2 of 17
and 1 of 50 — the three docs whose Log is the as-built doc in disguise.

A ratio and not a word count, so it needs no recalibration as a corpus matures and it does
not punish a doc with a long `## Design`. An absolute 400 words fires on 13 of 50.

`ship` refuses, because that is the last moment the doc is editable. This is the fifth thing
it refuses and the first about what the doc says rather than what the repo holds.

`audit` reports the same on a non-terminal doc, where the fix is still legal, so a Draft
growing mid-work is visible before anyone reaches `ship`. Terminal docs are skipped for
0049's reason.

Two writing rules join the protocol, both found by reading docs written under the current
set. A bold lead-in opening a bullet or a paragraph restates the sentence after it: 374 came
out of this corpus under 0073 and the rule was never written down, so nine are in one doc
drafted since. And a `## Design` paragraph per `## Tasks` box is the Tasks list restated —
the same fact in two sections, which is what `## Design` is not for.

## Tasks

- [x] `Ship` refuses when `## Log` outweighs the rest of the doc, naming `docs/TECH.md` and
      the routing rule
- [x] `log-outweighs-plan` reports it on a non-terminal doc, skipping terminal ones
- [x] The ratio, its sample and its date sit in the constant's comment
- [x] `kit help ship`, `kit help checks` and `docs/TECH.md` carry it
- [x] `### How to write` refuses a bold lead-in, and the feature template's `## Design`
      guidance says a paragraph per box is the Tasks list restated

## Log
