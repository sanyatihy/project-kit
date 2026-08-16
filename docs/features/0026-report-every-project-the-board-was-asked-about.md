---
id: 0026
title: Report every project the board was asked about
status: Shipped
created: 2026-08-16
shipped: 2026-08-16
check:
checked:
needs:
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem
One unusable path suppresses the whole fleet board. `kit next ~/code/*` is the
documented way to survey several projects, and a glob that picks up a deleted
directory, a stray file, or a moved repo reports nothing about any of them:

```
$ kit next ./td ./does-not-exist
kit next: not a directory: …/does-not-exist          (exit 2)
```

`./td` is a healthy project, listed first, and the board says nothing about it. A
directory that merely is not a project is handled correctly — only a missing or
non-directory path is fatal.

The failure mode is the one the codebase already rejected elsewhere. `remoteTimeout`
exists because *"a board that hangs because one remote is unreachable is worse than
a board that falls back to a stale mirror and says so"*. A path is the same argument:
the surveyed projects are independent, and one of them being gone says nothing about
the others.

It fails quietly in the direction that matters. An operator who globs a directory
that has since been tidied gets an error about the path they did not care about and
silence about the four projects they did.

## Non-goals
- **Guessing what a bad path meant.** No fuzzy matching, no skipping silently. The
  bad path is still reported, and still sets the exit code.
- **Changing single-path behaviour.** `kit next /gone` with one argument is a broken
  invocation and exit 2 is right. This is only about not letting it take the others
  down.
- **`init`, `new`, `claim`, `ship`, `drop`.** Those act on one project and should
  keep failing outright on a bad path.

## Design
`next` and `audit` survey every path they were given, then report. A path that cannot
be read becomes a per-project failure line in the output rather than an early return,
so the boards that could be produced are.

Exit code keeps its meaning: 2, because something the caller asked for could not be
done. `audit --strict` still exits 1 on a HIGH finding, and a bad path outranks it —
2 is the more serious answer and the one a script must not mistake for findings.

`--json` carries the failures too, keyed by path, so an agent parsing the fleet is not
told about four projects while a fifth vanished. Both formats come from one decision,
as with `guidance` — the two must not be able to disagree about what was surveyed.

Order is preserved: projects report in the order given, failures included, so a fleet
survey reads the same way twice.

## Tasks
- [x] `next` surveys every path, reporting unreadable ones per-project instead of
      returning early; conformance with a good path before and after a bad one
- [x] `audit` does the same, and `--strict` still exits 1 on HIGH while a bad path
      exits 2; conformance for both
- [x] `--json` carries per-path failures for both commands; conformance asserting the
      payload still contains the healthy projects
- [x] README's fleet example says what happens when one path is unusable

## Log
- 2026-08-16 — found probing the fleet view while two agents built 0023 and 0025.
- 2026-08-16 — the two `--json` payloads carry a failure differently, because their
  value types differ. `next`'s is one object per project, so the failure is an
  `error` field on the plan and the shape is unchanged. `audit`'s was a bare list of
  findings, which has nowhere to put one; a synthetic finding would have meant a code
  outside the registry `kit help checks` prints from, which is the drift the registry
  exists to prevent. So audit's entry is now `{"findings": [...], "error": "..."}` —
  a shape change for anything parsing `audit --json`.
- 2026-08-16 — a failed path is written twice on purpose: as a line in the board on
  stdout, so a fleet survey reads in order and both formats agree on what was
  surveyed, and as the same one-line diagnostic on stderr, which is where a
  single-path run has always reported it and what keeps `--json` pipe-clean.
