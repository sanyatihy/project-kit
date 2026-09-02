---
id: 0055
title: The binary never describes a feature it does not have
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs: 0050, 0051, 0052
---

## Problem

The last round removed a command, a status and five checks, and corrected every document
that named them. Eleven false statements survived it, including a check advertised in
`kit ship --help` that no longer exists. Every one was found by reading; nothing in the
build compares what the binary says about itself with what it does.

## Non-goals

- Golden files for all output. A whole-page snapshot fails on every wording change and
  teaches people to re-record it. What is pinned is the identifiers, not the prose.
- Rewriting the prose again. The corrections here are the ones the tests then hold.

## Design

Every identifier the help text names is checked against the thing that defines it. Three
classes, three tests, all cheap because the registries already exist: a check code named
anywhere in `help.go` must be in `Checks`; a command named in `commandHelp` must be a case
in the dispatcher and the reverse; a topic named in the overview must resolve in `topics`
and the reverse. `kit ship --help` advertising `doc-bloat` is what this catches.

A refusal list is complete or it is not a list. `kit ship` refuses for six reasons and
the protocol names four; `release`'s exit-2 list omits two that 0047 added; `claim --help`
says every refusal is one the board already makes, and three are not. Each list is
generated from, or tested against, the refusals in the code.

The machine-readable board is a contract, so its shape is pinned. `moved` carries
`0001-01-01T00:00:00Z` on every unclaimed feature because `omitempty` does nothing for a
struct; empty groups marshal as `null` while `findings` marshals as `[]`; `--json` on an
unset-up project is untested in both commands. A schema test covers the shapes, and the
zero time stops being emitted.

Help goes to stdout. `kit next -h` writes to stderr and exits 0, so redirecting it
produces an empty file, while `kit new --help` writes to stdout.

A comment that describes deleted code is deleted with it. The cut left a docstring for
`Bloat` sitting on `SetStatus`, one for `gitRun` sitting on `gitFed`, one for `Audit` sitting
on `docState`, three dangling fragments at the end of `parse.go`, and two rule-number
citations that are now wrong.

Also corrected here, each verified against the code: the `Accepted` gate still described by
`kit new` and `kit init`; the online git budget stated as 10 when it is 11; seven secret
shapes when there are eleven; the ceiling tiering in the wrong order; the wrong test named
as the one that enforces the registry.

## Tasks

- [x] Check codes, command names and topic names in help are tested against their registries
- [x] The refusal lists for `ship`, `release` and `claim` match the code, with a test
- [x] `--json` shapes are pinned, `moved` stops emitting a zero time, empty groups are `[]`
- [x] `-h` writes to stdout for every command
- [x] The eleven false statements are corrected and the orphaned comments deleted

## Log

- 2026-08-21 — the doc's eleventh false statement is not one:
  `TestEveryRegisteredCheckIsExercised` exists and does what AGENTS.md, README.md and
  TECH.md say. Nothing was corrected for it.
- 2026-08-21 — three counts were wrong. `ship` refuses for eight reasons, not six;
  `release`'s list omitted four, not two, one of them added by 0050 after this was drafted;
  and none of `claim`'s seven exit 2 refusals is one the board makes, not three.
