---
id: 0013
title: Never leave an agent without an instruction
status: Shipped
shipped: 2026-08-14
created: 2026-08-14
needs:
related: 0011-features-declare-what-they-need.md
---

## Problem
Two agents were launched in the same main checkout, which is the correct entry point:
`kit claim` is what moves an agent into a worktree, and the first agent did exactly
that. The second ran `kit next`, saw this, and started working on the feature the
first one had already claimed.

```
  taken
    0001  0/5   Serve HTTP with config and health che… — 0001-serve-http…
  blocked
    0002 … 0007   waiting on: needs … to ship first
```

That output ends there. No instruction, no closing line, nothing saying that none of
it is available. Every other terminal state has one — "nothing open — start a
feature", "settle the design". The state an agent actually reaches when a project is
a chain has none, and it is the state where the wrong move is most tempting: a board
full of work that looks like help waiting to be given.

The protocol pushed the same way. Rule 9 says *"Do not guess, and do not idle"*, which
reads as *find something to do*. Nothing anywhere says a claimed feature is off limits
or that stopping is ever the right answer. The agent followed the rules as written.

There is a second contradiction in the same function. With only drafts open, `next`
prints *"settle the design, then set status: Accepted"* — instructing the reader to do
the one thing rule 1 forbids an agent to do. The tool tells it to take the human's
gate.

## Non-goals
- **Detecting two agents in one checkout.** The checkout is the kit's unit of
  identity and two agents standing in one are indistinguishable to it. Nothing here
  pretends otherwise; the answer is that the second one is told to stop.
- **Refusing to print the board.** Seeing what is taken and why things are blocked is
  useful, and hiding it would trade one confusion for another. The board stays; it
  gains a conclusion.
- **A new command or state.** This is what `next` already computes, said out loud.
- **Making `next` write anything.** It reads only, and telling an agent to stop is
  still only telling.

## Design
**Every run of `next` ends with an instruction.** That is the invariant, and it is
what was missing rather than any particular sentence. The states are: something is
yours, something is free, nothing is free, and nothing exists yet. Three of the four
already closed.

**Nothing free gets the instruction the situation actually calls for.** Everything is
claimed or waiting, a claimed feature is not yours to help with, and stopping is the
correct outcome — said in those terms, because the failure was an agent concluding
the opposite from silence.

**Drafts stop telling an agent to accept.** Rule 1 makes accepting a human's call, so
`next` says the drafts are waiting on a human and offers the reader the thing it may
actually do: draft another, or stop. A tool that instructs an agent past the project's
own gate makes the gate advisory.

**The protocol gains the rule the vacuum needed.** A feature under `taken` is never
yours: not to work, not to tick, not to "help with". And having nothing to do is a
valid end — an agent that stops and says so is behaving correctly, which rule 9's
"do not idle" currently reads as forbidding.

## Tasks
- [x] `kit next` closes the state where nothing is free, and says stopping is right
- [x] `kit next` stops telling an agent to accept a Draft
- [x] Protocol: a claimed feature is never yours, and having nothing to do is a valid end
- [x] A test that every state `next` can reach ends with an instruction

## Open questions
<!-- none -->

## Log
- 2026-08-14 — the third vacuum of the same shape, after merging in 0009 and where a
  Draft lives in 0010. Each time the documented path ran out mid-flow and an agent
  filled the gap sensibly and wrongly. The pattern is worth naming: a state the tool
  can reach and does not close is a state something else will decide.
