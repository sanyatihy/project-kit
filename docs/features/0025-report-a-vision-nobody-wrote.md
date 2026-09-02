---
id: 0025
title: Report a vision nobody wrote
status: Shipped
created: 2026-08-16
shipped: 2026-08-16
needs:
related: 0016-make-an-unfinished-doc-fail-the-audit.md
---

## Problem
`kit init` creates `docs/VISION.md` from the template, and `no-vision` only fires when
the file is *missing*. So the run that adds an empty stub makes a project score better
while telling it nothing — the check is silenced by the act of not answering it.

Live in two projects adopted this week: both carry a `docs/VISION.md` that is entirely
template prose and HTML comments, and both audit clean.

This is `unfilled-template`'s defect one file over. Rule 1 sends every agent to
`docs/VISION.md` before drafting and says a feature that cannot be traced to it is the
wrong feature — against a stub, anything traces to nothing.

## Non-goals
- Judging whether a vision is any good. Only whether it was written. `audit`
  reports what is checkable and never guesses.
- Refusing to create the stub. The headings are the prompt; a project with no
  file at all is worse off.
- Blocking `new` or `claim` on it. A thin vision is a planning problem, not a
  reason to stop work already planned.

## Design
`unfilled-vision`, MEDIUM, fires when `docs/VISION.md` exists and every section under
its headings is empty once comments are stripped.

The same test `unfilled-template` uses, and safe for the same reason: the template's
guidance lives in HTML comments, which every parser here strips, so an empty result
means nothing was written rather than something being missed. A partly-filled vision
does not fire — one written section is someone having started, and a check that
nags at work in progress is one people learn to ignore.

MEDIUM, not HIGH: nothing is untrue and no agent is misled about state — the cost is
drafting without a target. `no-vision` stays LOW and unchanged for the missing-file
case; this is the same gap, found one step later.

## Tasks
- [x] `unfilled-vision` at MEDIUM, on `VISION.md` with every section empty after
      comments are stripped
- [x] Conformance: fires on a fresh `kit init` stub, stays quiet once one section is
      written, and stays quiet when the file is absent (that is `no-vision`)
- [x] README's check list, and the `vision` help topic saying the stub is reported

## Log
- 2026-08-16 — `kit init` cleared `no-vision` in both projects the kit was adopted into,
  without either gaining a sentence.
