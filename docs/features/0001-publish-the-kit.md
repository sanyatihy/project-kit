---
id: 0001
title: Publish the kit so go install works
status: In progress
created: 2026-08-11
shipped:
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
- [ ] Commit the tree and create the private `sanyatihy/project-kit` remote, then push
- [x] Document `GOPRIVATE` in the README install section — a private module needs it on every machine
- [ ] Verify `go install github.com/sanyatihy/project-kit/cmd/kit@latest` from a clean `GOMODCACHE` with `GOPRIVATE` set
- [ ] Tag `v0.1.0` so `@latest` pins a release rather than the branch
- [ ] Teach the generated `--stack go` `check.yml` to install the kit from a private module
- [ ] Add the kit's own `.github/workflows/check.yml` running `make check`

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
