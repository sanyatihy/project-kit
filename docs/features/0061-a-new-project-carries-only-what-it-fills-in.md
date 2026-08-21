---
id: 0061
title: A new project carries only what it fills in
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

`kit init` writes 269 lines across five files before the operator writes a word:
`AGENTS.md` at 121, `docs/features/TEMPLATE.md` at 69, `docs/VISION.md` at 46,
`docs/INBOX.md` at 21, `docs/features/BACKLOG.md` at 12. Four of the five are files the
project is then nagged to maintain.

`TEMPLATE.md` is read by nothing. Deleted from a project, `kit new` still writes a full
doc from the embedded template and `audit` never mentions it; `scaffold.go`'s own comment
says the conventions doc is read by nobody. And the `VISION.md` stub is the only finding a
virgin project has: `unfilled-vision` fires on it at once, where a project with no such
file is silent until features exist. Scaffolding a file is what makes the report worse.

The protocol is 113 lines and 27 rules, read every session in every project, and three of
them are false where they are read. Rule 10 says a branch only you can see is not a claim,
while `kit help` says local branches count and an unpushed one does halt the board. Rule 22
names `make check` in projects `kit init` gives no Makefile. Rule 27 names `make release`,
which in a scaffolded Go project builds a binary rather than cutting a tag.

Rule 2 carries a test that does not work. "A sentence that needs an 'and' is two features"
is a grammar check standing in for a judgement, with no cost on the other side, so it only
splits: it produced eleven docs for this round where five were right, three of them split
by kind of code change, which the next clause of the same rule forbids.

`templates/go-checklist.md` is 193 lines, the largest text the binary ships and larger than
the protocol every agent must obey. Nothing in the loop points at it, and a conformance
test exists to stop its checkboxes colliding with the host's.

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
hard case: it is the one file the operator must write, and the stub is what makes a virgin
project report a finding at all, so the stub goes and `no-protocol` carries the prompt once
features exist.

**A rule earns its place by deciding something no refusal can.** The protocol keeps the
rules that are judgement — how to decompose, what `needs:` means, that the doc is the one
home for the design, that discovered work never becomes a `TODO` — and drops the ones that
restate a refusal the tool already makes at the moment it matters. Rule 10 goes because it
is false. Rules 22 and 27 go because they name commands the default project does not have.
Rules 11 and 12 collapse into the refusal 0062 adds.

**Rule 2 keeps its definition and loses its test.** "One shipped increase in what the
project can do, or can be trusted to do" is the useful half. The "and" test is replaced by
the one this round needed and did not have: **a separate doc earns its place when it holds
a decision someone must read before building, that is not recoverable from the code or the
commit.** That test has a cost side, which is what VISION asks of every document, and it
would have produced this round's five docs instead of eleven.

**There is one kind of work.** The protocol sends a fix, a chore or a data update to "a
branch whose name contains no four-digit id, and needs no doc", so the only atomic
operation the kit has covers one kind of work. 0057 proposed a second command for the
other kind and was dropped: chores are 7 of the 76 branches this project has merged, four
of them one-line edits to `BACKLOG.md`, and a second entity costs a twin of every walk
over features — two were already missing on the day it was built. The sentence goes
instead. Work worth coordinating is a feature and gets an id; work not worth three lines
of doc is not worth an id either, and rides on a pull request.

**A doc is as long as its decisions, and no longer.** The template carries the rule:
`## Design` holds what was chosen, what it beat and what it costs, and a doc a reader
cannot hold in one sitting is describing a feature that is too big. No check enforces it —
`doc-bloat` was tried and deleted for reporting a state nobody could act on.

**The kit ships a loop, not advice.** `templates/go-checklist.md` and its help topic go.
VISION says the kit is not a judge of whether work is any good, `audit` cannot check a word
of it, and the checklist is single-language in a kit whose `--stack` flag already reports
which stacks exist. A project that wants it puts it in its own house rules, above the
generated marker, where it can be edited.

The ceilings in `budget_test.go` fall with the files rather than being left where they are.

## Tasks

- [x] `init` stops writing `TEMPLATE.md` and `BACKLOG.md`, and `kit block` creates `INBOX.md`
- [x] `init` stops writing the `VISION.md` stub, and `no-protocol` carries the prompt
- [x] The protocol drops the rules a refusal already makes, and rule 10 goes as false
- [x] Rule 2's "and" test is replaced by the doc-earns-its-place test
- [x] The protocol stops routing work away from the kit, and says how long a doc runs
- [ ] `go-checklist` leaves the binary, and the ceilings fall with the files

## Open questions

- **Which of the remaining rules an operator wants is partly theirs.** The audit judged ten
  cuttable; this doc names the ones that are false or unreachable and leaves the rest to
  the review. A rule that is only redundant, not wrong, is cheap to keep.
- **Does `kit init` still write `docs/VISION.md` as an empty heading?** Dropping the file
  entirely means a new project has no prompt at all until `audit` runs. The leaning is to
  drop it and let `no-protocol` say so.

## Log

- 2026-08-21 — the replacement position is folded into rule 2 rather than left as a
  paragraph of its own: a chore commit rides on a pull request under a branch name carrying
  no id. The rule that decides what is a feature is where the alternative belongs, and the
  id constraint is the load-bearing half of the sentence that went.

- 2026-08-21 — seven rules go, not the five this doc named: 8, 10, 13, 23, 24, 26, 27.
  Each of 8, 13, 23, 24 and 26 is printed by a command at the moment it matters —
  `kit claim` says "Land the round, then claim", `next` closes every run with an
  instruction including "say so and wait", each `ship` refusal names its own fix, `ship`
  prints "move durable facts into docs/TECH.md, one line each", and `next` prints
  "your own is never yours to merge" itself. 24 was also said twice inside the protocol,
  by **Every fact has one home**.
- 2026-08-21 — rule 22 is kept and corrected rather than cut. Dropping it would leave
  nothing saying to run `kit audit --strict` before `kit ship`, which no refusal makes;
  what was wrong with it was naming `make check`, so it now names the project's own gate.
  A test holds the protocol against naming `make check`, `make install` or `make release`.
- 2026-08-21 — rules 11 and 12 stay. The refusal they collapse into is 0062's and has not
  shipped, and 0062's own `## Design` says it turns them into a refusal and one line of
  prose — so it is that feature's cut. Removing them first would leave the protocol silent
  about the failure `split-claim` is HIGH for.
- 2026-08-21 — rule 27's fact is true of this repo and not of a project the kit scaffolds,
  so it moved above the marker in this repo's own `AGENTS.md` rather than being deleted.

- 2026-08-21 — the open question is settled the way it leaned: no `VISION.md` at all, not
  an empty heading. `no-protocol` gained a second fix line for it, because `kit init .`
  no longer writes the file the finding asks for and offering that command would be false.
- 2026-08-21 — `unfilled-vision` stays. Nothing scaffolds a stub now, but
  `kit help vision > docs/VISION.md` still produces one and leaving it unanswered is the
  same defect it always reported.

- 2026-08-21 — `init` still writes one thing into `docs/features/`: a `.gitkeep`. The board
  is a directory listing, so `next`, `audit` and `new` all exit 2 without the directory,
  and git drops an empty one — a clone of a project that has planned nothing would arrive
  unable to plan. Same rule the stack already uses for `internal/`.
- 2026-08-21 — `no-protocol` stops reporting a missing `docs/INBOX.md`. The check named
  the documents `kit init` writes; the inbox is no longer one of them, and reporting it
  would nag every project that has never been stuck.
