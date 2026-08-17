---
id: 0034
title: Run the tests in parallel, capped
status: Draft
created: 2026-08-17
shipped:
check:
checked:
needs:
related: 0002-restore-the-conformance-suite.md
---

## Problem
`make check` takes 32.8 seconds, and almost all of it is waiting. The suite is 190 tests
summing to 28.6s of work, run one at a time on a 12-core machine — no test calls
`t.Parallel()`.

The cost is paid on every box ticked. This session ran the gate 20-odd times; at half a
minute each that is ten minutes of an agent watching a progress bar, and it is the one
delay in the loop that nothing else hides.

## Non-goals
- **Using the whole machine.** Other work runs here. The point is a faster gate, not a
  saturated box.
- **Making the tests share fixtures.** Each builds its own `t.TempDir()`, and that
  independence is what makes this safe. A shared fixture would trade the wait for
  flakiness.
- **Touching what the tests assert.** Nothing about coverage or behaviour changes.

## Design
`t.Parallel()` as the first statement of every top-level test, and a cap in the Makefile.

Measured on this repo, `go test -race ./cmd/kit`:

| `-parallel` | 2 | 3 | 4 | 6 | 12 (default) |
|---|---|---|---|---|---|
| wall clock | 22.2s | 19.0s | **17.4s** | 12.0s | 8.8s |

**4 is the default**, as `PARALLEL ?= 4` in the Makefile so an idle machine can raise it
without editing a recipe. It leaves two thirds of the cores alone and still halves the
gate. Uncapped is nearly four times faster and is deliberately not the default: the gate
runs beside whatever else the operator is doing.

Three properties make this safe, and all three are already true — this feature only has
to avoid breaking them:

- every test builds its own `t.TempDir()`; none shares a directory
- no test calls `os.Chdir` or `os.Setenv`, so there is no process-wide state to race
- the binary is built once behind `sync.Once`, which is concurrency-safe already

Verified before drafting: all 190 marked parallel, run twice under `-race`, all passing at
8.81s and 8.86s.

## Tasks
- [x] `t.Parallel()` on every top-level test in `cmd/kit` and `internal/kit`
- [x] `PARALLEL ?= 4` in the Makefile, used by the `test` target, with the measurements
      in a comment so the number can be re-judged rather than guessed
- [x] The suite passes three consecutive `-race` runs, and `docs/TECH.md` records that
      test independence is what buys this

## Log
- 2026-08-17 — measured during a code audit: 190 tests, 28.6s of work, 32.8s of wall clock.
- 2026-08-17 — 214 tests marked, not the 190 the audit measured: that count was `cmd/kit`
  alone and `internal/kit` has 24 more. `make check` went from 35s to 20s wall clock at
  `PARALLEL=4`, using 3.4 cores of 12. Three consecutive `-race` runs passed at 14.0s,
  15.2s and 14.3s, so the spread is a second either way rather than flakiness.
