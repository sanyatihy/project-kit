---
id: 0084
title: Rules an agent must remember become things the kit carries or does
status: Draft
created: 2026-09-10
submitted:
needs:
---

## Problem

Three rules ask an agent to remember something the binary could carry or perform, and
feedback from nine features run through the protocol in another project named all three.
Money handling in Python — decimal arithmetic, minor units, no coercion at the edge — is a
rule every project touching money rewrites, and `kit help python-checklist` exists so that
no project has to. The checklists are invisible: rule 9 tells an agent to read one, which
is a rule about reading a rule, and an agent that does not already know they exist never
runs the command. And pausing work is rule 13's four-clause sequence held in prose, where
every other multi-step sequence the protocol described has become a command.

## Non-goals

- Asking an author which checklists they consulted. It is self-reported and unverifiable,
  and `audit` is believed because it never guesses.
- A `review-checklist` page. The proposal that prompted this encodes one application's
  concurrency and idempotency model, which does not belong in a binary every project
  installs. A project's own rules go above the protocol marker in `AGENTS.md`, which
  `init` preserves.
- Budgeting doc or report length. Built twice and withdrawn twice — 0038's `doc-bloat`
  by 0049, 0074's ratio by 0075, the second because the metric could not establish
  misplaced content.
- A status for paused work. Only `submit`, `drop`, `reopen`, `park` and `activate` write
  status, and a pause is not a lifecycle change.

## Design

One principle, three moves: a rule an agent has to remember becomes something the binary
carries or performs.

`python-checklist` gains money handling, in the boundary and datastore sections it already
has. Decimal in the domain, integer minor units in storage, decimal text at the HTTP
boundary, `json.loads(..., parse_float=Decimal)` wherever money is read, and no `str()` or
`float()` coercion of a request field. Naming libraries is already this page's convention,
so the rules land where a Python author is looking.

`kit start` prints the checklist topic for the stack it finds, on the exit-0 path beside
the worktree it names. There is no stack detection today — `DetectForge` has no
counterpart — so the marker file that `init --stack` writes is what names it: `go.mod` for
Go, `pyproject.toml` for Python, nothing printed when neither is there. Printing is the
whole of it. The agent that needed the reminder gets it as it is about to write code, and
the agent that did not loses one line.

`kit pause "<why>"` performs rule 13: it commits what is in the tree, appends
`- <date> — paused: <why>` to `## Log`, pushes the branch, and prints the pull-request
handoff naming what remains. The claim is retained, so this is not `release`. The Log line
follows `park`, which already records a scheduling decision there and reads it back
through `ParkingReason`; `next` reads this one the same way and shows the checkout as
still holding its claim, with the reason, so it is offered no other work.

## Tasks

- [x] `kit help python-checklist` carries money handling: decimal arithmetic, minor units
      in storage, boundary parsing, and no coercion of a request field
- [ ] The stack is detected from the marker file `init --stack` writes
- [ ] `kit start` prints the checklist topic for that stack on a won claim
- [ ] `kit pause "<why>"` commits, records the reason in `## Log`, pushes, and prints the
      handoff, retaining the claim
- [ ] `kit next` shows a paused claim as still yours, with its reason
- [ ] `kit help pause`, the command list, rule 13 in `templates/AGENTS-section.md` and
      `docs/TECH.md` carry it

## Log
