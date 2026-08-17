---
id: NNNN
title: <short imperative name>
status: Draft        # Draft | Accepted | Shipped | Dropped | Superseded
created: YYYY-MM-DD
shipped:             # fill the date when status flips to Shipped
check:               # optional — date to check whether this worked. Only for bets.
checked:             # date you checked; write the outcome in ## Log
review:              # optional — `human` means a person merges this one. kit accept --review
needs:               # feature ids that must ship first, e.g. 0002, 0003. Empty = can start now
related:             # other docs worth reading first
---

<!-- `needs:` is the only field that changes what `kit next` offers, so it earns care.
     Leave it empty if this could be built today against what already exists — that is
     the common case, and several empty ones is what lets agents work in parallel.

     Judge it by whether the work could START now, not by what it touches. Clean
     boundaries hide couplings that make a feature dependent anyway:
       - a shared composition root or wiring function both features must edit
       - a shared router, registry, or dispatch table both add an entry to
       - numbered files in one sequence — migrations above all. Two features each
         adding "the next number" merge cleanly and break at runtime.
     Any of those means the second feature needs the first, however separate they look.

     `related:` is different and does not affect ordering: it is reading material. -->


## Problem
<!-- 1-3 sentences: what is broken or missing, and the signal that justifies it now.
     Shorter than that and nobody can tell whether it was worth doing. -->

## Non-goals
<!-- What this deliberately does not do, and where that lives instead. One line each;
     a reason with each, or the next person reopens the argument. -->

## Design
<!-- Decisions only: what was chosen, what it beat, what it costs. Not a walkthrough — the
     code will say what the code does, and this is read alongside it. Call out data-model,
     migration, API and config changes explicitly. Written before the work, so it is the
     part most likely to be wrong by the end; let `## Log` carry what the doing teaches. -->

## Tasks
<!-- Each box is one PR-sized outcome you can verify without asking anyone, in the
     order they should be done:
       - [ ] the outcome, not the activity

     Guidance lives in comments here on purpose. A box written inside one is invisible
     to the tool, so an unwritten plan counts as unwritten: `no-tasks` says the plan is
     missing instead of `kit next` handing an agent a placeholder to build. -->

## Open questions
<!-- Optional. Delete when empty. Each with a leaning if you have one. -->

## Log
<!-- One line per entry, and only when the plan changed: a box rewritten, a premise
     falsified, an open question settled, a measurement that changes what to do next.
     Not a diary — nothing about mistakes made and fixed, rules complied with, or what
     the diff already shows. Append-only.
       - YYYY-MM-DD — decided X over Y because … -->

