---
id: 0052
title: Audit is silent only about what it checked
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

`audit` is believed because it says nothing when nothing is wrong. Seven reproduced states
make it say nothing when it could not look: one accented character in a filename disables
every plan check, an unreadable doc is invisible to every command, `--offline` drops a HIGH
check without a word, and this repo's own `AGENTS.md` is structurally broken while `kit
audit` reports that the docs and the repo agree.

## Non-goals

- **New checks for their own sake.** Each addition here closes a state where an existing
  check went quiet, not a new opinion about documents.
- **Guessing at what could not be read.** Unreadable stays unreadable; the change is that
  it is reported rather than skipped.

## Design

**A file the kit cannot read is a finding, not an absence.** `readFeature` returns nil on
any read error, so a doc with no read permission is invisible to `audit`, absent from the
board, and unknown to `claim` — and if it held the highest id, `kit new` mints a duplicate.
Report it at HIGH and keep the id reserved.

**Paths are compared as bytes git and the filesystem agree on.** `TrackedAtHEAD` matches
`os.ReadDir` names against `git ls-tree`, which C-quotes anything non-ASCII, so the lookup
never matches and `unfilled-template`, `no-tasks` and `done-not-shipped` are all skipped.
Reproduced: `0001-blank.md` reports 3 HIGH and `--strict` exits 1; the byte-identical
`0001-blànk.md` reports nothing and exits 0. Read paths with `-z`, and the same for
`DirtyFiles`, which gates `ship`.

**A protocol section that is missing is as bad as a missing file.** `no-protocol` tests
only that `AGENTS.md` exists, and `protocol-stale` compares a section it cannot find, so a
file whose `## Work protocol` heading is gone audits clean and can never be refreshed by
`kit init` again. This repo is in that state now. `no-protocol` covers both.

**A check that did not run says so.** The unreachable-remote path prints a notice;
`--offline` prints nothing and silently drops `shipped-not-pushed`, so `--strict` exits 0
where it would have exited 1. `--offline` names the checks it skipped.

**A project with no plan is not a passing project.** `next`, `audit` and `audit --strict`
all exit 0 in a directory with no `docs/features`, so a CI gate is green for a repo that
has no docs at all. That is "could not run": exit 2.

**A fix line that makes things worse is worse than no fix line.** `protocol-stale` reports
"older than the installed kit" whichever way the difference runs, and offers `kit init .`.
Where the repo is ahead of the binary — every time the protocol changes before `make
install` — that command overwrites the newer protocol with the embedded older one. It did
so here, silently, mid-edit.

**A fix line that cannot work is a bug in the check.** A doc with no `## Tasks` section
reports `done-not-shipped` — "every task is ticked" — whose fix, `kit ship`, refuses it;
a zero-byte doc gets the same finding with `run: kit ship ` and no id. And `tech-drift`
compares against a `shipped:` nothing validates, so one far-future date makes it
unsilenceable while an unparseable one reports "no shipped: date" when there is one.

## Tasks

- [x] `no-protocol` reports an `AGENTS.md` that carries no protocol section
- [x] An unreadable feature doc is a HIGH finding and still reserves its id
- [x] Paths are read `-z`, and a non-ASCII filename no longer disables the plan checks
- [x] `--offline` names what it skipped; a project with no `docs/features` exits 2
- [x] `done-not-shipped` needs a `## Tasks` section; `shipped:` is validated as a real date
- [ ] `protocol-stale` says which side is behind, and never offers a fix that overwrites the newer one

## Log
- 2026-08-21 — the Problem's claim that this repo's own `AGENTS.md` is structurally
  broken does not hold: the `## Work protocol` heading is intact here, so `no-protocol`
  is correctly quiet about it. The check still closes the state for every project that
  is in it.
