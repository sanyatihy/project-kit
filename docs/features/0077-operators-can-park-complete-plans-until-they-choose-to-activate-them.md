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

Add Parked and two commands: `kit park <id> "<reason>"` and `kit activate <id>`.
Agents call them when the operator asks to park or activate work.
Parking preserves the id, design, dependencies and tasks. Activation returns it to Draft;
Dropped remains permanent retirement. Existing Draft behavior stays unchanged.

Both transitions commit only the target feature on a planning branch and record the
reason in Log. They require human review through merge before changing shared availability.
Parking requires a complete, unstarted Draft; activation revalidates Parked. Refuse active
claims, terminal history, ambiguous documents, uncommitted target edits and unknown remote
state. Repeated transitions are no-ops. Reopen cannot activate parked work.

Next shows parked plans separately in text and additive JSON, with a reason and no build
action. Claim refuses them, including stale-clone attempts. Dependents wait for implemented,
integrated prerequisites. Parking creates no stalled-work or unanswered-question warning;
ordinary document validation still applies.

Use a status so older clients refuse claims rather than ignore a new eligibility field.
Verify previous-version refusal and document coordinated upgrades and repair limitations.
This extends 0075's lifecycle.

## Tasks

- [ ] Preserve full plans through command-owned parking and activation transitions, with safe refusal and retry behavior
- [ ] Make next, claim, dependencies and audit agree on parked availability, including activation awaiting merge and stale-clone attempts
- [ ] Document operator activation, distinguish parking from blockers and claim release, and verify text, JSON and previous-version safety

## Log
