---
id: 0080
title: Agents can use concise kit reference and help
status: Submitted
created: 2026-09-06
submitted: 2026-09-06
needs:
---

## Problem

Long reference documents and repeated workflow explanations consume agent context.
Agents need concise current contracts, command effects, refusals and recovery steps.

## Non-goals

New workflow commands or changed lifecycle rules. The guided workflow is planned separately.

## Design

Condense TECH, command help, protocol and embedded reference templates. Preserve command
interfaces, lifecycle rules, recovery constraints and generated house rules. Keep the
agent responsible for commands and the human responsible for approval and merge.

Remove incident narratives and duplication. Keep language checklists as review aids;
preserve their practices without adding a model-certification requirement. The protocol
refresh test compares the actual embedded template instead of a fixed sentence.

Distinguish required protocol behavior from command-enforced refusals. Command logic
remains unchanged.

## Tasks

- [x] Reduce TECH and embedded help while preserving workflow contracts, recovery guidance and generated house rules
- [x] Correct the harvest help to distinguish the planning restriction from enforced refusals

## Log

- 2026-09-06 — Added a help correction after checking the described harvest refusals against its implementation.
