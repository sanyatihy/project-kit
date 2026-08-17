---
id: 0018
title: Give the human an accept command
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
check:
checked:
needs: 0016, 0017
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem
`Draft → Accepted` is the only transition that belongs to a human, and the only one with no
command — so the human hand-edits frontmatter, which is exactly what `kit ship` exists to
stop. The same failure modes are available: accepting a doc whose `## Open questions` are
open, or one a BLOCKING line is waiting on, puts an agent to work on a design nobody settled.

It is also undefined *where* the edit happens. A draft waits on its own branch and `main` is
protected, so the flip reaches `main` either inside the draft's PR or in a second PR after
it — one round trip per feature or two, and the protocol never says which.

Separately: **`kit next` never mentions that the protocol is stale.** `protocol-stale` fires
only under `audit`, and agents are told to run `next`. One binary serves every project, so
after a change every other project runs superseded rules with nothing saying so — the exact
failure 0009 recorded.

## Non-goals
- **Any `--yes` that lets an agent accept.** An agent that could run it would remove the only gate in the model.
- **Reading review state from GitHub.** A PR approval is not a fact the kit can see, and an integration would tie it to one host.
- **A `kit reject`.** The states exist: leave it Draft with the question, or set Dropped.
- **Warning about staleness on every command.** `next` is where a session starts; a line on every invocation is a line nobody reads.

## Design
**`kit accept <id>`** sets `Accepted` on a Draft and refuses in three cases, each saying the
design is not settled:

| refusal | why |
|---|---|
| status is not `Draft` | nothing to accept; naming the current status is the answer |
| `## Open questions` is non-empty | those are what Draft is for |
| a BLOCKING inbox line names the id | a human already owes an answer; `next` would refuse it anyway |

The third reuses `BlockersFor`, so `accept` and `next` agree by construction rather than by
two implementations happening to match. No date is stamped: `created:` already records when
it was proposed.

**Accepting happens inside the draft's own PR** — one review and one merge per feature
instead of two, and the human is reading the doc at the moment they decide. The closing
instruction and the protocol both say so.

**`next` prints one line above the board** when the project's protocol differs from the
binary's. Above, because an agent that has started working has walked past anything printed
below; it never changes the exit code, and `--json` carries it as a field.

## Tasks
- [x] `internal/kit/accept.go`: `Accept` flips a `Draft` and refuses on a non-Draft, on
      non-empty `## Open questions`, and on a BLOCKING inbox line, reusing `BlockersFor`;
      unit tests for all four paths.
- [x] `kit accept` wired into `main.go` with `commandHelp`, the overview, and exit codes
      matching the rest (2 for a refusal, since accepting is not a race); conformance for
      the happy path and each refusal.
- [x] `kit next` prints the stale-protocol line above the board and carries it in
      `--json`; conformance that it appears with an old AGENTS.md, is absent with a
      current one, and never changes the exit code.
- [x] Confine the status rewrite to the frontmatter. `statusLineRe` is `(?m)^status:.*$`
      applied to the whole document, so a `status:` line inside a fenced example is
      rewritten too. Found while designing `Accept`, which would otherwise be a second
      writer inheriting the same defect; shared with `Ship` rather than duplicated.
- [x] Protocol and README: the human accepts with `kit accept` on the draft branch so it
      lands in that PR, and the new command appears in README's command list.

## Log
- 2026-08-14 — the status rewrite became a fifth box rather than a silent fix: `Accept`
  needed a writer, and the existing one rewrote `status:` lines inside fenced examples too.
  `SetField` is shared by both commands rather than the defect being duplicated.
