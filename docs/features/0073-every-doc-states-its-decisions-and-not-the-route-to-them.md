---
id: 0073
title: Every doc states its decisions and not the route to them
status: Shipped
created: 2026-09-01
shipped: 2026-09-02
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
- Cutting the comments already written. 1,969 lines is its own feature, and the rule changes
  what gets written next.

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

Code comments are the same defect in the third artifact. 27% of non-test source is comment —
1,969 lines of 7,239 — and 105 blocks of six lines or more hold 927 of them. `OnlyHere`
spends 20: the contract and two constraints earn their place, and seven restate 0039's
`## Problem` verbatim. A comment says what the code cannot and stops — the contract, and the
constraint the next editor would otherwise break. Not what the code already says, and not
the incident that produced it, which the feature doc holds. It goes in `### How to write`
beside the sentence rule, because a comment is prose and the language checklists are
per-language.

The commit message is the same defect in the same corpus. 151 of the last 200 commits carry
a body — 1,101 lines, median 7, longest 21 — restating the diff a paragraph per file, and
the protocol never named the message: rule 12 requires the commit to name its box honestly
and stops. The subject names the box and the body stays empty, because the diff shows what
changed and the doc says why. A body earns its place only where the diff cannot show what it
says — a revert and its cause, the source a commit was cherry-picked from — which are facts
about the commit rather than about the change in it. It goes in rule 12 rather than
`### How to write`, because a commit is written at the moment a box is ticked.

## Tasks

- [x] `### How to write` and `templates/features-TEMPLATE.md` carry the three rules, with
      this project's `AGENTS.md` regenerated in the same commit
- [x] The five docs over 100 lines state decisions only
- [x] Every doc of 60 lines or more states decisions only
- [x] Every doc under 60 lines loses its bold lead-ins and its route-narration
- [x] `docs/TECH.md` records the one-time compaction and what it may not cut
- [x] Rule 12 says the subject names the box and the body stays empty unless the diff cannot
      show what it says, with this project's `AGENTS.md` regenerated in the same commit
- [x] `### How to write` says a comment carries the contract and the constraint the next
      editor would break, and not the incident that produced it

## Log

- 2026-09-01 — the bloat is not uniform, so the pass is not either. Docs 0060–0072 averaged
  841 words against 0034–0059's 632, the latter written under 0041, 0044 and 0045's rules
  and the former after attention moved elsewhere. The first group is rewritten; the second
  mostly needed its bold lead-ins removed, which is one scripted change across 47 files.
- 2026-09-01 — the commit message joins the round rather than taking an id of its own. It is
  the same defect measured in the same corpus, and a second doc would restate this one's
  design. 0074 was drafted for it and dropped before it landed.
- 2026-09-01 — the comment rule ships without a pass over the 1,969 lines already written.
  Docs are prose with no behaviour behind them; comments sit in code, and a corpus that size
  is its own feature. It is a BACKLOG entry with the measurement.
