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

Add `kit plan <round>` as the guarded entry into planned work. It creates a `plan/`
branch whose name has no feature id, from an updated trunk. In a repository with no
commits it first establishes an empty trunk commit, keeping every staged and unstaged
file off that commit, then switches to the planning branch with the files preserved.
Setup and feature docs are reviewed together on that branch. `kit init` remains the
idempotent file scaffolder; it does not itself approve or commit setup. This replaces
0075's direct setup commit advice and shares the empty-base contract already used by harvest.

The command reports the review base and branch. With a remote, it names publishing the
empty trunk base as the prerequisite to the first planning PR; it never pushes setup or
product files to trunk. Existing advertised remote history, a dirty established trunk,
a conflicting branch, a foreign claim or unavailable required git evidence must refuse
without losing files. An interrupted invocation must have a command-driven retry.
Existing projects branch from their current trunk without creating a new root. No-commit
spikes keep using harvest and the same empty-base recovery rules. Without a remote,
planning can be saved locally; the PR handoff waits for the operator to supply one.

Make every PR handoff name its target trunk, source branch, exact local and published
commits, and whether the review base has been published. The board and command output
must distinguish committing a plan, publishing it, requesting human review, addressing
feedback and waiting for human integration. A missing base or unreadable remote cannot
produce a complete handoff. Do not infer an open PR, approval or passing checks from a
branch or Submitted stamp. Git remains the only runtime dependency; opening and inspecting
a PR uses the operator's existing forge interface outside kit.

Keep both human gates: plan before implementation and implementation before integration.
Planning comments are addressed on the same plan branch. Implementation comments use
reopen, revised tasks, implementation, gate, audit, submit and a new push to the existing
PR. A changed candidate needs review of the new tip. Agent-facing next actions stop at
the review handoff and never prescribe self-merge. Claim eligibility still requires
the approved plan on the current published trunk; submission alone never unlocks work.

Document how the operator reserves direct trunk writes and final merge authority outside
the agent's credentials, and how to verify that separation with a disposable repository.
Cover the empty-base publication before protection and the first PR after protection.
Required tests alone do not prove human approval. An agent with the operator's credentials
or an available bypass can still self-merge; a status or local approval marker cannot fix
that. Distinguish kit's tested refusals from verified remote permission enforcement and
from settings that remain unverified. This extends the review boundary stated in 0076.

## Tasks

- [ ] Establish a recoverable empty review base and planning branch without committing staged or unstaged user files to trunk
- [ ] Make PR handoffs name the review base and actual publication state, with explicit human ownership of approval and merge
- [ ] Make the printed protocol and help cover initial setup, both PR approval gates and review revisions without directing agents to merge
- [ ] Rehearse fresh planned and spike repositories, preserved staged files, existing repositories, unavailable remotes and first-PR publication in disposable fixtures
- [ ] Document and verify the external merge-authority boundary, reporting unverified live permissions explicitly

## Log
