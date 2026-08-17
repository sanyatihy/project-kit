---
id: 0031
title: Cut the kit's own docs to what gets read
status: In progress
created: 2026-08-17
shipped:
check:
checked:
needs:
related: 0022-say-how-much-to-write-and-hold-the-kit-s-own-docs-to-it.md
---

## Problem
`docs/TECH.md` is 546 lines and 6,325 words. `AGENTS.md` is 237. Both are read at the
start of every session — the protocol in every project, TECH before any change to the
kit — so their length is a tax paid per session, and past a point it is paid for
nothing: a long file is skimmed, and a skimmed rule is one an agent gets wrong.

The cost is time, which is the only thing the kit is for. `docs/VISION.md` now says the
kit is measured by speed and transparency, and a document nobody finishes reading is the
clearest case of a cost with no return.

It grew the way 0022 predicted, one justified paragraph at a time, and 0022's ceilings
only cover text the binary *ships*. TECH has no ceiling at all, and it shows:

- **The same fact is stated more than once.** "A claim is a branch name" appears twice.
  Shipping-as-handoff appears three times, under three headings.
- **Two sections answer the same question.** `## The model` (278 lines) and
  `## Decisions and why` (131) are both "what we chose and why"; together they are 75%
  of the file.
- **Incident narratives run to paragraphs** where a clause would carry the same weight.

The protocol has the opposite problem: it is not repetitive, it is *discursive*. Rule 13
is 11 lines. An agent skims it and follows the first sentence.

## Non-goals
- **Cutting decisions, constraints, or the fact that evidence exists.** 0022's rule
  stands: a doc too thin to act on costs a session to reconstruct. What goes is the
  retelling — the decision keeps a clause of evidence, not a paragraph.
- **Rewriting README.md.** It is the front door for a reader who chose to be there, not
  a per-session tax. Out of scope, and it is 426 lines.
- **Splitting TECH into several files.** Two files nobody reads is not an improvement.
- **Changing any behaviour.** Nothing in `internal/` or `cmd/` changes except the
  ceilings that hold this.

## Design
One editing pass per file, then a ceiling that stops the regrowth — the second is what
makes this a feature rather than a tidy-up, because 0022 already proved the first does
not hold on its own.

**`docs/TECH.md` → 200 lines or fewer**, from 546. `## The model` and `## Decisions and
why` merge: one list of what is true and why, deduplicated, each entry a sentence of
decision and at most a clause of evidence. `## What it is`, `## Gotchas`, `## Verifying`
and `## State of play` stay as sections and get the same treatment.

**`templates/AGENTS-section.md` → 110 lines or fewer**, from 153. Same thirteen rules
and the same two closing sections — nothing is dropped — but each rule states its
constraint and its consequence and stops. Where a rule needs a reason to be obeyed
rather than merely understood, the reason stays; where it is background, it moves to
TECH or goes. Rule 13 is the worst offender at 11 lines and the test of whether this
works.

**Ceilings for both, in `cmd/kit/budget_test.go`.** It already holds shipped text to a
number; this extends it to the two docs this repo reads every session, with the same
failure message and the same escape hatch — raise it in the test and say why.

The measure of success is not the number. It is that an agent reading the protocol once,
quickly, follows it correctly — so each rule has to survive being read in isolation and
in a hurry. Every line kept must answer "what does this save the reader"; the ones that
only record that we thought about something go.

## Tasks
- [ ] `docs/TECH.md` merged and cut to ≤200 lines, with no fact stated twice
- [ ] `templates/AGENTS-section.md` cut to ≤110 lines, every rule kept, and the repo's
      own `AGENTS.md` regenerated
- [ ] Ceilings for both in `cmd/kit/budget_test.go`, with the reasoning for the numbers
- [ ] `kit help as-built` and README describe what TECH is for, in one place rather than
      three
- [ ] The template and protocol shift weight from writing before the work to writing
      during it: `## Design` is decisions a builder cannot derive, `## Log` carries what
      was learned

## Log
- 2026-08-17 — asked for directly: 2–3x shorter TECH, a protocol that is bulletproof
  rather than long.
- 2026-08-17 — the fifth box was described in conversation as already added and was not
  written down. Added here with the standard it comes from: what was written before the
  work is the part that turned out wrong (0027's design kept a check it did not need,
  0028's never said which copy of the doc to read), and both were caught in a `## Log`
  line during. `docs/VISION.md` gains the balance this rests on — the correction to it
  rides in this branch because it is the standard this feature applies.
