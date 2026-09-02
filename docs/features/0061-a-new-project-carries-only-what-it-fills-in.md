---
id: 0061
title: Everything a new project needs comes from one command
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs:
---

## Problem

A project's files arrive from three commands and its protocol is partly untrue. `kit init`
writes some; `kit ship` creates `docs/TECH.md` on first use; `docs/features/BACKLOG.md` is
named by a protocol rule and by `ship` while nothing creates it. VISION says one command
sets a project up.

Plain `kit init` writes no `.gitignore`, so the first `git add -A` commits what the editor
and the OS left behind — and `kit claim` pushes `HEAD`, so it reaches the next agent's
branch.

Three of the protocol's 27 rules are wrong where they are read. Rule 10 says an unpushed
branch is not a claim, while `kit help` says local branches count and one does halt the
board. Rule 22 names `make check` in projects `init` gives no Makefile. Rule 27 names
`make release`, which in a scaffolded Go project builds a binary rather than cutting a tag.
Rule 2's test — a sentence needing an "and" is two features — is a grammar check with no
cost on the other side, and produced eleven docs for a round where five were right.

## Non-goals

- Cutting guidance an agent needs. `kit help go-checklist` stays; nothing points at it,
  which is a reason to point at it.
- Writing over a file a project already has. That guarantee outweighs any line it might add.
- Guessing the language. What plain `kit init` ignores is what is true of every project.

## Design

`kit init` writes the whole layout and nothing else creates a project file, so there is one
answer to where a file came from. `ship`'s lazy `docs/TECH.md` stays only as a fallback for
projects that predate this.

`unfilled-vision` waits for `len(Features) > 0`, the gate `no-protocol` already uses. It
fired on the stub the moment a project was created, which is what made scaffolding VISION
look expensive — one line, against moving four files between three commands.

A `.gitignore` is a refusal expressed as a file. Plain `init` covers editor and OS
droppings only; the alternative is a prose rule, the weakest tier there is. A stack's copy
replaces the default rather than extending it: one file to read, paid for by repeating the
editor and OS lines in every stack.

A rule earns its place by deciding something no refusal can. The protocol keeps the
judgement — how to decompose, what `needs:` means, the doc as the one home for the design —
and drops what restates a refusal made where it matters. Rule 10 goes as false. Rule 22 is
corrected rather than cut, since nothing else says to run `kit audit --strict` before
shipping. Rule 27 goes as a fact about a Go module rather than about every project.

Rule 2 keeps its definition and loses its test. The replacement: a separate doc earns its
place when it holds a decision someone must read before building, not recoverable from the
code or the commit. That has a cost side, which the "and" test did not.

There is one kind of work. Rule 5's routing sentence goes — work worth coordinating is a
feature with an id, and work not worth three lines of doc rides on a pull request. 0057
proposed a `kit chore` command and was dropped; chores are 7 of 76 merged branches.

`docs/features/TEMPLATE.md` stays gone. `kit new` writes from the embedded template and
`audit` never mentions the placed copy.

## Tasks

- [x] The protocol commits are kept, and the three that cut `init` and the checklist reversed
- [x] `init` writes `VISION.md`, `INBOX.md`, `TECH.md` and `BACKLOG.md`, overwriting nothing
- [x] `init` writes a `.gitignore`, and the Go stack's copy drops what is not Go
- [x] `unfilled-vision` waits for features, so a virgin project audits clean
- [x] `kit help backlog` describes the file the protocol names, which points at the checklist
- [x] A test holds every file the protocol names against something that writes or describes it

## Log

- 2026-08-21 — pivoted. The doc cut both what a project receives and what its protocol says;
  the cut of `init` is reversed and the checklist kept. The filename predates the retitle,
  since `Branch()` reads it and renaming would strand the claim.
- 2026-08-21 — a stack's `.gitignore` replaces the default rather than extending it, while
  there is one stack.
