---
id: 0078
title: Agents can take requested work through a guided kit workflow
status: Draft
created: 2026-09-06
submitted:
needs:
---

## Problem

Agents must classify work and assemble git, recordkeeping and review steps before kit
helps them. Raw git can be easier than following the protocol. Operators need agents
to carry requested work to human review with fewer manual steps and inferred decisions.

## Non-goals

Local review without PRs, a model-certification project or automatic human approval.
New lifecycle statuses, a workflow database or forge credentials managed by kit.
Parking remains 0077; this feature preserves existing eligibility checks.

## Design

Add `kit start --plan "<round>"`, `kit start <id>` for approved work, and
`kit start --work "<outcome>"` for directly requested changes. Agents translate the
operator's intent; kit prepares the branch/checkout. Plan mode retains new for feature
docs; approved work reuses claim. Direct work records its outcome in git on an isolated
spike and finishes through harvest. Remove the unrecorded chore exemption.

Add `kit finish`. For plans, validate and commit setup/docs, audit and push; keep Draft
and unticked tasks. Refuse unrelated changes. For work, require `kit finish -- <gate argv>`:
harvest if needed, request missing review facts, require complete tasks and a clean tree,
then run the supplied gate, audit, submit and push. Failed gates stop the sequence.
Retries preserve completed stages without duplicate records or stale validation.

Next names one action, checkout and responsible actor. Agents execute commands and open
PRs using the existing forge interface. Humans approve plans and merge implementations.
Handoffs name the base, candidate and known publication state; unavailable credentials or
remote evidence produce a stop. Kit never invents a PR, approval or successful check.

For a new repo, establish and publish an empty trunk base; preserve staged and unstaged
files on the branch for the first PR. Existing history stays intact. Refuse conflicting
branches, foreign claims, dirty established trunk and unreadable required state. Start
and finish must recover safely after interruption.

Preserve existing interfaces and approval guards. Never treat a pending plan as direct
work. Revisions use reopen and repeat validation; changed tips need renewed review.
Separate external permissions enforce human merge authority. Verify that boundary in
disposable fixtures and report unverified live settings. This extends 0075 and 0076.

### Agent sequences

| Request | Current assembly | Proposed route |
| --- | --- | --- |
| Draft | establish base, branch, new, commit, push, PR | start --plan → new/write → finish → PR → human review |
| Implement approved work | update trunk, claim, checkout, edit, gate, audit, submit, push | start id → edit/commit → finish with gate → PR |
| Small direct change | choose chore/spike, branch, edit, harvest, gate, audit, submit, push | start --work → edit → finish through harvest/gate → PR |

Compare manual git steps, repeated information and inferred decisions. Reduce all three
without weakening review or work preservation; an incomplete handoff is not completion.

## Tasks

- [x] Complete safe release of retained merged checkouts and report merged-but-unsubmitted work without inventing completion

- [x] Add guarded start routes with recorded intent and no unrecorded chore exemption
- [x] Preserve staged and unstaged files through recoverable empty-base setup and first-PR publication
- [x] Add plan and work finish routes with real gates, safe retries and candidate-bound validation
- [x] Make PR handoffs report actual publication state and human ownership of approval and merge
- [x] Align protocol and help with initial setup, both review gates and revisions
- [x] Rehearse fresh and existing repositories, preserved files, interrupted operations and unavailable remotes
- [ ] Verify external merge-authority boundaries and report unverified live permissions
- [ ] Compare complete draft, approved-work and small-change routes, including required human stops

## Log

- 2026-09-06 — Added recovery before guided routes after 0080 remained taken following release: its retained custom checkout kept the local claim alive. Review handoff must also direct agents to independent work from trunk, not end the session.

- 2026-09-06 — Expanded the review/setup plan to guided start and finish after the chore bypass. Retained its original outcomes; revised tasks for the wider scope and added comparative rehearsals. Implementation still awaits plan approval.
