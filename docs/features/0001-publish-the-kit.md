---
id: 0001
title: Publish the kit so go install works
status: Shipped
created: 2026-08-11
shipped: 2026-08-11
related: ../../README.md
---

## Problem
The documented install path is `go install github.com/sanyatihy/project-kit/cmd/kit@latest`,
and nothing is committed, there is no remote and no tag, so it cannot resolve. Every project
scaffolded with `--stack go` runs it in CI.

## Non-goals
- A release process beyond a tag: no goreleaser, no signed artefacts, no homebrew tap.
- Publishing anywhere but the module proxy.

## Design
Commit, create the remote, push, tag. `@latest` resolves to the highest semver tag, so an
untagged repo pins every consumer to a moving default branch.

The repo is **private**, so the proxy cannot serve it: every consumer needs
`GOPRIVATE=github.com/sanyatihy/*` and credentials for the host. That turns two
documentation tasks into part of publishing, and it is why generated CI does not run
`kit audit --strict` — every scaffolded repo would otherwise carry a token to rotate and
fail its first push. The kit's own CI keeps the audit: `go run ./cmd/kit` authenticates to
nothing.

The kit's own workflow lands last, because it installs the kit and cannot work until the
module resolves.

## Tasks
- [x] Commit the tree and create the private `sanyatihy/project-kit` remote, then push
- [x] Document `GOPRIVATE` in the README install section — a private module needs it on every machine
- [x] Verify `go install github.com/sanyatihy/project-kit/cmd/kit@latest` from a clean `GOMODCACHE` with `GOPRIVATE` set
- [x] `kit version` reports the module version from build info, not `dev`, when installed with `go install`
- [x] Tag `v0.1.0` so `@latest` pins a release rather than the branch
- [x] Drop `kit audit --strict` from the generated `--stack go` `check.yml` so scaffolding needs no credential
- [x] Add the kit's own `.github/workflows/check.yml` running `make check`

## Log
- 2026-08-11 — private repo: added the README and generated-CI tasks that carry its cost.
- 2026-08-11 — `go install` does not apply the Makefile's `-ldflags`, so the binary reported
  `dev`. Reading the module version from build info was added before the tag.
