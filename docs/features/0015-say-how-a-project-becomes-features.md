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
The same project, bootstrapped twice, produced one feature and then seven. The
designs were the same; the only difference was the word "decompose" in the prompt.
That is the prompt deciding the shape of the plan, which is exactly what the kit is
supposed to remove.

The protocol cannot break the tie. Rule 1 says *"size the doc to one agent's
reach"*, and both answers satisfy it: the single feature was carried by one agent in
twenty minutes, and each of the seven fits one too. Nothing anywhere says how many
features a project becomes, or where a boundary belongs.

The tool actively argues for one. `kit init` ends with:

```
next:
  kit new "<what the feature does>"
  fill in Problem, Non-goals, Design, Tasks; set status: Accepted
```

Singular, and the first run did precisely that. The same two lines also tell an agent
to set `status: Accepted`, which rule 1 reserves for a human — the contradiction 0014
removed from `kit next` and left in `init` and `new`, because the test it added reads
the protocol text and not what the commands print.

## Non-goals
- **Auditing decomposition quality.** Still the judgement call 0005 refused, and a
  check that guessed would fire on good plans. Everything here is guidance given
  before the work, not a finding after it.
- **A feature count.** Any number would be arbitrary and wrong for the next project.
  The criteria are about each feature, and the count falls out.
- **Deciding for the agent.** `kit new` writes a doc; it does not read the vision and
  emit a plan. The agent is better at that than any heuristic, and 0007 settled it.
- **Changing what a box is.** A box is one commit and that is already right. This is
  the level above it.

## Design
**Bootstrapping means producing the whole set.** The default is the full decomposition
of what the vision describes, left `Draft` for a human. Waiting to be asked is what
made the shape depend on the prompt, so the protocol says it plainly and `kit init`
stops suggesting a single feature.

**A feature is one pull request.** That is the criterion that discriminates, and it is
the one the kit had never written down. The agent pushes a branch and a human opens
and reads the PR, so the feature is the unit of review. Too big is a PR nobody can
review in one sitting — the first run's single feature was config, domain, storage,
HTTP and the CLI in one. It is a real bound, tied to the actual bottleneck rather than
to taste.

**Too small has a bound too**: a feature that changes nothing anyone can observe, and
exists only so the next one can begin, is a task box wearing a doc.

**The boundaries come from the vision.** `## What done looks like` is already a list of
observable outcomes, so each one is at least one feature. That gives the agent a
source for the split rather than an instruction to be sensible, and it makes the
vision earn its place twice over.

**And the commands stop contradicting the protocol.** `init` and `new` say the draft
waits for a human, like `next` does. The test that pins this moves from reading the
protocol to reading what every command prints, since the protocol was already right in
both cases and the output was not.

## Tasks
- [x] Protocol: decompose the whole thing by default, from the vision's outcomes
- [x] Protocol: a feature is one PR, with the bound at each end
- [x] `kit init` stops asking for one feature, and neither it nor `kit new` tells an agent to accept
- [x] A test that no command output tells an agent to set a status a human owns

## Open questions
<!-- none -->

## Log
- 2026-08-14 — the tie-breaker turned out to be the review unit rather than anything
  about size or independence. "One agent's reach" is satisfied by a single feature
  covering a whole project, which is how the first run passed the rule and still
  produced a plan nobody could review.
- 2026-08-14 — 0014's test read the protocol and not the commands, so the same
  contradiction survived in two places it had just been removed from. The test here
  reads output, which is where an agent actually meets the instruction.
