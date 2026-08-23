---
id: 0065
title: A doc never declares a dependency nobody wrote
status: Draft
created: 2026-08-23
shipped:
needs: 0064
---

## Problem

The template puts its guidance for `needs:` in a trailing `#` comment on the frontmatter
line, and 0051 stopped the parser truncating a value there. So the comment is the value,
and `ParseIDList` scans it for `\b\d{4}\b`:

```
ParseIDList("  # feature ids that must ship first, e.g. 0002, 0003. Empty = can start now")
→ [0002 0003]
```

A drafter who leaves the line alone declares `needs: 0002, 0003`. `kit ship` rewrites
`status:` and `shipped:`, dropping their comments, but nothing ever writes `needs:` — so
that one survives into frozen history. It is in 0063 and 0064 here, and in four of
localcode's docs. Nothing reports it: both projects have shipped a 0002 and a 0003, so the
wait is already satisfied. A younger project gets `needs-unknown` and a `claim` that
refuses; one whose 0002 is still open gets a feature parked in `waiting` for a
prerequisite nobody wrote.

## Non-goals

- **Keeping the guidance.** It is a third copy. The HTML block directly below the
  frontmatter spends ten lines on how to judge `needs:`, `kit help template` prints both,
  and protocol rule 6 says who writes `status:` and `shipped:`.
- **Reporting a terminal doc.** `needsState` already skips those, and says why: a retired
  doc waits for nothing, and the only fix would be an edit to frozen history, so it
  reported for ever. 0063 is exactly that case.
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

**It waits on 0064.** The last box clears the trunk, and the only doc on it that fires is
0064's, which is claimed.

## Tasks

- [x] The template's frontmatter carries no comment, and a test holds it to that
- [ ] `audit` reports a `needs:` carrying text that is not an id list, skipping terminal
      docs, with a conformance case
- [ ] The docs on the trunk pass it

## Log
