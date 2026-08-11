---
id: 0004
title: Close the go-checklist findings
status: Accepted
created: 2026-08-11
shipped:
check:
checked:
related: ./BACKLOG.md
---

## Problem
Reviewing the kit against `kit help go-checklist` (0003) turned up four things and
filed them in `BACKLOG.md`, unscheduled. Two have since acquired a signal rather than
just being true: 0002 shipped a loose date assertion because `NewFeature` reads the
clock inline, and the conformance suite now pins exit codes that are nowhere
documented, so the suite is the only statement of the contract. The other two —
error context and build hardening — were promoted because they were asked for.

Guidance the kit ships and does not itself follow is guidance nobody else should
take seriously.

## Non-goals
- **Chasing the whole checklist.** Three items were considered and rejected during
  the review as thumping; that reasoning is in `BACKLOG.md` and stands.
- **Restructuring for testability beyond the one seam.** `NewFeature` gets a clock
  because a test needs it. Nothing else grows an interface on speculation.
- **A coverage target.** Same non-goal as 0002, same reason.

## Design
Four independent changes, in the order their dependencies run.

**The clock** is the only one with a knock-on. `Audit` already takes `today` as a
parameter and `NewFeature` does not, which is why one is exactly testable and the
other is not. Passing the date in from `main` matches the shape already there rather
than inventing a `Clock` interface for a single caller. Once it lands, the loose
date assertion in the `new` conformance suite tightens to an exact one.

**Error context** is `fmt.Errorf("...: %w", err)` at each boundary that currently
returns a bare `err`. The failure to fix is `kit init` reporting `permission denied`
without naming the file it could not write. `%w` rather than `%v` so `errors.Is/As`
keep working.

**Exit codes** are already a contract — `TestExitCodesAreDistinct` pins 0, 1, and 2 —
but nothing tells a user what they mean. This documents what the suite already
enforces, in `kit help` and the README, and adds nothing new.

**Build hardening** is three unrelated gaps that share a home in the build: a
`toolchain` line in `go.mod`, `govulncheck` in CI, and `CGO_ENABLED=0 -trimpath
-ldflags "-s -w"` for reproducible binaries.

## Tasks
- [x] Pass the date into `NewFeature` instead of reading the clock, and tighten the conformance date assertion
- [ ] Wrap errors with the operation that failed, so a failure names the file
- [ ] Document the exit-code contract in `kit help` and the README
- [ ] Harden the build: `toolchain` in go.mod, `govulncheck` in CI, reproducible build flags

## Open questions
<!-- none -->

## Log
- 2026-08-11 — promoted from `BACKLOG.md`, where these sat unscheduled since 0003.
  Two had acquired a real signal: the clock forced a workaround in 0002, and the
  conformance suite became the only place the exit-code contract is written down.
- 2026-08-11 — passing `today` in rather than introducing a `Clock` interface. The
  kit already does exactly this for `Audit`; a second shape for the same problem
  would be the pre-abstraction the checklist warns about.
