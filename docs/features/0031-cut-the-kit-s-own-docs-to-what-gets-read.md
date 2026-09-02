---
id: 0031
title: Cut the kit's own docs to what gets read
status: Shipped
created: 2026-08-17
shipped: 2026-08-17
needs:
related: 0022-say-how-much-to-write-and-hold-the-kit-s-own-docs-to-it.md
---

## Problem

`docs/TECH.md` is 546 lines and 6,325 words; `AGENTS.md` is 237. Both are read at the start
of every session — the protocol in every project, TECH before any change to the kit — so
their length is a tax paid per session, and past a point it is paid for nothing: a long file
is skimmed, and a skimmed rule is one an agent gets wrong.

It grew the way 0022 predicted, one justified paragraph at a time, and 0022's ceilings only
cover text the binary ships. TECH has none, and it shows: "a claim is a branch name" appears
twice and shipping-as-handoff three times under three headings; `## The model` (278 lines)
and `## Decisions and why` (131) both answer "what we chose and why" and are 75% of the
file; incident narratives run to paragraphs where a clause would carry the same weight.

The protocol has the opposite problem. It is not repetitive but discursive — rule 13 is 11
lines, and an agent skims it and follows the first sentence.

## Non-goals

- Cutting decisions, constraints, or the fact that evidence exists. A doc too thin to act on
  costs a session to reconstruct; what goes is the retelling.
- Rewriting README.md. It is the front door for a reader who chose to be there, not a
  per-session tax.
- Splitting TECH into several files. Two files nobody reads is not an improvement.
- Changing any behaviour. Nothing in `internal/` or `cmd/` changes except the ceilings.

## Design

One editing pass per file, then a ceiling that stops the regrowth. The second is what makes
this a feature rather than a tidy-up, because 0022 already proved the first does not hold on
its own.

`docs/TECH.md` goes to 250 lines or fewer, from 546. `## The model` and `## Decisions and
why` merge into one deduplicated list, each entry a sentence of decision and at most a
clause of evidence. The remaining sections get the same treatment.

`templates/AGENTS-section.md` goes to 110 lines or fewer, from 153. Same thirteen rules and
the same two closing sections — nothing is dropped — but each rule states its constraint and
its consequence and stops. Where a rule needs a reason to be obeyed rather than merely
understood, the reason stays; where it is background, it moves to TECH.

Ceilings for both in `cmd/kit/budget_test.go`, which already holds shipped text to a number,
with the same failure message and the same escape hatch: raise it in the test and say why.

The measure of success is not the number but that an agent reading the protocol once,
quickly, follows it correctly. Every line kept must answer "what does this save the reader".

## Tasks

- [x] `docs/TECH.md` merged and cut to ≤250 lines, with no fact stated twice
- [x] `templates/AGENTS-section.md` cut to ≤115 lines, every rule kept, and the repo's own
      `AGENTS.md` regenerated
- [x] Ceilings for both in `cmd/kit/budget_test.go`, with the reasoning for the numbers
- [x] `kit help as-built` and README describe what TECH is for, in one place rather than three
- [x] The template and protocol shift weight from writing before the work to writing during
      it: `## Design` is decisions a builder cannot derive, `## Log` carries what was learned

## Log

- 2026-08-17 — `docs/VISION.md` gains the standard this applies: speed and transparency
  together, the record a byproduct of the work rather than a stage before it.
- 2026-08-17 — TECH lands at 247 lines from 546 and the protocol at 114 from 153, with all
  thirteen rules kept. The reasoning moved to TECH, read once rather than every session.
- 2026-08-17 — `TestHelpAnswersWhatAnOutsideReviewGotWrong` caught the rewrite dropping the
  literal "One feature per checkout." while keeping the meaning. That test pins the phrases
  an outside reader once got wrong, and rephrasing one is how the guarantee is lost.
