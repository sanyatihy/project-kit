---
id: 0061
title: A new project carries only what it fills in
status: Draft
created: 2026-08-21
shipped:
needs: 0057
---

## Problem

`kit init` writes 269 lines across five files before the operator writes a word:
`AGENTS.md` at 121, `docs/features/TEMPLATE.md` at 69, `docs/VISION.md` at 46,
`docs/INBOX.md` at 21, `docs/features/BACKLOG.md` at 12. Four of the five are files the
project is then nagged to maintain.

`TEMPLATE.md` is read by nothing. Deleted from a project, `kit new` still writes a full
doc from the embedded template and `audit` never mentions it; `scaffold.go`'s own comment
says the conventions doc is read by nobody. Writing `docs/VISION.md` makes the audit
worse: no file is `no-vision` at LOW, and the stub `init` writes is `unfilled-vision` at
MEDIUM — the only finding a virgin project has.

The protocol is 113 lines and 27 numbered rules, shipped into every project and read every
session. Three are worse than redundant. Rule 10 says a branch only you can see is not a
claim, while `kit help` says local branches, remote branches and worktrees all count — and
an unpushed local branch does halt the board. Rule 22 names `make check` in projects
`kit init` gives no Makefile, which is the default path. Rule 27 names `make release`,
which in a scaffolded Go project builds a binary rather than cutting a tag, and
`budget_test.go` records that the generated section cannot name the command.

And rule 2 carries a test that does not work. "A sentence that needs an 'and' is two
features" is a grammar check standing in for a judgement, with no cost on the other side,
so it only ever splits. Applied to the audit this round decomposes, it produced eleven
feature docs where five were right — and it split three of them by kind of code change,
which the next clause of the same rule forbids.

`templates/go-checklist.md` is 193 lines, the largest text the binary ships and larger
than the protocol every agent must obey. Nothing in the loop points at it. Eight of its
opening lines exist to stop the host tool misreading its checkboxes, and a conformance
test pins that it contains none.

## Non-goals

- **Making `kit init` write less than a project needs.** `AGENTS.md` stays, because it is
  the only file agents read, and the Go stack's load-bearing files stay.
- **Rewriting the protocol's substance.** What survives keeps its wording; this decides
  what survives.
- **A style guide somewhere else in the kit.** The checklist leaves; it does not move.

## Design

**A file the kit writes is one the project will fill in, or the kit does not write it.**
`TEMPLATE.md` and `BACKLOG.md` stop being written. `docs/INBOX.md` is created by
`kit block` on first use, the way `kit ship` already creates `docs/TECH.md` — the model
already exists in the codebase and is the one that does not nag. `docs/VISION.md` is the
hard case: it is the one file the operator must write, and shipping a stub is what turns a
LOW finding into a MEDIUM one, so the stub goes and `no-vision` carries the prompt.

**A rule earns its place by deciding something no refusal can.** The protocol keeps the
rules that are judgement — how to decompose, what `needs:` means, that the doc is the one
home for the design, that discovered work never becomes a `TODO` — and drops the ones that
restate a refusal the tool already makes at the moment it matters. Rule 10 goes because it
is false. Rules 22 and 27 go because they name commands the default project does not have.
Rules 11 and 12 collapse into the refusal 0057 adds.

**Rule 2 keeps its definition and loses its test.** "One shipped increase in what the
project can do, or can be trusted to do" is the useful half. The "and" test is replaced by
the one this round needed and did not have: **a separate doc earns its place when it holds
a decision someone must read before building, that is not recoverable from the code or the
commit.** That test has a cost side, which is what VISION asks of every document, and it
would have produced this round's five docs instead of eleven.

**The kit ships a loop, not advice.** `templates/go-checklist.md` and its help topic go.
VISION says the kit is not a judge of whether work is any good, `audit` cannot check a word
of it, and the checklist is single-language in a kit whose `--stack` flag already reports
which stacks exist. A project that wants it puts it in its own house rules, above the
generated marker, where it can be edited.

The ceilings in `budget_test.go` fall with the files rather than being left where they are.

## Tasks

- [ ] `init` stops writing `TEMPLATE.md` and `BACKLOG.md`, and `kit block` creates `INBOX.md`
- [ ] `init` stops writing the `VISION.md` stub, and `no-vision` carries the prompt
- [ ] The protocol drops the rules a refusal already makes, and rule 10 goes as false
- [ ] Rule 2's "and" test is replaced by the doc-earns-its-place test
- [ ] `go-checklist` leaves the binary, and the ceilings fall with the files

## Open questions

- **Which of the remaining rules an operator wants is partly theirs.** The audit judged ten
  cuttable; this doc names the ones that are false or unreachable and leaves the rest to
  the review. A rule that is only redundant, not wrong, is cheap to keep.
- **Does `kit init` still write `docs/VISION.md` as an empty heading?** Dropping the file
  entirely means a new project has no prompt at all until `audit` runs. The leaning is to
  drop it and let `no-vision` say so.

## Log
