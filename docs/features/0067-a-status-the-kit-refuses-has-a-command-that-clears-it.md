---
id: 0067
title: A status the kit refuses has a command that clears it
status: Draft
created: 2026-08-24
shipped:
needs: 0066
---

## Problem

A `status:` the kit does not recognise wedges the doc that carries it. `claim` exits 1 —
"it is not open work"; `next` lists the feature as blocked, `waiting on: unknown status In
Progress — set it to Draft`; `audit`'s `no-status` fix reads `use one of: Draft, Dropped,
Shipped`. Every one of those instructions is a hand-edit of `status:`, which is both what
rule 6 forbids and how the bad value arrived. The only command out is `kit drop`, which
retires the feature instead of repairing it. An unattended chain met this and complied
literally, committing a hand-edit to get its doc back.

## Non-goals

- **Letting `ship` overwrite an unrecognised status.** It already does, and shipping is not
  the way out of a doc whose boxes are still open.
- **Reopening a Shipped doc.** Frozen history: the work is in the repo, and a status saying
  otherwise would make the record lie.
- **Guessing which status was meant.** `Draft` is the only one that claims nothing, and it
  is where a doc the kit cannot read belongs.

## Design

**`kit reset <id>` returns a doc to `Draft`.** It is the third command that writes a status,
which is what keeps the field the kit's to write — the property the wedge was forcing agents
to break. It accepts any status but `Shipped`, so one rule covers both the unrecognised
value and reviving a `Dropped` plan, and it takes a reason it writes to `## Log` for the
same argument `drop` makes: a status change nobody explained reads as a mistake later.

**It takes `ship`'s guard on someone else's claim.** A doc claimed by another checkout is
work in flight, and resetting its status from here is the same interference as shipping it.

**It commits what it writes**, which is why this waits on 0066. A repair loose in the
working tree is the failure 0066 closes.

**Every instruction to set a status by hand names the command instead.** `no-status`'s
`fix:` line and `next`'s blocked reason are the two, and they are why the chain's model
repaired the damage the way it did: a `fix:` naming an exact command was followed every
time, and the one finding whose fix was prose is the one it botched.

## Tasks

- [x] `kit reset <id> "<why>"` sets any non-Shipped doc to `Draft`, logs the reason, and
      refuses a `Shipped` doc and another checkout's claim, with a conformance case
- [ ] `kit help` and the help topics list it beside `ship` and `drop`
- [ ] `no-status`'s `fix:` line and `next`'s blocked reason name it instead of an edit

## Log
