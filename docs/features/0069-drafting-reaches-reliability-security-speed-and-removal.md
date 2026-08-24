---
id: 0069
title: Drafting reaches reliability, security, speed and removal
status: Draft
created: 2026-08-24
shipped:
needs:
---

## Problem

The protocol gives one route from the vision to a draft: split `## What done looks like`,
where each observable outcome is at least one feature. `## Constraints` and `## What it is
not` are described only as things that reject a feature, so work that makes a constraint
true — a failure that stops happening, an exposure that closes, a number that crosses a
line — has no route to a draft at all. Rule 2 compounds it: "one shipped increase" reads as
addition, and both of its example sentences contrast a good name with a refactor name, so a
reader never sees a passing sentence that is not a new capability. This project shipped
0006, 0008, 0032, 0035, 0039 and 0048 anyway; an operator drafted them against the prose
rather than with it, and an agent reading the protocol cold would not have.

## Non-goals

- **A check that reports an all-additive round.** It would have to classify a feature's
  kind from its title, which is a guess, and a check that fires on healthy projects is
  worse than no check.
- **A `kind:` field in the template.** Every doc pays for it for ever, the tool cannot
  validate it, and the judgement it records is made once per round.
- **Saying what a round should contain.** The protocol says what counts as a feature.
  Which ones are worth building is the operator's, and no rule can hold that.

## Design

**Every section of the vision can produce a feature, or it is decoration.** Rule 1 gains
the second route: a constraint the project does not yet honour is a feature, and
`## Constraints` and `## What it is not` generate work as well as filter it. This is the
structural half — the definition can be read generously and still reach nothing, because
nothing sends a planner to the sections where this work lives.

**Rule 2 is change-shaped rather than increase-shaped.** A feature is one shipped change in
what the project can do, can be trusted to do, or can be trusted not to do. 0061 kept
"increase ... or can be trusted to do" as the useful half of the old rule and that judgement
stands; what changes is its reach, which never covered a capability removed with its cost,
and covered speed only by argument. Adding a capability is one shape of the sentence and not
the privileged one.

**The evidence gate is the backlog's, not a new one.** A widened definition licenses
speculative hardening and optimisation with no number unless something says otherwise, and
`docs/features/BACKLOG.md` already asks for a value somebody would notice and a trigger that
can be observed. Rule 2 asks the same two questions at draft time, in the same words. The
naming test already handles the other abuse: "extract the audit into modules" is not a
sentence an operator recognises, and no widening makes it one.

**Prose, because no refusal can make this judgement.** The protocol's own rule is to ask
what would make a rule unnecessary; here the answer is nothing the kit can build, so this is
the case prose is honest for. The cost is about five lines in the section every project
reads at every planning session, which is the reason the four shapes are one sentence rather
than a list with an example each.

**`AGENTS.md` moves in the same commit as the template.** `protocol-stale` compares this
project's section against the binary's copy, so a template edit alone reports drift here;
`kit init .` writes the section. `docs/TECH.md` carries the definition as one line and is
the third home for it.

## Tasks

- [ ] `templates/AGENTS-section.md` sends a planner to `## Constraints` and `## What it is
      not`, defines a feature as a change in what the project can do, can be trusted to do
      or can be trusted not to do, and gates it on the backlog's two questions — with this
      project's `AGENTS.md` regenerated in the same commit
- [ ] `templates/VISION.md` names what stops breaking among the observables `## What done
      looks like` asks for
- [ ] `docs/TECH.md`'s definition line says what the protocol now says

## Open questions

- **Does the protocol state how a shipped capability is removed?** `kit drop` refuses a
  Shipped doc because it is frozen history, so a removal is a new doc citing the old one —
  which is what 0032 and 0048 did, and what nothing says. Leaning: one clause in rule 2,
  since promoting subtraction to a named shape is what raises the question; a reader who
  cannot find the answer will edit a frozen doc instead.
