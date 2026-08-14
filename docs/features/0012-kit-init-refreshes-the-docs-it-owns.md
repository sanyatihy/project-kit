---
id: 0012
title: kit init refreshes the docs it owns
status: Accepted
created: 2026-08-14
needs:
related: 0011-features-declare-what-they-need.md
---

## Problem
0011 added `needs:` to the feature template. Every project scaffolded before it still
has the old one on disk, including this repo — `docs/features/TEMPLATE.md` here does
not mention the field that `kit new` now writes into every doc.

Nothing breaks, because `kit new` writes from the embedded template rather than the
project's copy. That is exactly what makes it worth fixing: the file has no effect, so
it is pure documentation, and a piece of pure documentation that disagrees with the
tool is the thing `audit` exists to prevent everywhere else. A human reading
`docs/features/TEMPLATE.md` to learn the format currently learns a format that is one
release out of date, and nothing anywhere says so.

`kit init` already solves this for one file. The `## Work protocol` section of
`AGENTS.md` is regenerated on every run, which is how projects pick up protocol
changes without losing their own rules. The reference docs never got the same
treatment, so protocol changes travel and format changes do not.

## Non-goals
- **Schema versioning.** Every format change so far has been additive: a missing
  `needs:` reads as empty, `no-vision` waits for features to exist, and no project has
  broken. A version field is durable state to maintain forever in a tool whose whole
  claim is having none. The trigger to revisit is the first change that *cannot* be
  additive — a renamed field or changed meaning, where an old project misbehaves
  rather than merely missing something.
- **Migrating a project's own documents.** `VISION.md`, `TECH.md`, `INBOX.md`,
  `BACKLOG.md` and every stack file are what the project filled in. Their copy is the
  truth and nothing here may touch them, now or later.
- **A check that reports the drift.** It would fire on every project after every
  release, for a file with no effect, and the fix would always be the same command.
  Noise for something a re-run already corrects.
- **Merging edits.** No three-way merge, no markers inside the reference docs. They
  are replaced wholesale or not at all.

## Design
**A file is the kit's to refresh when nothing reads the project's copy of it.** That
is the whole rule, and it sorts the scaffold cleanly. `docs/features/TEMPLATE.md` is
never read — `kit new` uses the embedded template. `docs/features/README.md` is never
read by anything at all. Edits to either have no effect today, so replacing them takes
nothing away and keeps them true.

Everything else exists to be filled in, and its copy is the only copy. The test for
adding a file to the refreshed set is not "did the kit write it first" but "would the
project notice if its version were replaced".

**House rules already have a home**, which is what makes wholesale replacement safe.
Anything project-specific goes above the marker in `AGENTS.md`, where `kit init`
already preserves it. Nothing is lost by treating the reference docs as the kit's.

**Refreshed is reported, not silent.** `init` already prints `refreshed` for the
protocol section and `kept` for everything else, so the existing output says which
happened without inventing a vocabulary.

## Tasks
- [ ] `kit init` refreshes the reference docs it owns, and reports it the way it already reports the protocol
- [ ] A project's own documents are never touched, pinned by a test that fails if that line moves

## Open questions
<!-- none -->

## Log
- 2026-08-14 — considered schema versioning and deferred it deliberately. The problem
  in front of us is a reference doc going stale, which is fixed by regenerating it;
  versioning solves migration, which nothing has needed yet because every change has
  been additive. Recorded so the next person meets the reasoning rather than the gap.
