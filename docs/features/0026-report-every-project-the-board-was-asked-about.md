---
id: 0026
title: Report every project the board was asked about
status: Shipped
created: 2026-08-16
shipped: 2026-08-16
needs:
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem

One unusable path suppresses the whole fleet board. `kit next ~/code/*` is the documented
way to survey several projects, and a glob that picks up a deleted directory, a stray file
or a moved repo reports nothing about any of them:

```
$ kit next ./td ./does-not-exist
kit next: not a directory: …/does-not-exist          (exit 2)
```

`./td` is a healthy project, listed first, and the board says nothing about it. A directory
that merely is not a project is handled correctly — only a missing or non-directory path is
fatal. The surveyed projects are independent, so one being gone says nothing about the
others.

## Non-goals

- Guessing what a bad path meant. No fuzzy matching and no silent skipping: the bad path is
  still reported and still sets the exit code.
- Changing single-path behaviour. `kit next /gone` with one argument is a broken invocation
  and exit 2 is right.
- `init`, `new`, `claim`, `ship`, `drop`. Those act on one project and should keep failing
  outright on a bad path.

## Design

`next` and `audit` survey every path they were given, then report. A path that cannot be
read becomes a per-project failure line rather than an early return, so the boards that
could be produced are.

Exit code keeps its meaning: 2, because something the caller asked for could not be done.
`audit --strict` still exits 1 on a HIGH finding, and a bad path outranks it — 2 is the more
serious answer and the one a script must not mistake for findings.

`--json` carries the failures too, keyed by path, so an agent parsing the fleet is not told
about four projects while a fifth vanished. Both formats come from one decision, as with
`guidance`.

Order is preserved, failures included, so a fleet survey reads the same way twice.

## Tasks

- [x] `next` surveys every path, reporting unreadable ones per-project instead of returning
      early
- [x] `audit` does the same, and `--strict` still exits 1 on HIGH while a bad path exits 2
- [x] `--json` carries per-path failures for both commands
- [x] README's fleet example says what happens when one path is unusable

## Log

- 2026-08-16 — `audit --json` changed shape to `{"findings": [...], "error": "..."}`. A bare
  list has nowhere to say why it is empty, and a synthetic finding would mean a code outside
  the registry `kit help checks` prints from. Breaking for anything parsing it.
