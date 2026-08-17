---
id: 0015
title: Say how a project becomes features
status: Shipped
shipped: 2026-08-14
created: 2026-08-14
needs:
related: 0007-a-living-vision-agents-decompose-from.md
---

## Problem
The same project, bootstrapped twice, produced one feature and then seven. The designs were
the same; the difference was the word "decompose" in the prompt — the prompt deciding the
shape of the plan, which is what the kit exists to remove. Rule 1's *"size the doc to one
agent's reach"* cannot break the tie: both answers satisfy it.

`kit init` argued for one, printing `kit new "<what the feature does>"` singular, and told
the agent to set `status: Accepted` — the contradiction 0014 removed from `next` and left
here, because its test read the protocol text rather than what commands print.

## Non-goals
- **Auditing decomposition quality.** A check that guessed would fire on good plans.
- **A feature count.** Any number would be arbitrary; the criteria are per feature and the count falls out.
- **Deciding for the agent.** `kit new` writes a doc; it does not read the vision and emit a plan.

## Design
**Bootstrapping means producing the whole set**, left `Draft` for a human. Waiting to be
asked is what made the shape depend on the prompt.

**A feature is one pull request.** That is the criterion that discriminates and the one the
kit had never written down: the agent pushes a branch and a human reads the PR, so the
feature is the unit of review. Too big is a PR nobody can review in one sitting — the first
run's single feature was config, domain, storage, HTTP and CLI together. Too small changes
nothing observable and exists only so the next can begin: that is a task box wearing a doc.

**The boundaries come from the vision's `## What done looks like`**, already a list of
observable outcomes, so each is at least one feature.

**The commands stop contradicting the protocol**, and the test that pins it reads command
output rather than the protocol text — which is where an agent meets the instruction.

## Tasks
- [x] Protocol: decompose the whole thing by default, from the vision's outcomes
- [x] Protocol: a feature is one PR, with the bound at each end
- [x] `kit init` stops asking for one feature, and neither it nor `kit new` tells an agent to accept
- [x] A test that no command output tells an agent to set a status a human owns

## Log
- 2026-08-14 — the tie-breaker turned out to be the review unit, not size or independence.
  "One agent's reach" is satisfied by a single feature covering a whole project.
