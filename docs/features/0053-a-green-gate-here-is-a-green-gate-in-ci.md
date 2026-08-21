---
id: 0053
title: A green gate here is a green gate in CI
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs:
---

## Problem

The protocol tells an agent to run `make check` before shipping and the as-built doc calls
it "the whole gate". CI runs three more things, so an agent can pass the documented gate,
ship, and turn `main` red — which is the state `main` is in now.

## Non-goals

- **Making the local gate need the network.** `make vuln` and the audit that needs a remote
  stay CI's, and the split is stated rather than implied.
- **New linters for their own sake.** The set is what CI already runs, pinned so it cannot
  change underfoot.

## Design

**`make check` runs what CI runs, minus what needs the network.** Today CI adds
`golangci-lint`, `make vuln` and `kit audit --strict`; `make check` is `fmt-check vet test`.
The lint moves into `make check`, and CI becomes `make check` plus the two network steps, so
there is one list and it lives in the Makefile.

**The linter is pinned, and its configuration is committed.** `golangci-lint-action` runs
`version: latest`, so a release of the linter turns `main` red with no commit — and the
repo carries no `.golangci.yml`, so the enabled set is whatever that version defaults to.
Both become files.

**A green `make check` is a green CI, and a test proves it.** The two lists drift silently
because nothing compares them; the workflow and the Makefile are checked against each other.

**A job that hangs is a job that fails.** The workflow sets no `timeout-minutes`, so a hung
step burns the default six hours.

**Coverage is measured or it is not claimed.** The conformance suite drives a separately
built binary, so `go test -cover` reports 0.8% and 12.6% and the as-built doc explains the
number away. Build the conformance binary with `-cover` and merge with `go tool covdata`,
or stop printing a number nobody can act on.

The failing lint on `main` is one deprecation, `parser.ParseDir` in the test that bans rule
numbers — introduced by the round that added it, and invisible to `make check`.

## Tasks

- [x] `make check` runs the lint, and the deprecation that has `main` red is fixed
- [x] `.golangci.yml` is committed and the action pins a version
- [x] CI is `make check` plus the network steps, and a test compares the two lists
- [x] The workflow sets `timeout-minutes`
- [x] Coverage is either instrumented through the conformance binary or no longer reported

## Log
- 2026-08-21 — an absent `golangci-lint` skips with a notice rather than failing, which is
  what the Makefile `kit init` writes already does: a fresh clone goes green with only Go
  present. CI installs the linter, so the skip cannot hide a finding there.
- 2026-08-21 — the coverage number is dropped rather than instrumented. Instrumenting is
  real — `go build -cover` on the suite's binary, `GOCOVERDIR` per subprocess, `go tool
  covdata` to merge — but it buys a number no threshold reads, and two documents existed
  only to explain the old one away. Both are gone with it.
