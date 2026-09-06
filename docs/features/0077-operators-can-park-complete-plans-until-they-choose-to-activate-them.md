---
id: 0077
title: Operators can park complete plans until they choose to activate them
status: Draft
created: 2026-09-06
submitted:
needs:
---

## Problem

Operators need to keep fully decomposed plans without authorizing their implementation.
Today a complete Draft on trunk is claimable, while Dropped retires it permanently.
A blocking question misrepresents a scheduling choice and eventually creates audit noise.

## Non-goals

Dates, priorities or a roadmap. The operator only chooses whether work is available.
Pausing an active claim. Preserve and hand off ongoing work through the existing workflow.

## Design

Add status Parked and commands `kit park <id> "<reason>"` and
`kit activate <id>`. Parked keeps the same id, design, dependencies and task boxes;
activation returns it to Draft. Dropped keeps its existing meaning. This extends 0075's
lifecycle without changing the meaning or availability of existing Draft documents.

Parking and activation are reviewed changes on a planning branch. The command commits
only its feature change and records the transition reason in Log; activation records
that the operator chose to make the plan available. Neither command merges or acquires
a claim. The published trunk determines availability across clones: parking becomes
shared after merge, and activation does not permit a claim before its own review merge.
An operator can draft, finish and park a new plan on the same planning branch.

Both commands require one unambiguous document and refuse active claims, terminal
history, unrelated dirty changes to the target and unreadable remote evidence. Parking
requires a complete unstarted Draft; activation requires Parked and validates the plan
again. Repeating an already completed transition is a no-op. Reopen, including repair,
must not become an alternative way to activate a parked feature.

`kit next` lists parked plans separately in text and an additive JSON group, with their
reason and no implementation action. A board containing only parked plans says to wait
for operator activation. `claim` refuses Parked consistently, including direct calls
and stale checkouts. A dependent remains blocked until its prerequisite is implemented
and integrated, not merely activated. Parking is not stalled work or an unanswered question;
malformed documents, leaked secrets and other real drift remain auditable.

Use a status rather than an optional eligibility field: an older kit already refuses to
claim a status other than Draft, whereas it could ignore a new field and start parked work.
Verify that behavior against the supported previous binary. Update all writers before
using the commands; document older audit and repair limitations without rewriting history.

## Tasks

- [ ] Preserve full plans through command-owned parking and activation transitions, with safe refusal and retry behavior
- [ ] Make next, claim, dependencies and audit agree on parked availability, including activation awaiting merge and stale-clone attempts
- [ ] Document operator activation, distinguish parking from blockers and claim release, and verify text, JSON and previous-version safety

## Log
