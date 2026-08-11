---
id: 0002
title: Restore the conformance suite
status: In progress
created: 2026-08-11
shipped:
check:
checked:
related: ../../AGENTS.md
---

## Problem
`./test` — the suite that built the binary and drove `init`, `next`, and `audit`
from the outside — was deleted while untracked, so it is unrecoverable. It was the
behavioural specification: the Go port was verified against it, and several rules
hold only because a test pinned them. Without it `make check` covers parsing and doc
creation but nothing a user actually runs, and the kit has already taken changes
under that weaker gate.

## Non-goals
- Reproducing the old file line for line. The behaviour is the spec, not the text.
- Replacing the Go unit tests. They cover the parsing primitives well and stay.
- A coverage target. Coverage is not the goal; exercising the commands is.

## Design
Rewrite as a Go test in `cmd/kit`, not as an external script. The old suite was
Python driving a binary, which made sense while the kit *was* Python; now it means a
second language and a build step to test one. A Go test can build the binary once
with `go test`'s own tooling, run it against temporary projects, and assert on
stdout, exit codes, and the files produced — the same black-box contract, in-tree.

Each command gets both directions: a case proving a check fires, and a case proving
it stays quiet when it should. The quiet cases are the ones that matter, because a
noisy check trains people to ignore the tool.

## Tasks
- [x] Harness in `cmd/kit`: build the binary once, helpers to make a temp project and run subcommands
- [x] `init`: fresh layout, never overwrites, appends vs refreshes the protocol, creates the repo, `--stack go`, no duplicate entry point
- [x] `new`: next free ID, slug, frontmatter filled, refuses to overwrite
- [ ] `next`: topmost box, In progress before Accepted, branch claims, INBOX blocks, drafts, WIP warning
- [ ] `audit`: every finding code fires, and each stays quiet on a clean project
- [ ] Robustness: BOM/CRLF, missing frontmatter, unparseable dates, non-git dirs, `--json`, `--strict` exit codes
- [ ] Put `conformance` back in `make check` and delete the missing-file guard

## Open questions
<!-- none -->

## Log
- 2026-08-11 — moving the suite in-tree as a Go test: the external Python script was
  a holdover from when the kit itself was Python, and it could be deleted without
  anything noticing because it was untracked and outside the build.
