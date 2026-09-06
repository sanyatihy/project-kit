---
id: 0079
title: Local coding models can follow the workflow through explicit next actions
status: Dropped
created: 2026-09-06
submitted:
needs: 0077, 0078
---

## Problem

Kit serves operators using local coding models as well as frontier models. A caller
currently has to combine a long protocol, command help and branch conventions to choose
its next safe action. Ambiguous directions such as "land it" or "draft a feature, or stop"
leave room to invent work or merge a plan without the operator.

## Non-goals

Certifying every small model or measuring product-code quality. This tests following kit's workflow.
A model runtime dependency in kit or model-specific protocols. All agents use the same commands.
A second workflow state machine. Derive instructions from the same evidence as command refusals.

## Design

Make a supported local model completing the workflow a stated VISION outcome. The short
entry contract is: follow the operator's requested scope, run kit next, perform the one
permitted next action in the named checkout, and stop when the action belongs to the human.
Reading a board does not authorize new planning, activation of parked work, implementation
of an unapproved plan or merging one's own work. Intent still comes from the operator;
kit cannot infer it from repository state.

Replace ambiguous transition wording with explicit actor, checkout and action. For a
pending plan, say the agent opens or updates the planning review and the human merges.
For no available work, say to stop and request direction; describe drafting only when
planning is already authorized. Feature and planning commands should create or name
branches and worktrees, so callers do not reconstruct naming rules from prose.

Extend next JSON with one structured next-action object, without changing existing
fields. It names the action kind, responsible actor, working directory, relevant feature
or branch, argument array for a kit invocation where available, and the reason for waiting
or refusal. External actions such as human review are typed handoffs, not shell strings
for automatic execution. Use the same action representation to render the text closing
instruction. Existing ready choices remain visible; the recommended action selects one
deterministically when scope permits and does not claim it on the reader's behalf.

Cover planning, parked-only boards, claims, the first unfinished task, gate and submit,
PR review, revisions, integration, missing setup, unknown remote state and recovery. Do not declare tests passed from task boxes or make an unavailable remote look
like local-only mode. Refusals name the recoverable cause and a safe next step without
suggesting deletion of another checkout's work. Keep detailed language checklists and
reference material in reachable help instead of expanding the startup protocol.

Use deterministic conformance cases for state transitions and a separate opt-in model
rehearsal through the operator's localcode tool using the same generated AGENTS.md and
tool output available to any project.
The acceptance target is the operator's localcode setup: Qwen3.8-27B Q4_K_M via the
Claude Code harness, with config/agent.env's 49,152-token serving context. Identify the
actual installed build and served settings at rehearsal time. Do not substitute a frontier model or enlarge the context to
make the workflow pass. Record the exact model, quantization, runner, context and sampling
settings, prompts, commands, interventions and outcome. Scenarios include initial planning and a required
human stop, parked work, an approved claim, a failed gate, review corrections and remote
failure. Three fresh runs per scenario must avoid unauthorized implementation, self-merge,
parked claims and destructive recovery, and reach the expected handoff without workflow
coaching. Scripted refusals alone are not evidence that a model can follow the whole flow.
Failures remain evidence to fix, not a reason to soften the approval boundaries. The
rehearsal stays outside the ordinary CI gate so development does not require model access.

## Tasks

- [ ] Add structured next actions shared by text and JSON while preserving existing consumers and agreement with mutation guards
- [ ] Shorten the startup route, name human-owned actions explicitly and make every supported workflow state lead to an actionable command or an explicit stop
- [ ] Verify the full action and refusal matrix, including parking, initial review, revision and unavailable evidence
- [ ] Provide reproducible local-model rehearsal scenarios and record actual results against the agreed model without substituting a frontier-model run

## Log
- 2026-09-06 — dropped: The operator has already used Qwen successfully; keep small-model usability as a standing design constraint, not a separate workflow redesign or model-certification feature.
