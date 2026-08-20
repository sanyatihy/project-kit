---
id: 0042
title: Make review the thing that ships a feature
status: Dropped
created: 2026-08-18
shipped:
check:
checked:
review:
needs:
related: docs/features/0028-see-what-is-waiting-for-review.md, docs/features/0032-drop-the-in-progress-status.md
---

## Problem

The builder sets `Shipped` and then asks for review, so the status means *the builder
believes this is done*. Nothing in the docs or the branches records that anybody else
agreed, which is the one fact a loop running unattended most needs to carry.

What is supposed to keep the two apart is prose, at the tier `docs/TECH.md` says prose
fails — measured there: the rule against merging your own work was broken twice by an agent
that had read it, while `ship`'s coded refusal was obeyed the first time it fired. And the
signal the kit would naturally use to tell builder from reviewer does not exist: agents here
share one credential, so every commit carries the operator's name.

## Non-goals

- **Telling agents apart.** The kit cannot and will not try. Everything below is a fact
  about a commit, not about a person.
- **Running anything.** Not tests, not builds. A verdict the kit produced would be one
  nobody could audit. It says what to establish; the reviewer establishes it.
- **Merging.** The forge owns that. Required approvals on the trunk is the only real
  enforcement there has ever been, and the protocol says to turn it on.
- **A status for waiting.** 0032 removed the last one. Finished-and-waiting is derivable:
  every box ticked, on a pushed branch, not yet on the trunk.

## Design

**`Shipped` is set by a checkout that does not hold the claim.** A claim is a branch name,
so a checkout on `0042-…` *is* the claim and is refused. The reviewer works detached —
`git worktree add --detach <path> origin/<branch>` — which is what makes "not the builder"
a structural fact rather than an etiquette. `kit ship` inverts its current refusal, and that
refusal carries the builder's next instruction: push and stop, a reviewer ships this. Its
other verifications, boxes ticked and `needs:` met, stay where they are.

**The act is recorded as a ref, because a commit cannot record it.** `kit ship` creates
`refs/kit/reviewed/<id>` at the commit it reviewed, then writes the status. Ref creation is
a server-side compare-and-swap — the same primitive `claim` trusts — so one approval wins
and a second is told the work is already shipped. Verified against the forge: a `refs/kit/*`
ref is created, advertised to `ls-remote` and deleted like any other.

**The ref names the commit, so the check is exact.** The reviewed ref must be the *parent*
of the commit that set `Shipped`: the reviewer approves the work, and the status commit sits
directly on top of it. `unreviewed-ship` at HIGH reports a Shipped doc whose ref is missing
or names anything else — which is what a hand-edited status looks like, and what shipping
work that grew after it was read looks like.

**It costs no extra network call.** `remoteHeads` asks `ls-remote --heads`; dropping the
flag returns the kit refs in the same process, filtered apart locally. The git-process
ceiling in `cmd/kit/gitbudget_test.go` fails the build if that stops being true.

**What the board reads changes with it.** `AwaitingReview` stops meaning *Shipped on the
branch* and starts meaning *every box ticked, on a pushed branch, not on the trunk* — true
of docs shipped either way, so the board stays correct across the change. `done-not-shipped`
must stay quiet for exactly that state or it reports MEDIUM on every feature waiting its
turn. Both callers read one memoised set, or the per-feature `git show` is the fan-out 0039
removed.

**What this guarantees.** That the status was set by the tool rather than typed; that it was
set from a checkout not holding the claim; that the work shipped is the work reviewed, to
the commit. **What it does not.** That two different agents were involved — an agent that
releases its claim can review its own work, which is the tier the kit has always been:
escapable by not using the kit. The wall is required approvals on the remote.

## Tasks
- [ ] `AwaitingReview` reads ticked-and-pushed rather than Shipped, from one memoised set,
      and `done-not-shipped` stays quiet for that state; the git-process ceiling holds
- [ ] `kit ship` refuses the checkout holding the claim, refuses a HEAD that is not the
      branch's tip on the remote, and its refusal names what happens next
- [ ] `kit ship` creates `refs/kit/reviewed/<id>` at the reviewed commit, one winner by ref
      creation, and says so when the race is lost
- [ ] `unreviewed-ship` at HIGH: Shipped, and the reviewed ref is missing or is not the
      parent of the commit that set it; conformance for hand-edited and for grown work
- [ ] Protocol, `docs/TECH.md`, README and `kit help` carry it, including that required
      approvals on the trunk is the enforcement and the kit is the legibility
- [ ] One conformance run of the loop: build, push, ship refused for the builder, ship from
      a detached checkout, and the ref and the check agreeing at the end

## Log
- 2026-08-18 — the kit is not a test runner, so shipping records that a review happened and
  never judges the work.
- 2026-08-18 — ordering settled before any code: `Shipped` set by the builder records only
  that the builder stopped, which the branches already show.
- 2026-08-18 — identity is out as a signal, so the guarantee is about commits: a ref naming
  the reviewed commit, and a check that the shipped commit sits directly on it.
- 2026-08-20 — dropped: Parked before implementation. The design stands — Shipped set by a checkout that does not hold the claim, recorded as refs/kit/reviewed/<id> at the reviewed commit — but review by a second agent is not being run yet. Revive it when it is.
