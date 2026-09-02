---
id: 0007
title: A living vision agents decompose from
status: Shipped
created: 2026-08-12
shipped: 2026-08-12
related: 0005-make-claims-and-blocking-mean-what-the-docs-say.md
---

## Problem
An agent asked to draft the next feature has nowhere to read what the project is *for*.
That lives in the prompt today, and a prompt is not durable — the next agent does not have
it, and neither does the same agent after compaction, so two agents drafting on different
days decompose against different pictures. The kit already has the other two corners:
`TECH.md` is what exists, `features/` is what is being done now.

## Non-goals
- Auditing whether a feature matches the vision. A judgement call; a check that guessed would fire on good features.
- Freezing it. It moves while the project is built — that is the normal case, not drift.
- A roadmap, milestones or dates. The board is `features/` and BACKLOG.md holds what is unscheduled.
- Generating features from it. The agent decomposes; there is no heuristic here worth encoding.

## Design
`docs/VISION.md`, written by `kit init` as a stub with prompts. It completes the triad:
vision is what the project is becoming, feature docs what is being done now, as-built what
exists — one owner and one lifecycle each.

Mutable, no status. Same rules as `TECH.md`; the difference is tense — the as-built doc
must be true now, the vision is allowed to be aspirational.

A registered `kit help vision` topic, so the document's purpose is reachable without
this repository. Protocol gains one line, not a section: read the vision before drafting.

`no-vision` is LOW and fires only once a project has feature docs — before that there is
nothing to decompose and it would be noise on a fresh `init`.

## Tasks
- [x] `templates/VISION.md` — a stub with prompts, mutable, no lifecycle
- [x] `kit init` writes it, and re-running never overwrites an existing one
- [x] `kit help vision` — a registered topic explaining what it is for and what it is not
- [x] Audit: `no-vision` (LOW), once a project has feature docs
- [x] Protocol: read the vision before drafting, and decompose to one agent's reach

## Log
- 2026-08-12 — pivoted in place from "Put running several agents in the binary": still a
  Draft, and its goal changed when the target became agents building a whole project from a
  prompt. What it described was the operator's tooling, not the kit's model; substance kept
  in BACKLOG.md.
- 2026-08-12 — chose a document over teaching `kit new` to pre-fill a draft, which would put
  the kit in the business of decomposing. The agent is better at that than any heuristic;
  what was missing was somewhere durable to read from.
