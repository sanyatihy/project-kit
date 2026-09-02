---
id: 0045
title: Write for the smallest agent that reads it
status: Shipped
created: 2026-08-20
shipped: 2026-08-20
needs:
related: 0041-write-about-the-system-not-the-session.md, 0044-route-each-fact-to-one-home.md, 0022-say-how-much-to-write-and-hold-the-kit-s-own-docs-to-it.md
---

## Problem

How much to write is stated in seven places, and in none of them is it the protocol, the one
file every session reads. `README.md`, `kit help conventions`, the `VISION` and `TECH`
template comments, `kit ship`, `kit help ship` and the budget test's failure message each
carry their own wording. The protocol itself has only grown: 115 lines to 121 to 123.

The words are in the clauses rather than in the sections. Measured over 953 sentences in the
44 docs here and 630 in `localcode`'s 16: mean 20.0 and 22.9 words, p99 47 and 57, and
roughly one sentence in three carries an em-dash or a semicolon. Every length instrument the
kit has is slack against that — `doc-bloat` fires on nothing here, at a median of 68 lines
against a limit of 150.

A small local model pays this first and worst. It reads 496 lines of `AGENTS.md` and
`docs/TECH.md` before it opens a feature doc, and then imitates the style it just read.

## Non-goals

- A tighter word or line budget on a document. It is the gameable instrument: the cheapest
  way to clear it is to delete a decision.
- A filler-phrase marker set. 2 of 44 docs here and 1 of 16 there contain one, so stock
  phrases are not what makes these docs long.
- A restatement check inside one doc. Shingle overlap between `## Problem` and `## Design`
  peaks at 6.2%, so the restatement 0037 named is not literal and cannot be matched.
- The `## Problem` ≥ `## Design` composition check. Still the sharper instrument, still needs
  its own calibration.
- A short-form protocol for small models. One fact with two homes, and the short one would
  be the one nobody updates.
- Rewriting shipped docs. Rule 11 freezes them.

## Design

One rule, in the protocol: a sentence earns its place by deciding something, constraining
something, or measuring something. It is a test applied per sentence with no taste required,
which is what a model with few parameters can do. "As much as necessary and no more" is not
that test — necessity is a judgement, and the model making it is the one that wrote the
sentence.

One claim per sentence, and evidence gets one trailing clause and never a second. 0044
already grants the first clause where it stops a rule being reversed. The second is where
the words are: p99 is 47 words here and 57 in the reference corpus, and a sentence needing
three clauses is two sentences with the smaller one hidden.

The rule replaces its seven homes rather than joining them, and the protocol ceiling does
not rise. Each of the others keeps one line citing the rule instead of restating it. A raise
would be the signal that the rule was written as an addition, so `protocolBudget` holding at
123 or falling is the check on this feature.

The backstop is per sentence, not per document, which is the whole answer to gaming. A
document ceiling is cleared most cheaply by cutting a paragraph, so it teaches deletion; a
sentence ceiling is cleared most cheaply by splitting the sentence, which deletes nothing.
`long-sentence` at LOW, over the docs rule 11 still leaves editable, at a threshold chosen
for the volume it produces rather than taken from either corpus's tail.

The template carries one rejected sentence beside its rewrite. A small model imitates the
corpus it just read more reliably than it follows an instruction about it, and a template is
read when a doc is created rather than every session.

## Tasks

- [x] The rule in `templates/AGENTS-section.md`, stated once, with the per-sentence test and
      the one-clause limit; `protocolBudget` holds or falls
- [x] `README.md`, `templates/features-README.md`, `templates/VISION.md`,
      `templates/TECH.md` and the budget test's failure message cite the rule instead of
      stating a variant
- [x] A rejected sentence beside its rewrite in `templates/features-TEMPLATE.md`
- [x] `long-sentence` at LOW over feature docs, `docs/VISION.md` and `docs/TECH.md`, with the
      threshold, its sample and its date in the constant's comment
- [x] Fenced blocks, tables, HTML comments and `## Tasks` boxes are excluded from the scan,
      and a wrapped sentence is measured unwrapped
- [x] README and `kit help checks` carry the new code

## Log

- 2026-08-20 — the box grew the budget test's failure message. It taught "write as much as
  necessary and no more", which box 1 replaced, so shipping without it would have put the
  rule and its superseded wording in the same binary.
- 2026-08-20 — the threshold is 45 words, closing the open question against its own leaning.
  `localcode`'s p99 of 57 fires once here and cannot change what gets written; 45 leaves 5
  sentences across 2 docs here and 11 across 3 there.
- 2026-08-20 — the scan skips every terminal status, not only Shipped. A Dropped doc in
  `localcode` produced a finding nobody would ever fix.
