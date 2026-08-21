---
id: 0061
title: Everything a new project needs comes from one command
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

A project's files arrive from several places, and its protocol is partly untrue.

`kit init` writes some files; `kit ship` creates `docs/TECH.md` on first use; and
`docs/features/BACKLOG.md` is named by protocol rule 14 and by `kit ship` while nothing
creates it and no topic describes it. To know where a file comes from you have to know
which command makes it. VISION says a new project is set up with one command; it is set up
by one command plus whichever others happen to run later.

There is no `.gitignore` unless `--stack` is given, so the first `git add -A` an agent runs
commits whatever the editor and the OS left behind — and `kit claim` pushes `HEAD`, so it
reaches the branch the next agent works in. The stack copy spends roughly fifteen lines on
Terraform state, `vault.yml` and Ansible inventories in projects with no infrastructure.

The protocol is 113 lines and 27 rules, read at the start of every session, and three are
wrong where they are read. Rule 10 says a branch only you can see is not a claim, while
`kit help` says local branches count and an unpushed one does halt the board. Rule 22 names
`make check` in projects `kit init` gives no Makefile. Rule 27 names `make release`, which
in a scaffolded Go project builds a binary rather than cutting a tag. Rule 2's test — "a
sentence that needs an 'and' is two features" — is a grammar check standing in for a
judgement with no cost on the other side, and it produced eleven docs for one round where
five were right. Rule 5 routes a fix or a data update to "a branch whose name contains no
four-digit id, and needs no doc", so the only atomic operation the kit has covers one kind
of work.

## Non-goals

- **Cutting guidance an agent needs.** `kit help go-checklist` stays. Nothing in the
  protocol points at it, which is a reason to point at it rather than delete it — VISION
  says anything an agent has to know is embedded in `kit help`, and it is.
- **Writing over a file a project already has.** `kit init` overwrites nothing the project
  wrote, and that guarantee is worth more than any line it might add.
- **Guessing the language.** What plain `kit init` ignores is what is true of every
  project, not what this one will be built in.

## Design

**`kit init` writes the whole layout, and nothing else creates a project file.** One
command, one answer to where a file came from. `docs/VISION.md`, `docs/INBOX.md`,
`docs/TECH.md` and `docs/features/BACKLOG.md` are written from the templates they already
have; `ship`'s lazy path stays only as a fallback for projects that predate this, and is no
longer where a file first appears. Splitting creation across commands is reversed here: it
moved four files between three commands to remove one finding.

**That finding is a check gate, not a scaffolding problem.** `unfilled-vision` fires on the
stub the moment a project is created, which is what made scaffolding it look expensive.
`no-protocol` already gates its own VISION arm on `len(Features) > 0` and says why — a
project set up and not yet planned has nothing to decompose. `unfilled-vision` takes the
same gate, and a virgin project audits clean while carrying every file it will need. One
line, against moving four files between three commands.

**A `.gitignore` is a refusal expressed as a file.** Plain `kit init` writes one covering
editor and OS droppings and nothing else. The alternative is a rule in prose telling agents
what not to commit, which is the weakest tier the kit has, and the cost of not having one
lands on whoever reviews a branch carrying somebody's `.DS_Store`. The Go stack's copy
drops the lines that are not about Go, and replaces the default rather than extending it:
one file to read, paid for by repeating the editor and OS lines in every stack the kit
grows.

**A rule earns its place by deciding something no refusal can.** The protocol keeps the
rules that are judgement — how to decompose, what `needs:` means, that the doc is the one
home for the design, that discovered work never becomes a `TODO` — and drops the ones that
restate a refusal made at the moment it matters. Rule 10 goes because it is false. Rule 22
is corrected rather than cut, because nothing else says to run `kit audit --strict` before
shipping: it names the project's own gate. Rule 27 goes as a fact about a Go module rather
than about every project.

**Rule 2 keeps its definition and loses its test.** "One shipped increase in what the
project can do, or can be trusted to do" is the useful half. The replacement is the test
that round needed: **a separate doc earns its place when it holds a decision someone must
read before building, that is not recoverable from the code or the commit.** That test has
a cost side, which is what VISION asks of every document.

**There is one kind of work.** Rule 5's routing sentence goes. Work worth coordinating is a
feature and gets an id; work not worth three lines of doc is not worth an id either, and
rides on a pull request. 0057 proposed a `kit chore` command for the other case and was
dropped — chores are 7 of the 76 branches this project has merged.

**Guidance the protocol does not carry, it points at.** A closing line names
`kit help go-checklist`, which is the only thing it was missing, and `kit help backlog`
describes the file rule 14 names.

**`docs/features/TEMPLATE.md` stays gone.** Deleted from a project, `kit new` still writes a
full doc from the embedded template and `audit` never mentions it; `scaffold.go`'s own
comment says the conventions doc is read by nobody. A dead file, not a deferred one.

Commits `2926425`, `3aace68`, `f5f26b1` and `6045e7f` on this feature's branch are the
protocol half, already built and tested, including a test that fails on a gap in the rule
numbering. They are kept; `b9c8b08`, `654116b` and `d8bdd11` are reversed.

## Tasks

- [x] The protocol commits are kept, and the three that cut `init` and the checklist reversed
- [x] `init` writes `VISION.md`, `INBOX.md`, `TECH.md` and `BACKLOG.md`, overwriting nothing
- [x] `init` writes a `.gitignore`, and the Go stack's copy drops what is not Go
- [ ] `unfilled-vision` waits for features, so a virgin project audits clean
- [ ] `kit help backlog` describes the file rule 14 names, and the protocol points at the checklist
- [ ] A test holds every file the protocol names against something that writes or describes it

## Log

- 2026-08-21 — pivoted. The doc cut both what a project receives and what its protocol
  says; the cut of `init` is reversed and the checklist kept, leaving one feature about
  what a new project gets. The filename predates the retitle: `Branch()` reads it, and
  renaming it would strand the claim this feature already holds.
- 2026-08-21 — the `.gitignore` question is settled on its leaning: a stack's copy replaces
  the default rather than extending it, while there is one stack.
