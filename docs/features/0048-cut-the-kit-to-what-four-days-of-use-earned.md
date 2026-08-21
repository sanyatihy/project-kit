---
id: 0048
title: Cut the kit to what four days of use earned
status: Draft
created: 2026-08-21
shipped:
check:
checked:
review:
needs:
related:
---

## Problem

Across 82 feature docs in five projects, a measurable part of the kit was never reached
once: a status, two commands, four frontmatter fields, two help topics and the multi-path
board. Every one of them still costs an agent context on every read, and two of the five
ways the kit can lose work live inside machinery nobody invoked.

## Non-goals

- **Cutting what the evidence used.** `go-checklist` has a commit behind it and stays;
  `--fyi`, `needs:` and `--strict` are thin but real and stay.
- **Deprecation.** No project depends on a released tag, so removal is removal. Docs
  carrying a removed field keep parsing — unknown frontmatter keys are ignored.
- **Rewriting the shipped corpus.** 46 docs carry `related:` and `Accepted`; those are
  frozen history and stay as written.

## Design

**The rule: a surface earns its place by being used, and the measurement is the corpus.**
Never reached in four days across five projects means removed, not deferred.

| removed | measured |
| --- | --- |
| `Accepted`, `kit accept`, `--review`, `review:` | 0 of 82 docs; every ship went `Draft` to `Shipped` |
| `kit check`, `check:`, `checked:`, `unchecked` | 0 of 82; set once, on a doc dropped the same day |
| `Superseded` | 0; no command can write it |
| `related:` | filled 77 times, read by no line of Go |
| multi-path `next`/`audit` | 0 invocations |
| `kit help conventions`, `features-README.md` | teaches the hand-copy `kit new` replaced |
| `kit help deployment` | 0 references in any project |

**Dropping `Accepted` collapses the board's ordering rule.** Rank becomes the claim this
checkout holds, then lowest id. `unfilled-template` and `no-tasks` were gated on
"not Draft" and would go dead, so they regate on **tracked in `HEAD`** — an unwritten doc
that has been committed is the real defect, and a doc fresh out of `kit new` is untracked
and stays quiet. That is a better test than the one it replaces.

**Dropping multi-path deletes the fan-out, not just the flag.** `inParallel`, the
per-project payload keyed by directory basename, and the error field that existed because
a bare list had nowhere to report a bad path all go with it. The keying bug — two projects
sharing a basename, one silently dropped — is deleted rather than fixed. A shell loop is
the replacement, and `kit audit` gains the empty-target guard in `linksIn` that a stray
markdown link turned into a panic.

**A feature is one decision; size is the sanity check.** "One pull request" is circular
and is what let this audit first decompose into eight docs split by layer. The protocol
leads with the decision test instead.

**Nothing may describe a surface that is gone.** The protocol, `kit help`, the README, the
templates and `docs/TECH.md` are cut in the same change, because a kit whose prose names a
deleted command is worse than one that never cut.

## Tasks

- [ ] `accept`, `check` and their statuses, fields, flags and checks are gone from the binary
- [ ] `Superseded` and `related:` are gone; `unfilled-template` and `no-tasks` gate on tracked
- [ ] `next` and `audit` take no paths; the fan-out and the keyed payload are deleted
- [ ] `conventions` and `deployment` topics and their templates are gone
- [ ] `repo.go` splits by subject and the dead identifiers named in the audit are deleted
- [ ] Protocol, help, README, TECH and templates describe only what remains
- [ ] The protocol's feature rule leads with "records a decision"

## Log
