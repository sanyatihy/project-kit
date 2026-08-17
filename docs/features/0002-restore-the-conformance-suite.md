---
id: 0002
title: Restore the conformance suite
status: Shipped
created: 2026-08-11
shipped: 2026-08-11
check:
checked:
related: ../../AGENTS.md
---

## Problem
`./test` — the suite that built the binary and drove `init`, `next` and `audit` from the
outside — was deleted while untracked and is unrecoverable. It was the behavioural
specification, and the kit has already taken changes under the weaker gate that remains.

## Non-goals
- Reproducing the old file line for line: the behaviour is the spec, not the text.
- A coverage target. Exercising the commands is the goal.

## Design
Rewrite as a Go test in `cmd/kit` rather than an external script. The old suite was Python
driving a binary, which is how it could be deleted without anything noticing — untracked
and outside the build. A Go test builds the binary once and asserts on stdout, exit codes
and the files produced: the same black-box contract, in-tree.

Every check gets both directions, a case that it fires and a case that it stays quiet. The
quiet ones matter more: a noisy check trains people to ignore the tool.

## Tasks
- [x] Harness in `cmd/kit`: build the binary once, helpers to make a temp project and run subcommands
- [x] `init`: fresh layout, never overwrites, appends vs refreshes the protocol, creates the repo, `--stack go`, no duplicate entry point
- [x] `new`: next free ID, slug, frontmatter filled, refuses to overwrite
- [x] `next`: topmost box, In progress before Accepted, branch claims, INBOX blocks, drafts, WIP warning
- [x] `audit`: every finding code fires, and each stays quiet on a clean project
- [x] Robustness: BOM/CRLF, missing frontmatter, unparseable dates, non-git dirs, `--json`, `--strict` exit codes
- [x] Put `conformance` back in `make check` and delete the missing-file guard

## Log
- 2026-08-11 — every case was verified by planting the regression it guards. Two probes
  passed while testing nothing: a fenced example under `## Design`, which `OpenTasks` never
  reads, and a comment test asserting a status. Planting is why they were found.
- 2026-08-11 — `TestEveryRegisteredCheckIsExercised` turns "add a test that it fires" from
  an honour-system rule into a build failure.
