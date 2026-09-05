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

Shipped means submitted for review until it reaches the trunk. Reset supports review
changes before integration; the integrated record stays frozen and audit checks edits
and deletions against it. This revises 0042 and 0067 without another lifecycle status.

Mutations require positive evidence for their safety. Release preserves every unmerged
commit and leases the remote tip it inspected. The board, claim and ship share validation
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

## Tasks

- [x] Harden lifecycle, claims, audit coverage and generated gates against reproduced failures
- [x] Pin and bound the Go scaffold's CI gate

- [x] Refuse reservation release when branch state is unreadable or changes concurrently

- [x] Make harvested submissions visible to audit and the review board

- [ ] Enforce current foundations and complete the review and integration handoffs
- [ ] Keep an initial harvested MVP reviewable off the trunk
- [ ] Document both workflows and remove the arbitrary global work limit

## Log

- Added the scaffold task when requiring lint exposed its missing GitLab installation.
- Applied the release safety check to reservation disposal after finding the same stale-read gap.
- 2026-09-05 — reset to Draft: Audit the harvested handoff
- 2026-09-05 — reset to Draft: Review the complete planned and exploratory workflows
