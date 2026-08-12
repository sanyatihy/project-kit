---
id: 0007
title: A living vision agents decompose from
status: In progress
created: 2026-08-12
shipped:
check:
checked:
related: 0005-make-claims-and-blocking-mean-what-the-docs-say.md
---

## Problem
The intended loop is: a human gives a prompt, an agent drafts a feature, the human
reviews and accepts. The kit supports the last two steps and not the first. An agent
asked to draft the next feature has nowhere to read what the project is *for*.

Today that lives in the prompt. A prompt is not durable: the next agent does not have
it, and neither does the same agent after its context is compacted. So every
decomposition starts from whatever the operator remembers to restate, and two agents
drafting on different days decompose against different pictures.

The kit already has the other two corners. `docs/TECH.md` describes what exists;
`docs/features/` holds what is being done now. Nothing holds what the project is
trying to become, which is the one an agent needs to write a feature doc at all.

## Non-goals
- **Auditing whether a feature matches the vision.** That is a judgement call, and a
  check that guesses at it would fire on good features. `audit`'s only asset is being
  believed.
- **Freezing the vision.** It changes while the project is built — that is the normal
  case, not drift. Nothing here should make editing it feel like a violation.
- **A roadmap, milestones, or dates.** The board is `docs/features/`, and BACKLOG.md
  already holds what is not scheduled. A second planning surface would compete with
  both.
- **Generating features from it.** The agent reads the vision and drafts; the kit does
  not decompose anything. There is no heuristic here worth encoding.
- **Requiring it.** A project without one is not broken, so this is LOW at most, on
  the same footing as `no-tech`.

## Design
**`docs/VISION.md`, written by `kit init`, as a stub with prompts.** It completes a
triad the kit already half-has: the vision is what the project is becoming, the
feature docs are what is being done now, and the as-built doc is what exists. Each has
one owner and one lifecycle, which is the convention the kit already states as one
source of truth per fact.

**It is mutable and has no status.** Same rules as `TECH.md`: no lifecycle, no
frontmatter to keep in sync, and editing it is ordinary work rather than an exception.
The difference between the two is tense — the as-built doc must always be true now,
the vision is allowed to be aspirational and is expected to move.

**A registered `kit help vision` topic**, so an agent asked to draft a feature can
find out what the document is for without this repository.

**Protocol gains one line, not a section.** The agent reads the vision before
drafting, and a feature is decomposed to where one agent can carry it alone. Both are
prose. Neither is checkable, and pretending otherwise is how `audit` stops being
trusted.

**`no-vision` is LOW**, and only fires once a project has feature docs — before that
there is nothing to decompose and the finding would be noise on a fresh `kit init`.

## Tasks
- [x] `templates/VISION.md` — a stub with prompts, mutable, no lifecycle
- [x] `kit init` writes it, and re-running never overwrites an existing one
- [ ] `kit help vision` — a registered topic explaining what it is for and what it is not
- [ ] Audit: `no-vision` (LOW), once a project has feature docs
- [ ] Protocol: read the vision before drafting, and decompose to one agent's reach

## Open questions
<!-- none -->

## Log
- 2026-08-12 — this doc was drafted as "Put running several agents in the binary", a
  `kit help agents` topic covering worktrees, panes and process trees. Pivoted rather
  than dropped: it was still `Draft`, and the goal it was serving changed under it
  when the target became agents building a whole project from a prompt. What it
  described was the operator's tooling rather than the kit's model, which made it the
  wrong thing to spend the kit's weight on. Its substance is preserved in BACKLOG.md.
- 2026-08-12 — chose a document over a command. The alternative was teaching `kit new`
  to read the vision and pre-fill a draft, which would put the kit in the business of
  decomposing. The agent is better at that than any heuristic the kit could carry, and
  the only thing it was actually missing was somewhere durable to read from.
