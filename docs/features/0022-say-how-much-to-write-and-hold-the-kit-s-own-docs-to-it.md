---
id: 0022
title: Say how much to write, and hold the kit's own docs to it
status: Shipped
created: 2026-08-15
shipped: 2026-08-16
check:
checked:
needs: 0021
related: 0003-ship-standing-engineering-guidance-in-the-binary.md
---

## Problem
Nothing states how long a doc should be, so both failures are live. Two drafts ran to
145 lines each where the template asks `## Problem` for 1–3 sentences;
`templates/AGENTS-section.md` is 142 lines, written into every project and re-read at the
start of every session, so its length is a tax paid per session per project. From the
other side, `unfilled-template` catches an empty section and nothing catches a thin one —
a `## Design` of one vague line passes every check and costs the next agent a session to
reconstruct.

The guidance that exists is scattered and one-sided: the VISION template says *"keep it
short"*, the TECH template asks for *"two or three sentences"*, and the feature template
gives a budget only for `## Problem`.

## Non-goals
- **An `audit` check on document length.** Necessity is not checkable and a line count
  fires on healthy docs, which the vision names as worse than no check. Under-writing is
  already covered where it is objective: `unfilled-template` for an empty section,
  `missing-section` for an empty `## Log`.
- **A house style guide.** This is one rule about length, not prose taste.
- **Rewriting every shipped doc.** They are frozen history.

## Design
**One rule, stated once, in the protocol:** *write as much as necessary and no more.*
Padding, restatement and hedging go; a decision, a constraint, or the evidence behind it
never does. Terseness is not a virtue on its own — a doc too thin to act on costs more
than a long one.

**Per-section budgets in every template**, two-sided and consistent across
`features-TEMPLATE.md`, `VISION.md` and `TECH.md`. Several exist already; they are
one-sided and only in some files.

**The kit holds its own text to it with a Go test**, following `help_test.go`'s existing
precedent of policing embedded prose — `TestNoTopicLooksLikeATaskList`,
`TestProtocolCitesRulesByNameNotNumber`. A budget on `AGENTS-section.md` and each help
topic means the protocol cannot grow unnoticed, which is the only place where growth
compounds across every project.

The number is a ceiling on kit-owned text, not a target for anyone's docs, and the failure
message says so — otherwise the test teaches the wrong half of the rule.

**`AGENTS-section.md` is cut to fit.** 0021 already deletes the per-feature-branch rule and
0014's delete-the-branch rule, so most of the reduction is work that feature is doing
anyway.

## Tasks
- [x] The rule in `templates/AGENTS-section.md`, stated once, both halves
- [x] Two-sided per-section budgets in `features-TEMPLATE.md`, `VISION.md`, `TECH.md`
- [x] A test budgeting `AGENTS-section.md` and every help topic, failing with a message
      that names the rule rather than only the number
- [x] Cut `AGENTS-section.md` under its budget, coordinating with 0021's rewrite
- [x] `kit help` and README state the rule where an agent will meet it

## Log
- 2026-08-15 — scoped from two drafts that ran 145 lines each against a template asking
  for three sentences.
