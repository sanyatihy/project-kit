---
id: NNNN
title: <observable improvement>
status: Draft
created: YYYY-MM-DD
submitted:
needs:
---

<!-- A feature is one coherent, reviewable improvement. Addition, removal, security,
     reliability, performance and cost reductions all qualify. Prefer a substantial
     outcome with implementation steps as tasks. Split for independent value, distinct
     approval decisions or reviewability, not by code layer or agent count.
     Name the benefit and an observable result before choosing implementation tasks. -->

<!-- List prerequisite ids in `needs:` only when this work cannot start until their
     implementation is merged. Leave it empty for independent work.
     Depend on an API, schema or decision you must build on. Shared files alone do not
     create a dependency; incompatible decisions or a shared number allocation can.
     `status: Submitted` on a working branch records a submission, not integration. -->


## Problem
<!-- 1-3 sentences: the standing tension and the signal that justifies it now — not the
     incident that surfaced it, which the pull request holds. Shorter and nobody can tell
     whether it was worth doing; longer and the doc is arguing instead of deciding. -->

## Non-goals
<!-- What this deliberately does not do, and where that lives instead. One line each, with
     the reason on it: "A database. None is needed at four hundred rows a month." Not an
     argument, and never a thing nobody would have built. -->

## Design
<!-- Decisions only: what was chosen, what it beat, what it costs. Not a walkthrough — the
     code will say what the code does, and this is read alongside it. Call out data-model,
     migration, API and config changes explicitly. Written before the work, so it is the
     part most likely to be wrong by the end; let `## Log` carry what the doing teaches.

     One claim per sentence, and evidence gets one clause. Rejected, then the rewrite:
       no   the cache is per-process because a shared one would need invalidation across
            workers, which this deployment does not have, and it also drops the dependency
            we have been trying to get rid of since the last migration
       yes  The cache is per-process. A shared one needs cross-worker invalidation, which
            this deployment does not have.
     The third clause was about the work rather than the system, so it is gone rather
     than split. Write the decision and not the route to it: "Changed JSONL for SQLite",
     never "we researched the append-only design, found it would not scale, and chose a
     database".

     A paragraph per `## Tasks` box is the Tasks list restated, and the same fact then
     sits in two sections. What a box cannot say is what belongs here.

     A doc is as long as its decisions and no longer. One a reader cannot hold in one
     sitting is describing a feature that is too big, not a design worth the length. -->

## Tasks
<!-- Each box is one commit with an outcome you can verify, in execution order.
     The feature is one pull request containing those commits:
       - [ ] the outcome, not the activity

     Guidance lives in comments here on purpose. A box written inside one is invisible
     to the tool, so an unwritten plan counts as unwritten: `unfilled-template` says the
     plan is missing instead of `kit next` handing an agent a placeholder to build. -->

## Open questions
<!-- Optional. Delete when empty. Each with a leaning if you have one. -->

## Log
<!-- One entry per change to the plan, and each names the change: a box rewritten, a
     premise falsified, an open question settled, a number something downstream reads.
     Take the room a change needs; append-only.

     An entry naming no change is not an entry. What the work did before it converged —
     the run, the re-run, the reading that was revised — belongs in the pull request, and
     a fact still true after this ships belongs in the as-built doc, one line. Never both:
     each fact has one home.
       - YYYY-MM-DD — decided X over Y because … -->

