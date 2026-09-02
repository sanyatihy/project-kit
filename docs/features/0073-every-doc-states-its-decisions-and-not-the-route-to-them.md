---
id: 0073
title: Every doc states its decisions and not the route to them
status: Draft
created: 2026-09-01
shipped:
needs:
---

## Problem

72 feature docs, 5,199 lines, average 72; 0070 to 0072 shipped at 116 to 149. The protocol
already says a sentence earns its place by deciding, constraining or measuring something,
and to write about the system rather than the session. Both are too abstract to follow, and
the docs written under them argue in `## Non-goals`, narrate how a decision was reached in
`## Design`, and record process in `## Log`. A sibling project runs the same protocol and
averages 62 lines over 15 docs, so the difference is practice rather than rules.

## Non-goals

- Changing rule 17. A Shipped doc stays frozen; this pass is one exception and says so.
- A length check. The number that fires on a 150-line doc also fires on one whose decisions
  need the space, and a check that fires on healthy projects is worse than no check.
- Touching what a doc decided. Compaction cuts narration, never a decision, a constraint or
  a measurement.
- Retitling or re-scoping a feature. `Branch()` reads the filename, so a rename strands a
  claim.
- A target length. The longest doc in the sibling project is 95 lines.

## Design

Three rules join `### How to write`, each carrying the example the abstract version lacked:
write the decision and not the route to it; a non-goal is one line, what is excluded and
why; cite another feature by id rather than restating its reasoning. The rule about writing
about the system rather than the session is replaced by the first rather than kept beside
it — two statements of one rule is what let both be skipped.
`templates/features-TEMPLATE.md` gets the same examples, where a writer meets them.

Prose only. Length is a symptom rather than the defect, and the tier below prose is a check
that cannot tell a long doc from a verbose one.

The 69 Shipped docs are compacted in one pass, named here as the exception rule 17 does not
carry. What is frozen in practice is the decision record — status, boxes, and every
decision, constraint and measurement — and cutting words that carry none of those changes
nothing a reader would act on.

Bold lead-ins go. They restate the sentence that follows them and cost a line each in a
list of six.

A doc is as long as its decisions. One already at that density loses its bold lead-ins and
any route-narration rather than being rewritten.

## Tasks

- [ ] `### How to write` and `templates/features-TEMPLATE.md` carry the three rules, with
      this project's `AGENTS.md` regenerated in the same commit
- [ ] The five docs over 100 lines state decisions only
- [ ] Every doc of 60 lines or more states decisions only
- [ ] Every doc under 60 lines loses its bold lead-ins and its route-narration
- [ ] `docs/TECH.md` records the one-time compaction and what it may not cut

## Log
