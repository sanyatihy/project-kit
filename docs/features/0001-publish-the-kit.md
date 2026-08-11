---
id: 0001
title: Publish the kit so go install works
status: Shipped
created: 2026-08-11
shipped: 2026-08-11
check:
checked:
related: ../../README.md
---

## Problem
The kit is a Go module whose documented install path is
`go install github.com/sanyatihy/project-kit/cmd/kit@latest`, but nothing is
committed and there is no remote, so that command cannot resolve. Every project
scaffolded with `--stack go` gets a CI workflow that runs it, so their pipelines
will fail on the first push.

## Non-goals
- A release process beyond a tag: no goreleaser, no signed artefacts, no homebrew tap.
- Publishing to any registry other than the module proxy.
- Making the repo public if it should stay private — either works for `go install`
  with the right `GOPRIVATE`.

## Design
Commit the current tree, create the remote at `github.com/sanyatihy/project-kit`,
push, then tag. `@latest` resolves to the highest semver tag, so an untagged repo
would pin every consumer to a moving default branch; tagging is what makes the
documented command stable.

The repo is **private**, so the module proxy cannot serve it: every consumer needs
`GOPRIVATE=github.com/sanyatihy/*` and credentials Git can use for that host. That is
one line in a developer's environment, but in CI it is a checkout token plus the same
`GOPRIVATE`, and it applies to *every* project scaffolded with `--stack go` — the
generated `check.yml` installs the kit to run `kit audit --strict`. So the private
choice turns two documentation tasks into part of publishing, not follow-up work.

The kit's own CI is deliberately added last: the generated workflow installs the kit
to run `kit audit --strict`, which cannot work until the module resolves. Ordering
the tasks this way keeps every intermediate state green.

## Tasks
- [x] Commit the tree and create the private `sanyatihy/project-kit` remote, then push
- [x] Document `GOPRIVATE` in the README install section — a private module needs it on every machine
- [x] Verify `go install github.com/sanyatihy/project-kit/cmd/kit@latest` from a clean `GOMODCACHE` with `GOPRIVATE` set
- [x] `kit version` reports the module version from build info, not `dev`, when installed with `go install`
- [x] Tag `v0.1.0` so `@latest` pins a release rather than the branch
- [x] Drop `kit audit --strict` from the generated `--stack go` `check.yml` so scaffolding needs no credential
- [x] Add the kit's own `.github/workflows/check.yml` running `make check`

## Open questions
<!-- none -->

## Log
- 2026-08-11 — Go port replaced the Python scripts; distribution is now `go install`
  rather than per-project vendoring, which is what makes publishing a prerequisite
  rather than a nicety.
- 2026-08-11 — the repo is **private**. `go install` still works, but only with
  `GOPRIVATE` and Git credentials for the host on every machine and in every CI job
  that installs the kit. Added the two tasks that carry that cost: the README note and
  the generated `check.yml`.
- 2026-08-11 — cold `go install` from an empty `GOMODCACHE` verified working against
  the private remote over the SSH rewrite. Two traps found doing it, both now in the
  README: `GOPRIVATE` is a list that an existing shell `export` silently overrides, and
  a `sum.golang.org` 404 quotes a git auth failure from *its* server, which reads like
  a local credential problem but is not.
- 2026-08-11 — the installed binary reports `kit dev`: `Version` is stamped only by the
  Makefile's `-ldflags`, which `go install` does not apply, so tagging alone will not
  make `kit version` report the tag. The module version is in the embedded build info.
  Added a task to read it from there, ordered before the tag so `v0.1.0` is the first
  release that reports itself correctly.
- 2026-08-11 — generated projects no longer run `kit audit --strict` in CI. Auditing
  elsewhere means installing a private module, so every scaffolded repo would carry a
  token to store and rotate, and its first push would fail until someone set it.
  Zero-setup scaffolding is worth more than catching drift a few hours before
  `kit next` would. The kit's own CI keeps the audit — it has the source, so
  `go run ./cmd/kit` needs nothing to authenticate. Revisit if the kit goes public;
  the two steps cost nothing then.
- 2026-08-11 — adding `golangci-lint` to the kit's CI surfaced an unchecked `Close`
  on a written file in `NewFeature`. A flush failure there would have `kit new`
  report a doc it had only half written. Fixed by reporting the `Close` error rather
  than deferring it.
- 2026-08-11 — shipped. `v0.1.0` is tagged, and a cold `go install …@latest` against an
  empty `GOMODCACHE` resolves to the tag and reports `kit v0.1.0`. Durable facts moved
  to [TECH.md](../TECH.md).
- 2026-08-11 — the work landed in two PRs, not one. The first was merged while the
  branch still had two of five commits, so `main` briefly carried a `check.yml` that
  installed a private module and a binary that reported `dev`. Nothing was lost, but it
  is why the tag was checked against `main`'s content rather than the branch's: a
  merged PR is not evidence that the branch is merged.
