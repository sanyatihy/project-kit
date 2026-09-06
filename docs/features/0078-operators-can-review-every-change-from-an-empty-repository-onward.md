---
id: 0078
title: Operators can review every change from an empty repository onward
status: Draft
created: 2026-09-06
submitted:
needs:
---

## Problem

Operators cannot review every initial change when planned-project setup commits files
directly to trunk. An agent that merges its own plan can then claim it because kit checks
integration, not the identity that approved it. Every setup, plan and implementation
change needs a clear PR handoff owned by the human reviewer.

## Non-goals

A complete local review workflow. The operator chose PR review first; retain that later
need in BACKLOG without adding local approval state in this round.
An identity service or approval token an agent can write for itself. Git remains the record.
Automatically changing forge permissions or merging agent work. Human authority stays separate.
Rewriting an existing project's initial commit. Preserve its history and files.

## Design

Add `kit plan <round>` to create a plan branch with no feature id from updated trunk.
With no commits, create an empty trunk base and preserve staged and unstaged files on the
planning branch. Review setup and plans in the first PR. Init remains a file scaffolder;
harvest shares the empty-base contract. Existing history is preserved.

Print the empty-base push before the first PR; never push setup or product files to trunk.
Refuse conflicting branches, foreign claims, dirty established trunk and unreadable remote
evidence. Detect existing remote history before creating a base. Interrupted setup must
be safely retryable. Without a remote, save the plan and wait for one.

PR handoffs name the base, source branch, local and published commits, and next responsible
actor. Distinguish commit, push, request review, revise and wait for human merge. Never infer
an open PR, approval or passing tests from git state. Forge interaction stays outside kit;
git remains its only runtime dependency.

Humans review both plans and implementations. Plan corrections stay on the planning branch;
implementation corrections use reopen and repeat the gate, audit, submit and push sequence.
Changed tips need renewed review. Agent instructions never prescribe self-merge. Claim
requires the plan on current published trunk.

Document and rehearse empty-base publication, branch protection and the first PR in a
disposable repository. Reserve direct trunk writes and final merge authority outside agent
credentials. Required tests alone do not enforce human approval. Report verified permissions
separately from unverified settings. This extends 0076's review boundary.

## Tasks

- [ ] Establish a recoverable empty review base and planning branch without committing staged or unstaged user files to trunk
- [ ] Make PR handoffs name the review base and actual publication state, with explicit human ownership of approval and merge
- [ ] Make the printed protocol and help cover initial setup, both PR approval gates and review revisions without directing agents to merge
- [ ] Rehearse fresh planned and spike repositories, preserved staged files, existing repositories, unavailable remotes and first-PR publication in disposable fixtures
- [ ] Document and verify the external merge-authority boundary, reporting unverified live permissions explicitly

## Log
