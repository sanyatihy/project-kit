---
id: 0054
title: Latest is the kit you built
status: Draft
created: 2026-08-21
shipped:
needs: 0053
---

## Problem

The README tells you to install with `go install …@latest`. That resolves to `v0.2.0`, which
is 192 commits behind `main` and old enough that `internal/kit/release.go` does not exist in
it. Anyone following the documented instruction today gets a kit without `kit release` and
with every bug the last two rounds fixed.

## Non-goals

- **A changelog written by hand.** The feature docs are the record; a release names the
  range, it does not restate it.
- **Publishing anywhere.** The module stays private and `go install` stays the delivery.
- **Automatic tagging on merge.** What to release is a human call.

## Design

**A tag is what `@latest` means, so cutting one is part of shipping.** Nothing in the
protocol, the Makefile or CI mentions tagging, and the gap has grown to 192 commits without
anything reporting it. `make release` cuts an annotated tag after running the gate, and the
protocol names it as the step after a round lands.

**The version the binary prints is the version you can install.** `kit version` reports
`v0.2.0-192-ge1ebe92` from `git describe`, which is honest about the source and silent about
the fact that no such thing is installable. When the tag is behind the trunk by more than a
release, say so.

**A stale tag is drift, and `audit` reports drift.** Everything else that can fall behind
here is a finding — the protocol, the as-built doc, the board. The distance between the
newest tag and the trunk is the same kind of fact.

Measured today: `v0.2.0`, 192 commits behind, predating the file that holds the command the
README advertises.

## Tasks

- [ ] `make release` runs the gate and cuts an annotated tag
- [ ] The protocol names tagging as what follows a round landing
- [ ] `audit` reports a tag far enough behind the trunk to mislead `@latest`
- [ ] A release is cut for the current trunk, and `go install @latest` is verified against it

## Log
