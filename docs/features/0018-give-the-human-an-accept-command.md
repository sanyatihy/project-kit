---
id: 0018
title: Give the human an accept command
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
needs: 0016, 0017
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem

`Draft → Accepted` is the only transition that belongs to a human and the only one with no
command, so the human hand-edits frontmatter — exactly what `kit ship` exists to stop. The
same failure modes are available: accepting a doc whose `## Open questions` are open, or one
a BLOCKING line is waiting on, puts an agent to work on a design nobody settled.

Where the edit happens is undefined. A draft waits on its own branch and `main` is
protected, so the flip reaches `main` either inside the draft's PR or in a second PR after
it — one round trip per feature or two, and the protocol never says which.

`kit next` never mentions that the protocol is stale. `protocol-stale` fires only under
`audit`, and agents are told to run `next`. One binary serves every project, so after a
change every other project runs superseded rules with nothing saying so.

## Non-goals

- Any `--yes` that lets an agent accept. It would remove the only gate in the model.
- Reading review state from GitHub. A PR approval is not a fact the kit can see, and an
  integration would tie it to one host.
- A `kit reject`. The states exist: leave it Draft with the question, or set Dropped.
- Warning about staleness on every command. A line on every invocation is a line nobody
  reads.

## Design

`kit accept <id>` sets `Accepted` on a Draft and refuses in three cases, each saying the
design is not settled:

| refusal | why |
|---|---|
| status is not `Draft` | nothing to accept; naming the current status is the answer |
| `## Open questions` is non-empty | those are what Draft is for |
| a BLOCKING inbox line names the id | a human already owes an answer |

The third reuses `BlockersFor`, so `accept` and `next` agree by construction rather than by
two implementations happening to match. No date is stamped, `created:` already recording
when it was proposed.

Accepting happens inside the draft's own PR: one review and one merge per feature instead of
two, and the human is reading the doc at the moment they decide.

`next` prints one line above the board when the project's protocol differs from the
binary's. Above, because an agent that has started working has walked past anything printed
below. It never changes the exit code, and `--json` carries it as a field.

## Tasks

- [x] `Accept` flips a `Draft` and refuses on a non-Draft, on non-empty `## Open questions`,
      and on a BLOCKING inbox line, reusing `BlockersFor`
- [x] `kit accept` wired in with help, overview and exit codes matching the rest
- [x] `kit next` prints the stale-protocol line above the board and carries it in `--json`,
      never changing the exit code
- [x] The status rewrite is confined to the frontmatter, shared by `Accept` and `Ship`
- [x] Protocol and README: the human accepts on the draft branch so it lands in that PR

## Log

- 2026-08-14 — the status rewrite became a fifth box rather than a silent fix. `statusLineRe`
  was applied to the whole document, so a `status:` line inside a fenced example was
  rewritten too; `Accept` would have been a second writer inheriting it.
