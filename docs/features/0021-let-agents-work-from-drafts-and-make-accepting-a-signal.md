---
id: 0021
title: Let agents work from drafts, and make accepting a signal
status: Shipped
created: 2026-08-15
shipped: 2026-08-16
check:
checked:
needs:
related: 0018-give-the-human-an-accept-command.md, 0014-reach-the-agent-not-just-the-reader.md
---

## Problem
The drafting rules were written for a per-feature review flow the operator does not run,
and they blind the board. In a real project that adopted the kit, five features were
drafted onto five branches exactly as documented, none merged, and:

```
$ kit next /…/observer
  nothing is planned yet. Read docs/VISION.md and draft a feature, or stop.
```

The protocol also contradicts itself three lines apart in `AGENTS-section.md`: a draft
*"waits on a branch of its own"* and *"one review and one merge per feature"*, but a
project just set up gets *"one pull request landing the scaffold and the whole set of
drafts"*.

Underneath both: `status: Accepted` gates every agent, while the facts that actually mean
*unsettled* — an open `## Open questions` section, a BLOCKING inbox line — are already
machine-readable and ignored by `claim`. And because `Branch()` derives the claim from the
filename, a draft on its own branch is a claim, which 0014 had to patch with a
delete-the-branch rule.

## Non-goals
- **Removing the human.** `kit accept` stays; it stops being a precondition for work and
  becomes a statement of confidence that orders the board. The implementation pull request
  is still reviewed.
- **Letting an agent past a question only the operator can answer.** A BLOCKING inbox line
  still refuses a claim. That is the gate that survives.
- **Auto-stamping `status:` on claim.** Real question, separate feature.
- **Reserving IDs over the network.** Batching makes `duplicate-id` likelier; the fix is
  already scoped in `BACKLOG.md` and is its own feature.
- **Any review workflow.** No approvals, no forge API. Unchanged from 0018.

## Design
**Drafting produces a plan round, not a branch per feature.** A session's drafts land on
one branch named `plan/<something>` — never a leading four-digit ID, which `Mentions` would
read as a claim. The operator iterates locally, accepts what they want to bless, merges
once. Docs reach `main` at whatever status they reached.

This deletes the name collision rather than patching it, so 0014's delete-the-branch rule
goes with it.

**`status:` orders the board instead of gating it**: In progress first, then Accepted
(blessed — work it hard), then Draft (invented, not blessed — expect to think).

**`kit claim` accepts a Draft, and refuses an empty one.** This is the load-bearing half.
`unfilled-template` exempts Draft only because Drafts cannot be claimed today; lift the
gate without closing that and `claim` hands an agent a doc straight out of `kit new`. So
`claim` reuses the emptiness test on `## Problem`, `## Design` and `## Tasks`.

Still refuses a claim: a BLOCKING inbox line, a duplicate ID, a terminal status, an
unwritten plan. No longer refuses: `status: Draft`.

**`## Open questions` means "settle these first", not "stop".** An agent claiming such a
draft settles them, or moves what only a human can answer to the inbox with `kit block`.
That is the refine-then-build loop, with no new machinery.

**`draft-claimed` is re-pointed** at building on a doc whose `## Open questions` are still
open — the defect it was reaching for, now that Draft-plus-work is the normal flow.

**`kit drop <id>`** retires a doc that will not be built, since drafts now live on `main`
permanently instead of being a branch you delete.

## Tasks
- [x] `kit claim` allows Draft and refuses an empty doc, with a conformance case each
- [x] `ActiveFeatures` ranks Draft below Accepted; `kit next` offers drafts as work
- [x] `draft-claimed` re-pointed at open questions, with fires and stays-quiet cases
- [x] `kit drop <id>`, refusing a Shipped doc, with `kit help` and README entries
- [x] Rewrite the drafting rules in `AGENTS-section.md` around the plan round, deleting the
      per-feature-branch rule and 0014's delete-the-branch rule
- [x] `kit help` and README describe status as priority, not permission

## Log
- 2026-08-15 — measured `kit next` against a project that adopted the kit: five drafts on
  five branches, board reports nothing planned.
- 2026-08-15 — shipped before ID reservation: batching collides five IDs where one collided
  before, but the cost is a rename and the fix deserves its own feature.
