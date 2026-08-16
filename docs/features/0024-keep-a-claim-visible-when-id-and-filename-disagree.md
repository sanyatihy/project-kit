---
id: 0024
title: Keep a claim visible when id and filename disagree
status: Draft
created: 2026-08-16
shipped:
check:
checked:
needs: 0023, 0025
related: 0006-make-the-kit-safe-for-concurrent-agents.md
---

## Problem
A doc whose `id:` disagrees with its filename makes its own claim invisible, and the
board goes on handing the feature out. `audit` calls that `id-mismatch` and rates it
MEDIUM — a tidiness defect. It is not: it breaks the one guarantee the concurrency
model rests on.

Measured, with `docs/features/0002-confused.md` declaring `id: 0009`:

```
$ kit claim 0009    claimed  0002-confused                     (exit 0)
$ kit next          start here  0009  →  kit claim 0009
$ kit claim 0009    kit claim: git branch: fatal: a branch named
                    '0002-confused' already exists              (exit 2)
```

`Branch()` derives the branch from the filename; `Claims(id)` matches branch names
that mention the `id:`. `0002-confused` does not contain `0009`, so nothing connects
them. The first agent holds the feature, the board says it is free, and the second
agent gets **exit 2** — the code documented as "it could not run" — with a raw git
error. An agent reading the contract concludes its invocation is broken, not that the
work is taken, so it escalates or retries instead of moving on.

`duplicate-id` is HIGH because the ID is the primary key. This is the same key
disagreeing with itself and it is a severity lower.

`needs: 0023, 0025` is file contention, not logic: 0023 rewrites the same refusal
block in `ClaimFeature` and 0025 adds a check to the same registry in `audit.go`.
Judged by whether the work could *start* now, it cannot without conflicting.

## Non-goals
- **Renaming files.** The kit does not move a project's docs; the fix is a report the
  human acts on.
- **Making the filename authoritative, or the `id:`.** Either choice silently changes
  which doc a branch or an inbox line refers to. The disagreement is the defect.
- **Reworking exit codes generally.** Only this path is wrong, and it is wrong
  because it reaches git at all.

## Design
**`id-mismatch` goes to HIGH.** `--strict` then fails on it, which is the point: it
cannot be left sitting while agents run. Its fix line already says what to do.

**`claim` refuses a mismatched doc** before touching git, as `NotYours`. It cannot
create a claim the board will not see, and the mismatch is repairable in one rename —
the same shape as the `duplicate-id` refusal directly above it, and for the same
reason.

That refusal also removes the raw-git exit 2: the second agent now gets the message
its predecessor should have.

**`Claims` matches the filename prefix as well as the `id:`.** Belt and braces for
docs already in this state — a claim made before the fix stays visible to the board
rather than being orphaned by it. `FilenamePrefix` exists and is already used by
`NextID`, so this is one extra comparison.

## Tasks
- [x] `id-mismatch` raised to HIGH, with its `--strict` behaviour covered
- [x] `ClaimFeature` refuses a doc whose `id:` and filename prefix disagree, as
      `NotYours` and before any git call; conformance case asserting exit 1
- [x] `Claims` matches the filename prefix as well as the declared id, so an existing
      mismatched claim is still seen; conformance case
- [x] A conformance case for the whole scenario: claim, then the board must not offer
      the same feature again
- [ ] README's check list and `kit help checks` reflect the new severity
- [ ] `next` blocks a mismatched doc instead of offering it, so the board and `claim`
      agree — the loop 0013 exists to stop

## Log
- 2026-08-16 — found probing the lifecycle after the url-shortener bootstrap.
- 2026-08-16 — the refusal leaves `next` offering a feature `claim` turns away, which
  is the 0013 loop. Appended as a box rather than widening this change.
