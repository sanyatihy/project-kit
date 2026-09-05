---
id: 0075
title: Agents can complete and revise work without silently bypassing the protocol
status: Draft
created: 2026-09-05
shipped: 
needs:
---

## Problem

Agents following the printed workflow can lose concurrent work, pass an incomplete
gate, or need a forbidden hand edit to address review feedback. Passing the existing
suite did not establish that the protocol's refusals were complete or consistent.

## Non-goals

A new workflow engine or identity store. Markdown and git remain the state.
Controlling raw git with prose. Merge authorization belongs on the remote.

## Design

Submit records Submitted on the feature branch; integration is derived from git.
Reopen returns an unmerged submission to Draft. Legacy Shipped documents remain readable
and frozen on trunk; ship/reset remain compatibility spellings. Routine review and
explicit repair or disposal have distinct command contracts. This revises 0042 and 0067.

Mutations require positive evidence for their safety. Release preserves every unmerged
commit and leases the remote tip it inspected. The board, claim and submit share validation
of ambiguous plans and dependencies. Unknown remote state fails strict online audit;
an explicit offline request reports its narrower coverage.

The protocol states the sequence and ownership boundaries once. Remove 0074's Log
ratio gate: a short legitimate review reason can exceed half a short plan, so the metric
cannot establish misplaced content. Durable facts still move to TECH before submission.

Planned work has two human review points: the plan and the implementation. Exploration
joins at implementation review, including an MVP with no commits. Dependencies become
available only after integration; a submission stamp alone is not a foundation. The
board distinguishes local completion, pushed review, requested revisions and integration,
including squash merges. Each checkout holds one feature without a global agent limit.

Gates enforce correctness and preservation of work. Project age is advisory. Remove
editorial proxies and count-based implementation tests; retain behavioral coverage and
measure subprocess growth rather than incidental totals. Features include useful
subtractions, security, reliability and performance. Prefer the largest coherent outcome
that can be reviewed and integrated safely; implementation steps stay task boxes.

## Tasks

- [x] Harden lifecycle, claims, audit coverage and generated gates against reproduced failures
- [x] Pin and bound the Go scaffold's CI gate

- [x] Refuse reservation release when branch state is unreadable or changes concurrently

- [x] Make harvested submissions visible to audit and the review board

- [x] Enforce current foundations and complete the review and integration handoffs
- [x] Keep an initial harvested MVP reviewable off the trunk
- [x] Document both workflows and remove the arbitrary global work limit

- [x] Verify frozen history against the published integration state

- [x] Make gates fail correctly and test behavior rather than editorial proxies
- [x] Clarify submission, reopening and explicit disposal command contracts
- [ ] Teach substantial value-shaped features and align the workflow documentation

## Log

- Extended the freeze check after the lifecycle tests exposed its remaining reliance on local trunk state.

- Added the scaffold task when requiring lint exposed its missing GitLab installation.
- Applied the release safety check to reservation disposal after finding the same stale-read gap.
- 2026-09-05 — reset to Draft: Audit the harvested handoff
- 2026-09-05 — reset to Draft: Review the complete planned and exploratory workflows
- 2026-09-05 — reset to Draft: Tighten command contracts and gates; favor substantial value-shaped features
