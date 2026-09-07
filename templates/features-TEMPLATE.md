---
id: NNNN
title: <observable improvement>
status: Draft
created: YYYY-MM-DD
submitted:
needs:
---

<!-- One coherent, reviewable improvement per feature; implementation steps are tasks.
     Additions, removals, reliability, security, performance and cost gains qualify.
     Split for independent value, separate approval or safe review, not code layer or agent count.
     needs: names prerequisite features whose implementation must be integrated into
     trunk and this checkout. Shared files alone are not dependencies. Leave empty
     for independent work. Submitted on a work branch does not mean integrated. -->
<!-- Depend on an API, schema or decision you must build on. Incompatible decisions
     or shared number allocation can require ordering. Name the benefit and observable
     result before choosing tasks. -->

## Problem
<!-- Who benefits, the standing problem and the observable improvement. Keep it brief. -->

## Non-goals
<!-- Material exclusions, their reason and where they belong. Exclude plausible work:
     "A database. None is needed at four hundred rows a month." -->

## Design
<!-- Facts and decisions needed for approval: contracts, constraints, alternatives and
     accepted costs. Name data-model, migration, API and config changes. Omit implementation
     walkthroughs, repeated tasks and decision-making history. Keep it short. -->
<!-- One claim per sentence; evidence gets one clause. Example:
     "The cache is per-process. A shared one needs cross-worker invalidation, which
     this deployment does not have." Record plan corrections in Log. A design too
     large to review in one sitting calls for splitting the feature. -->

## Tasks
<!-- Ordered, verifiable outcomes. One box per implementing commit; one feature per PR.
     - [ ] <outcome>
     Example boxes stay inside comments so they do not count as executable work. -->

## Open questions
<!-- Optional; remove when empty. Give a leaning. Use kit block for human decisions. -->

## Log
<!-- Append changes to the plan, with their reason. May be empty. Durable facts go to
     TECH; validation and handoff go to the PR. Do not repeat the implementation history.
     - YYYY-MM-DD — changed <decision/task> because <reason> -->
