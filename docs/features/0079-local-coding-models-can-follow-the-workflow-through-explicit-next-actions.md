---
id: 0079
title: Local coding models can follow the workflow through explicit next actions
status: Dropped
created: 2026-09-06
submitted:
needs: 0077, 0078
---

## Problem

Agents must be able to follow kit's workflow using smaller local models. The separate
workflow redesign and model-certification scope exceeded the operator's request.

## Non-goals

Model runtime dependencies or changes to product-code quality requirements.

## Design

The proposed workflow redesign added structured next actions shared by text and JSON,
explicit human handoffs and a shorter startup protocol. Existing JSON fields and command
refusals remained compatible.

The proposed acceptance rehearsal used localcode with Qwen3.8-27B Q4_K_M, the Claude Code
harness and a 49,152-token context. Three runs per scenario covered planning, parking,
claims, failed gates, review corrections and remote failures without workflow coaching.

This feature was dropped: the operator already uses Qwen successfully and requested a
standing small-model usability constraint, not a separate redesign or certification effort.

## Tasks

- [ ] Add structured next actions shared by text and JSON while preserving existing consumers and agreement with mutation guards
- [ ] Shorten the startup route, name human-owned actions explicitly and make every supported workflow state lead to an actionable command or an explicit stop
- [ ] Verify the full action and refusal matrix, including parking, initial review, revision and unavailable evidence
- [ ] Provide reproducible local-model rehearsal scenarios and record actual results against the agreed model without substituting a frontier-model run

## Log
- 2026-09-06 — dropped: The operator has already used Qwen successfully; keep small-model usability as a standing design constraint, not a separate workflow redesign or model-certification feature.
