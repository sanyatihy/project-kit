---
id: 0080
title: Agents can use concise kit reference and help
status: Draft
created: 2026-09-06
submitted: 
needs:
---

## Problem

Long reference documents and repeated workflow explanations consume agent context.
Agents need concise current contracts, command effects, refusals and recovery steps.

## Non-goals

New workflow commands or changed lifecycle rules. The guided workflow is planned separately.
Go/Python checklist compression is deferred; retain the original wording.

## Design

Condense TECH, command help, protocol and embedded reference templates. Preserve command
interfaces, lifecycle rules, recovery constraints and generated house rules. Keep the
agent responsible for commands and the human responsible for approval and merge.

Restore both language checklists byte-for-byte from baseline 237ab912c288. Keep their
shortened candidates in git history for a separate, deferred comparison. The protocol
refresh test compares the actual embedded template instead of a fixed sentence.

For retained prose changes, map every baseline rule, exception and example to its
candidate equivalent. Review omissions explicitly. Word counts and make check establish
size and implementation integrity, not equivalent agent behavior.

Before resubmission, compare baseline and candidate guidance on the same predefined tasks
using the same localcode model, runner, tools and settings. Use fresh, repeated sessions
and alternate A/B order. Score task correctness, rule compliance and human intervention
before context savings. Investigate regressions and retain original wording where value
is not established. Record actual results and limits in the review evidence.

Distinguish required protocol behavior from command-enforced refusals. Command logic
remains unchanged.

## Tasks

- [x] Produce shorter TECH and embedded-help candidates while preserving generated house rules
- [x] Correct the harvest help to distinguish the planning restriction from enforced refusals

- [ ] Restore the original Go/Python checklists and defer their compression with pinned comparison references
- [ ] Account for baseline rules, exceptions and examples in the retained prose changes
- [ ] Run paired A/B tasks and retain original wording wherever equivalent value is not established

## Log

- 2026-09-06 — Added a help correction after checking the described harvest refusals against its implementation.
- 2026-09-06 — reopen to Draft: Review requires A/B evidence that prose compression preserves behavior, especially the battle-tested language checklists.
- 2026-09-06 — Operator chose to preserve both language checklists and defer compression. Reworded the completed edit task as candidate production; behavioral equivalence remains unproven and now gates resubmission.
