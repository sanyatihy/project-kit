---
id: 0024
title: Keep a claim visible when id and filename disagree
status: Shipped
created: 2026-08-16
shipped: 2026-08-16
needs: 0023, 0025
related: 0006-make-the-kit-safe-for-concurrent-agents.md
---

## Problem

A doc whose `id:` disagrees with its filename makes its own claim invisible, and the board
goes on handing the feature out. `audit` rates that MEDIUM, a tidiness defect. It breaks the
one guarantee the concurrency model rests on.

Measured, with `docs/features/0002-confused.md` declaring `id: 0009`:

```
$ kit claim 0009    claimed  0002-confused                     (exit 0)
$ kit next          start here  0009  →  kit claim 0009
$ kit claim 0009    kit claim: git branch: fatal: a branch named
                    '0002-confused' already exists              (exit 2)
```

`Branch()` derives the branch from the filename and `Claims(id)` matches branch names
mentioning the `id:`, so nothing connects them. The first agent holds the feature, the board
says it is free, and the second gets exit 2 — the code documented as "it could not run" —
with a raw git error, so it escalates or retries instead of moving on.

`duplicate-id` is HIGH because the id is the primary key. This is the same key disagreeing
with itself, a severity lower.

`needs: 0023, 0025` is file contention: all three touch the same refusal block and check
registry.

## Non-goals

- Renaming files. The kit does not move a project's docs.
- Making the filename authoritative, or the `id:`. Either choice silently changes which doc
  a branch or an inbox line refers to; the disagreement is the defect.
- Reworking exit codes generally. Only this path is wrong, and because it reaches git at all.

## Design

`id-mismatch` goes to HIGH, so `--strict` fails on it and it cannot be left sitting while
agents run.

`claim` refuses a mismatched doc before touching git, as `NotYours`. It cannot create a
claim the board will not see, and the mismatch is repairable in one rename — the same shape
as the `duplicate-id` refusal directly above it. That also removes the raw-git exit 2.

`Claims` matches the filename prefix as well as the `id:`, so a claim made before the fix
stays visible rather than being orphaned by it. `FilenamePrefix` already exists for
`NextID`, so this is one extra comparison.

## Tasks

- [x] `id-mismatch` raised to HIGH, with its `--strict` behaviour covered
- [x] `ClaimFeature` refuses a doc whose `id:` and filename prefix disagree, as `NotYours`
      and before any git call
- [x] `Claims` matches the filename prefix as well as the declared id
- [x] A conformance case for the whole scenario: claim, then the board must not offer the
      same feature again
- [x] README's check list and `kit help checks` reflect the new severity
- [x] `next` blocks a mismatched doc instead of offering it, so the board and `claim` agree

## Log

- 2026-08-16 — the refusal alone leaves `next` offering a feature `claim` turns away, which
  is the loop 0013 exists to stop. Appended as a box rather than widening the change.
