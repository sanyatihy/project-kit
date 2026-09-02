---
id: 0042
title: Make review the thing that ships a feature
status: Dropped
created: 2026-08-18
shipped:
needs:
related: docs/features/0028-see-what-is-waiting-for-review.md, docs/features/0032-drop-the-in-progress-status.md
---

## Problem

The builder sets `Shipped` and then asks for review, so the status means the builder
believes this is done. Nothing in the docs or the branches records that anybody else agreed,
which is the one fact a loop running unattended most needs to carry.

What keeps the two apart is prose, at the tier `docs/TECH.md` says prose fails: the rule
against merging your own work was broken twice by an agent that had read it, while `ship`'s
coded refusal was obeyed the first time it fired. And the signal the kit would naturally use
to tell builder from reviewer does not exist — agents here share one credential, so every
commit carries the operator's name.

## Non-goals

- Telling agents apart. Everything below is a fact about a commit, not about a person.
- Running anything. A verdict the kit produced would be one nobody could audit.
- Merging. The forge owns it, and required approvals on the trunk is the only real
  enforcement there has ever been.
- A status for waiting. Finished-and-waiting is derivable: every box ticked, on a pushed
  branch, not yet on the trunk.

## Design

`Shipped` is set by a checkout that does not hold the claim. A claim is a branch name, so a
checkout on `0042-…` is the claim and is refused. The reviewer works detached, which makes
"not the builder" a structural fact rather than an etiquette. `kit ship` inverts its current
refusal, and that refusal carries the builder's next instruction.

The act is recorded as a ref, because a commit cannot record it. `kit ship` creates
`refs/kit/reviewed/<id>` at the commit it reviewed, then writes the status. Ref creation is a
server-side compare-and-swap — the primitive `claim` already trusts — so one approval wins
and a second is told the work is already shipped. Verified against the forge: a `refs/kit/*`
ref is created, advertised to `ls-remote` and deleted like any other.

The reviewed ref must be the parent of the commit that set `Shipped`: the reviewer approves
the work and the status commit sits directly on top of it. `unreviewed-ship` at HIGH reports
a Shipped doc whose ref is missing or names anything else, which is what a hand-edited status
looks like and what shipping work that grew after it was read looks like.

It costs no extra network call. `remoteHeads` asks `ls-remote --heads`; dropping the flag
returns the kit refs in the same process, filtered apart locally.

`AwaitingReview` stops meaning Shipped-on-the-branch and starts meaning every box ticked, on
a pushed branch, not on the trunk — true of docs shipped either way, so the board stays
correct across the change. `done-not-shipped` must stay quiet for exactly that state or it
reports MEDIUM on every feature waiting its turn.

This guarantees that the status was set by the tool rather than typed, from a checkout not
holding the claim, and that the work shipped is the work reviewed to the commit. It does not
guarantee two different agents: one that releases its claim can review its own work. The
wall is required approvals on the remote.

## Tasks

- [ ] `AwaitingReview` reads ticked-and-pushed rather than Shipped, from one memoised set,
      and `done-not-shipped` stays quiet for that state
- [ ] `kit ship` refuses the checkout holding the claim, refuses a HEAD that is not the
      branch's tip on the remote, and its refusal names what happens next
- [ ] `kit ship` creates `refs/kit/reviewed/<id>` at the reviewed commit, one winner by ref
      creation
- [ ] `unreviewed-ship` at HIGH: Shipped, and the reviewed ref is missing or is not the
      parent of the commit that set it
- [ ] Protocol, `docs/TECH.md`, README and `kit help` carry it, including that required
      approvals on the trunk is the enforcement and the kit is the legibility
- [ ] One conformance run of the loop, ending with the ref and the check agreeing

## Log

- 2026-08-18 — the kit is not a test runner, so shipping records that a review happened and
  never judges the work.
- 2026-08-18 — identity is out as a signal, so the guarantee is about commits: a ref naming
  the reviewed commit, and a check that the shipped commit sits directly on it.
- 2026-08-20 — dropped before implementation. The design stands; review by a second agent is
  not being run yet. Revive it when it is.
