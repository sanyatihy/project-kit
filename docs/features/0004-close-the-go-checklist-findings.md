---
id: 0004
title: Close the go-checklist findings
status: Shipped
created: 2026-08-11
shipped: 2026-08-11
check:
checked:
related: ./BACKLOG.md
---

## Problem
Two of 0003's four backlog findings acquired a signal: 0002 shipped a loose date assertion
because `NewFeature` reads the clock inline, and the conformance suite became the only
statement of the exit-code contract. Guidance the kit ships and does not follow is guidance
nobody else should take seriously.

## Non-goals
- **Chasing the whole checklist.** Three items were rejected as thumping in 0003 and stay rejected.
- **Restructuring for testability beyond the one seam.** `NewFeature` gets a date parameter because a test needs it.

## Design
**The clock** is passed in from `main`, matching `Audit`'s existing shape rather than
inventing a `Clock` interface for one caller. The loose date assertion then tightens.

**Exit codes** are already pinned by `TestExitCodesAreDistinct`; this documents what the
suite enforces and adds nothing new.

**Build hardening** is three gaps sharing a home: a `toolchain` line, `govulncheck` in CI,
and `CGO_ENABLED=0 -trimpath -ldflags "-s -w"` for reproducible binaries.

## Tasks
- [x] Pass the date into `NewFeature` instead of reading the clock, and tighten the conformance date assertion
- [x] Document the exit-code contract in `kit help` and the README
- [x] Harden the build: `toolchain` in go.mod, `govulncheck` in CI, reproducible build flags

## Log
- 2026-08-11 — error-context box dropped: its premise was false. Every bare `return err`
  originates in an `os`/`fs` call returning `*PathError`, which already names the path and
  the operation, so wrapping would repeat it. Reopen if a failure is ever hard to read.
- 2026-08-11 — checked `-s -w` against `debug.ReadBuildInfo`, which `kit version` needs for
  network installs: module info survives stripping and two builds are byte-identical.
