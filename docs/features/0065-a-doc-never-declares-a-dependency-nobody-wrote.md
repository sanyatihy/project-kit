---
id: 0065
title: A doc never declares a dependency nobody wrote
status: Draft
created: 2026-08-23
shipped:
needs: 0064
---

## Problem

A frontmatter `#` is a comment only when two or more spaces set it off — 0051 chose that
so `title: Fix #123 crash` keeps its whole title. One space and the `#` is the value, and
`ParseIDList` scans it for `\b\d{4}\b`:

```
ParseIDList("# feature ids that must ship first, e.g. 0002, 0003. Empty = can start now")
→ [0002 0003]
```

The template writes that comment on the `needs:` line, padded wide enough to be stripped.
One deleted space and a doc declares `needs: 0002, 0003` that nobody wrote, and nothing
writes `needs:` afterwards, so it survives into frozen history. `needs-unknown` cannot see
it, because the scanned ids exist and have shipped: the wait reads as satisfied. A project
whose 0002 is still open gets a feature parked in `waiting` for a prerequisite nobody
declared. The same hole passes `needs: TBD` and `needs: after the router lands`.

## Non-goals

- **Keeping the guidance.** It is a third copy. The HTML block directly below the
  frontmatter spends ten lines on how to judge `needs:`, `kit help template` prints both,
  and protocol rule 6 says who writes `status:` and `shipped:`.
- **Reporting a terminal doc.** `needsState` already skips those, and says why: a retired
  doc waits for nothing, and the only fix would be an edit to frozen history, so it
  reported for ever.
- **Guessing whether the declared ids are the right ones.** Unchanged from
  `needs-unknown`: no heuristic has that judgement.
- **Repairing the sibling project.** localcode fixes its own docs once it installs this.

## Design

**The frontmatter carries no prose.** Every value on it is machine-read, so guidance there
is one deletion away from becoming data. The three comments come off `templates/features-TEMPLATE.md`,
and a test holds the embedded template's frontmatter to id, title, status, created,
shipped and needs with nothing after the value. That is the refusal; the check below is
what catches the docs already written.

**`needs-unparsed` reports a `needs:` that is not an id list.** MEDIUM, beside
`needs-unknown`, whose blind spot it is: the leaked value names ids that exist and have
shipped, so nothing today can see it. A `needs:` is ids, commas and spaces — anything else
in it was not written as a dependency, which is why the check cannot fire on a healthy
doc. It takes `needsState`'s terminal gate for the reason that function gives.

**It waits on 0064.** The last box reads the trunk, which 0064 is still changing.

## Tasks

- [x] The template's frontmatter carries no comment, and a test holds it to that
- [x] `audit` reports a `needs:` carrying text that is not an id list, skipping terminal
      docs, with a conformance case
- [x] The docs on the trunk pass it

## Log

- 2026-08-23 — the premise was too strong and `## Problem` is rewritten. The template's
  own padding is two or more spaces, so `trailingRe` strips it: every doc here parses
  `needs:` to what its author wrote, 0063 and 0064 included. What is true is that one
  space instead of two makes the comment the value, which is what the deletion in box 1
  and the check in box 2 close.
- 2026-08-23 — no doc on the trunk fires `needs-unparsed`, which the corrected premise
  predicts. What box 3 left is `docs/features/TEMPLATE.md`, a copy nothing reads: its
  frontmatter comes into line here, and whether the file should exist is a backlog line.
