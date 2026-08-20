---
id: 0045
title: Write for the smallest agent that reads it
status: Draft
created: 2026-08-20
shipped:
check:
checked:
review:
needs:
related: 0041-write-about-the-system-not-the-session.md, 0044-route-each-fact-to-one-home.md, 0022-say-how-much-to-write-and-hold-the-kit-s-own-docs-to-it.md
---

## Problem
How much to write is stated in seven places and in none of them is it the protocol, the
one file every session reads. `README.md`, `kit help conventions`, the `VISION` and `TECH`
template comments, `kit ship`, `kit help ship` and the budget test's failure message each
carry their own wording of it. The protocol itself has only grown: 115 lines to 121 to 123.

The words are in the clauses rather than in the sections. Measured over 953 sentences in
the 44 docs here and 630 in `localcode`'s 16: mean 20.0 and 22.9 words, p99 47 and 57, and
roughly one sentence in three carries an em-dash or a semicolon. Every length instrument
the kit already has is slack against that — `doc-bloat` fires on nothing here, at a median
of 68 lines against a limit of 150.

A small local model pays this first and worst. It reads 496 lines of `AGENTS.md` and
`docs/TECH.md` before it opens a feature doc, and then it imitates the style it just read.

## Non-goals
- **A tighter word or line budget on a document.** It is the gameable instrument: the
  cheapest way to clear it is to delete a decision. `DocLineLimit` stays where it is.
- **A filler-phrase marker set.** Measured over both corpora: 2 of 44 docs here and 1 of
  16 there contain one. The stock phrases are not what makes these docs long.
- **A restatement check inside one doc.** Shingle overlap between `## Problem` and
  `## Design` peaks at 6.2% across the 44 docs, so the restatement 0037 named is not
  literal and cannot be matched.
- **The `## Problem` ≥ `## Design` composition check.** Still the sharper instrument on
  what a doc spends its lines on, still needs its own calibration. Stays in BACKLOG.
- **A short-form protocol for small models.** That is one fact with two homes, and the
  short one would be the one nobody updates.
- **Rewriting shipped docs.** Rule 11 freezes them. This changes what gets written next.

## Design
**One rule, in the protocol: a sentence earns its place by deciding something, constraining
something, or measuring something.** Everything else goes. It is a test applied per
sentence with no taste required, which is what a model with few parameters can actually do.
"As much as necessary and no more" is not that test — necessity is a judgement, and the
model making it is the one that wrote the sentence.

**One claim per sentence, and evidence gets one trailing clause and never a second.** 0044
already grants the first clause where it stops a rule being reversed. The second is where
the words are: p99 is 47 words here and 57 in the reference corpus, and a sentence that
needs three clauses is two sentences with the smaller one hidden.

**The rule replaces its seven homes rather than joining them, and the protocol ceiling does
not rise.** Each of the others keeps one line that cites the rule instead of restating it.
A raise is the signal that the rule was written as an addition, so `protocolBudget` holding
at 123 or falling is the check on this feature rather than a matter of taste.

**The backstop is per sentence, not per document, and that is the whole answer to gaming.**
A document ceiling is cleared most cheaply by cutting a paragraph, so it teaches deletion.
A sentence ceiling is cleared most cheaply by splitting the sentence, which deletes nothing
and leaves every decision, constraint and measurement in place. `long-sentence` at LOW,
with the threshold taken from `localcode`'s p99 rather than from this corpus, on 0041's
reasoning: this repo has been trimmed by hand three times and cannot calibrate itself.

**The template carries one rejected sentence beside its rewrite.** A small model imitates
the corpus it just read more reliably than it follows an instruction about it, and this is
how every convention here that stuck was taught. It costs no protocol lines and no session
context, because a template is read when a doc is created rather than every session.

## Tasks
- [x] The rule in `templates/AGENTS-section.md`, stated once, with the per-sentence test and
      the one-clause limit; `protocolBudget` holds or falls, and the test's comment says
      which rules it replaced
- [x] `README.md`, `templates/features-README.md`, `templates/VISION.md`,
      `templates/TECH.md` and the budget test's failure message cite the rule instead of
      stating a variant, and `features-README.md` stops claiming every section states its
      own budget
- [x] A rejected sentence beside its rewrite in `templates/features-TEMPLATE.md`, under
      `## Design`, showing a three-clause sentence cut to two claims
- [ ] `long-sentence` at LOW over feature docs, `docs/VISION.md` and `docs/TECH.md`, with
      the threshold, its sample and its date in the constant's comment; conformance that it
      fires and that a doc of short sentences stays quiet
- [ ] Fenced blocks, tables, HTML comments and `## Tasks` boxes are excluded from the scan,
      for `Narration`'s reason, and a wrapped sentence is measured unwrapped
- [ ] README and `kit help checks` carry the new code, and this doc is measured against the
      rule it adds

## Open questions
- Which corpus sets the threshold. `localcode`'s p99 of 57 fires on 5 of its 16 docs and 2
  of the 44 here, which is `secret-in-doc`'s shape and may be too quiet to change what gets
  written. Leaning: take it anyway, and let the rule carry the work.

## Log
- 2026-08-20 — the box grew the budget test's failure message. It taught "write as much as
  necessary and no more", which box 1 replaced, so shipping without it would have put the
  rule and its superseded wording in the same binary.
