---
id: 0054
title: Installing the kit gets you the kit you built
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs: 0053
---

## Problem

The README tells you to install with `go install …@latest`. That resolves to `v0.2.0`, which
is 192 commits behind `main` and old enough that `internal/kit/release.go` does not exist in
it. Anyone following the documented instruction today gets a kit without `kit release` and
with every bug the last two rounds fixed.

## Non-goals

- A changelog written by hand. The feature docs are the record; a release names the
  range, it does not restate it.
- Publishing anywhere. The module stays private and `go install` stays the delivery.
- Automatic tagging on merge. What to release is a human call.

## Design

A tag is what `@latest` means, so cutting one is part of shipping. Nothing in the
protocol, the Makefile or CI mentions tagging, and the gap has grown to 192 commits without
anything reporting it. `make release` cuts an annotated tag after running the gate, and the
protocol names it as the step after a round lands.

The version the binary prints is the version you can install. `kit version` reports
`v0.2.0-192-ge1ebe92` from `git describe`, which is honest about the source and silent about
the fact that no such thing is installable. When the tag is behind the trunk by more than a
release, say so.

A stale tag is drift, and `audit` reports drift. Everything else that can fall behind
here is a finding — the protocol, the as-built doc, the board. The distance between the
newest tag and the trunk is the same kind of fact.

Measured today: `v0.2.0`, 192 commits behind, predating the file that holds the command the
README advertises.

## Tasks

- [x] `make release` runs the gate and cuts an annotated tag
- [x] The protocol names tagging as what follows a round landing
- [x] `audit` reports a tag far enough behind the trunk to mislead `@latest`
- [x] A release is cut for the current trunk, and `go install @latest` is verified against it

## Log
- 2026-08-21 — the protocol names the act and not `make release`. The Go stack `kit init`
  scaffolds already ships a `release` target that builds a static binary, so a generated
  rule naming the command would mean the wrong thing in every project the kit writes. The
  command is named in this repo's own `## Changing the kit`, beside `make check`.
- 2026-08-21 — v0.3.0 names a commit that does not contain the target that cut it.
  `make release` refuses a HEAD that is not the trunk's tip, and the target ships on this
  branch, so the first release was cut from a detached checkout of the trunk with the
  target passed by `make -f`. Every release after this one finds it on the trunk.
