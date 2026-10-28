---
id: 0086
title: Agent-drafted docs land at decision density
status: Draft
created: 2026-10-28
submitted:
needs:
---

## Problem

Under v0.12.0, an agent drafted eight features at 5,266 words. The reviewer trimmed them
to 3,191 (−39%) and lost no decision, file or task. The excess came from four sources:
a paragraph arguing each decision, measurements in Problem, narration of mechanics, and
prose where bullets fit. The template's only Design example is a sentence, and nothing
says where evidence goes, so drafts copy the sentence shape and keep the evidence.

## Non-goals

- A word-count finding. It fires on a doc whose decisions need the space. 0073's ruling stands.
- Flagging Submitted docs. They are frozen, and a finding nobody can fix is noise (0060).
- Rewriting existing docs. This changes what gets drafted next.

## Design

- **Design shape.** The template gives a bold-labelled bullet example: one decision per
  bullet, with a `Rejected:` clause for each option it beat. This reverses 0073's
  removal of bold lead-ins for Design only. A label names the decision so a reviewer can
  scan for it. Rejected: plain bullets, which are harder to scan across eight docs.
- **Budget.** The template says most docs fit in 200–600 words. It is guidance and no
  check reads it. Rejected: no stated size, which is 0073's position and produced the
  draft above.
- **Evidence.** One template line: measurements and evidence go in the PR, not the doc.
  Problem keeps only the signal, in a clause.
- **`doc-long` (LOW).** Fires on a Draft or Parked doc whose `## Problem` contains a
  table, or whose `## Design` has a paragraph of more than three sentences. Bullets are
  not paragraphs. A sentence ends at `.`, `?` or `!` followed by a space or newline, so
  code spans and file names do not count. Rejected: a word budget, which cannot tell a
  verbose doc from a large one.
- Files: `templates/features-TEMPLATE.md`, `internal/kit/audit.go`,
  `cmd/kit/conformance_audit_test.go`, `docs/TECH.md`.

## Tasks

- [ ] `templates/features-TEMPLATE.md` carries the bullet example, the 200–600 word
      budget and the evidence-goes-in-the-PR line
- [ ] `doc-long` is registered as LOW and fires on a Problem table or a Design paragraph
      over three sentences, in Draft and Parked docs only, with conformance cases for
      both triggers and for a Submitted doc that stays silent
- [ ] `kit audit` on this repo reports no `doc-long` finding for 0086 or 0087
- [ ] `docs/TECH.md` records the check and its sentence rule

## Log
