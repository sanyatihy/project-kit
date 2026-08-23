---
id: 0012
title: kit init refreshes the docs it owns
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
related: 0011-features-declare-what-they-need.md
---

## Problem
0011 added `needs:` to the template. Every project scaffolded before it still has the old
one on disk, including this repo. Nothing breaks — `kit new` writes from the embedded
template — which is what makes it worth fixing: the file has no effect, so it is pure
documentation that disagrees with the tool. Protocol changes already travel through
`AGENTS.md`'s regenerated section; format changes do not.

## Non-goals
- **Schema versioning.** Every change so far has been additive; a version field is durable state to maintain forever in a tool that claims to have none.
- **Migrating a project's own documents.** VISION, TECH, INBOX, BACKLOG and stack files are the project's; their copy is the truth.
- **A check that reports the drift.** It would fire on every project after every release, for a file with no effect, with the same fix every time.
- **Merging edits.** Replaced wholesale or not at all.

## Design
**A file is the kit's to refresh when nothing reads the project's copy of it.** That rule
sorts the scaffold: `TEMPLATE.md` and `features/README.md` are read by nothing, so replacing
them takes nothing away and keeps them true. The test is not "did the kit write it first"
but "would the project notice if its version were replaced".

**House rules already have a home** above the marker in `AGENTS.md`, which is what makes
wholesale replacement safe. Refreshed is reported, not silent — `init` already prints
`refreshed` and `kept`.

## Tasks
- [x] `kit init` refreshes the reference docs it owns, and reports it the way it already reports the protocol
- [x] A project's own documents are never touched, pinned by a test that fails if that line moves

## Log
- 2026-08-14 — schema versioning deferred deliberately: the problem is a reference doc going
  stale, fixed by regenerating it. The trigger to revisit is the first change that cannot be
  additive — a renamed field or changed meaning.
