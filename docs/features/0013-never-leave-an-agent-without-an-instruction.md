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
Two agents were launched in the same checkout. The first claimed 0001 and moved to its
worktree; the second ran `kit next`, saw a board of taken and blocked features, and started
working on the claimed one. The board ended with no closing line — every other terminal
state has one, and the state an agent reaches when a project is a chain has none. It is also
the state where the wrong move is most tempting: work that looks like help waiting to be
given.

The protocol pushed the same way — rule 9's *"do not guess, and do not idle"* reads as *find
something to do*, and nothing said a claimed feature is off limits or that stopping is ever
right.

Separately, with only drafts open `next` printed *"settle the design, then set status:
Accepted"* — instructing an agent to take the human's gate.

## Non-goals
- Detecting two agents in one checkout. The checkout is the kit's unit of identity; two agents in one are indistinguishable to it.
- Refusing to print the board. Seeing what is taken and why things are blocked is useful; the board gains a conclusion, not a censor.
- A new command or state. This is what `next` already computes, said out loud.

## Design
Every run of `next` ends with an instruction. That is the invariant — the states are:
something is yours, something is free, nothing is free, nothing exists yet.

Nothing free gets the instruction the situation calls for: everything is claimed or
waiting, a claimed feature is not yours to help with, and stopping is correct — in those
terms, because the failure was an agent concluding the opposite from silence.

Drafts stop telling an agent to accept. A tool that instructs past the project's own gate
makes the gate advisory.

## Tasks
- [x] `kit next` closes the state where nothing is free, and says stopping is right
- [x] `kit next` stops telling an agent to accept a Draft
- [x] Protocol: a claimed feature is never yours, and having nothing to do is a valid end
- [x] A test that every state `next` can reach ends with an instruction

## Log
- 2026-08-14 — third vacuum of this shape, after who merges (0009) and where a draft lives
  (0010). A state the tool can reach and does not close is one something else will decide.
