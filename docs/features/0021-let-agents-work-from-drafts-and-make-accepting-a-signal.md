---
id: 0021
title: Let agents work from drafts, and make accepting a signal
status: Draft
created: 2026-08-15
shipped:
check:
checked:
needs:
related: 0018-give-the-human-an-accept-command.md, 0014-reach-the-agent-not-just-the-reader.md
---

## Problem
The drafting half of the protocol was designed for a review flow the operator does not
run, and it has now failed twice in the field.

**The board goes blind on the repo that adopts the kit.** In a real project that took the
kit on, five features were drafted and each was pushed to a branch of its own, exactly as
written. None merged, because none was worth a pull request on its own. The result:

```
$ kit next /…/observer
  nothing is planned yet. Read docs/VISION.md and draft a feature, or stop.
```

Five features exist. The board reads `main`, the plan is not on `main`, and the tool whose
stated job is answering *what should I work on* in every repo answers it wrongly in the
repo most recently set up. Forty other branches make the five drafts invisible by eye too.

**The protocol contradicts itself, three lines apart.** `templates/AGENTS-section.md` says
a draft *"waits on a branch of its own"* and *"one review and one merge per feature"*, then
says that for a project just set up this means *"one pull request landing the scaffold and
the whole set of drafts"*. Batching is already the rule for a new project and per-feature
branches the rule for an existing one, and nothing says why. The agent in that repo
followed the second and was right to.

**The gate is in the wrong place.** `status: Accepted` blocks every agent until a human
transitions it, on the theory that nobody should build a design nobody settled. In
practice the operator brainstorms a handful of drafts, refines them across later agentic
sessions, and reviews the implementation pull request — which is a review either way. The
gate does not add a review; it adds a stop before one. Meanwhile the thing that genuinely
signals *unsettled* is not `status:` at all: it is an open `## Open questions` section, or
a BLOCKING line in the inbox, and both are already machine-readable.

**A branch named after a doc is a claim, so a draft on one is a claim.** `Branch()` derives
the branch from the filename, which 0006 made the claim. 0014 had to patch the fallout with
a rule that the draft branch is deleted on merge or the feature becomes permanently
unclaimable. That rule is a workaround for two meanings sharing one name.

## Non-goals
- **Removing the human from the loop.** `kit accept` stays, and so does the inbox. What
  changes is that acceptance stops being a *precondition for work* and becomes a
  *statement of confidence* that orders the board. The operator still reviews every
  implementation pull request, which is the review that was always going to happen.
- **Letting an agent past a question only the operator can answer.** A BLOCKING line in
  `docs/INBOX.md` still refuses a claim, unchanged. That is the gate that survives,
  because it is the one that marks work a human genuinely has to unblock.
- **Auto-managing `status:` on claim.** Whether claiming should stamp `In progress` is a
  real question and a separate one; nothing here changes who writes that field.
- **Reserving IDs over the network.** Batching makes `duplicate-id` more likely, not less
  — see `## Open questions`. The mechanism is already scoped in `BACKLOG.md` and is its
  own feature.
- **A review workflow of any kind.** No approvals, no forge API, no state outside markdown
  and git. Unchanged from 0018.

## Design

**Drafting produces a plan round, not a branch per feature.** A session's drafts land
together on one branch whose name is not claim-shaped — `plan/<something>`, never a leading
four-digit ID, because `Mentions` matches an ID token anywhere in a branch name and would
read it as a claim. The operator iterates locally, accepts what they want to bless, and
merges once. Docs reach `main` at whatever status they reached, `Draft` included.

This deletes the collision rather than patching it: a draft never sits on a
feature-named branch, so the branch name means *claim* and nothing else, and 0014's
delete-the-draft-branch rule can go.

**`status:` orders the board instead of gating it.** `ActiveFeatures` already ranks
`In progress` before `Accepted`; `Draft` joins below them:

| status | `kit next` offers it | means |
|---|---|---|
| In progress | first | someone is on it |
| Accepted | next | the operator blessed it — work it hard |
| Draft | last | invented, not blessed — take it, expect to think |

Autonomy falls out of the sort. Accepting several at once is how the operator says *these
are settled, go*, which is the case they already described wanting; nothing forces them to
do it before work can start.

**`kit claim` accepts a Draft, and refuses an empty one.** This is the load-bearing half.
`unfilled-template` deliberately exempts `Draft` today, and that exemption is only safe
*because* Drafts cannot be claimed. Lift the gate without closing it and `claim` will hand
an agent a doc straight out of `kit new` with three empty sections. So `claim` reuses the
same emptiness test `Accept` should have — empty `## Problem`, `## Design` or `## Tasks`
is refused as `NotYours`, naming what is missing.

What still refuses a claim: a BLOCKING inbox line, a duplicate ID, a terminal status, and
now an unwritten plan. What no longer refuses one: `status: Draft`.

**`## Open questions` stops meaning "stop" and starts meaning "settle these first".** The
protocol already splits the two — questions an agent could not decide go in the section,
questions only the operator can decide go in the inbox — and only the second is a genuine
block. An agent claiming a draft with open questions is expected to settle them, or move
what only a human can answer to the inbox with `kit block`. That is the refine-then-build
loop, and it needs no new machinery.

**`draft-claimed` is re-pointed rather than deleted.** It currently reports building on a
design nobody settled, which under this model is the normal flow and would fire on healthy
projects. The real defect it was reaching for survives: building on a doc whose
`## Open questions` are still open. Same check, honest again.

**`kit drop <id>`** retires a doc that will not be built. Drafts now live on `main`
permanently, so an abandoned one stays on the board for ever; under the old model it was a
branch you deleted. `ship` and `accept` exist because hand-stamping frontmatter is what
`unticked-shipped` and `shipped-no-date` are the wreckage of, and this is the same edit
with the same failure mode.

**The protocol section is rewritten around the plan round**, and the contradiction goes
with it.

## Tasks
- [ ] `kit claim` allows `Draft`, and refuses any doc with an empty Problem, Design or
      Tasks — with a conformance case for each half
- [ ] `ActiveFeatures` ranks Draft below Accepted, and `kit next` offers drafts as work
      rather than as a note that drafts exist
- [ ] `draft-claimed` re-pointed at open questions, with a fires case and a stays-quiet case
- [ ] `kit drop <id>`, refusing a Shipped doc, with `kit help` and README entries
- [ ] Rewrite the drafting rules in `templates/AGENTS-section.md` around the plan round;
      delete the per-feature-branch rule and 0014's delete-the-branch rule
- [ ] `kit help` topics and README updated to describe status as priority, not permission

## Open questions
- **Does the plan round make `duplicate-id` bite often enough to fix now?** `NextID` reads
  only the current checkout, so a plan branch in flight is invisible to the next session
  and a batch collides five IDs where a single draft collided one. `BACKLOG.md` already
  scopes the fix — reserve `refs/kit/ids/<id>` with the same compare-and-swap `claim` uses
  — and its promote-when is "a second agent starts drafting in parallel for real", which
  this model makes routine. Leaning: ship this first and promote that next, because the
  cost is a rename and the fix deserves its own feature rather than a corner of this one.
- **How does a Draft already on `main` get accepted later?** `accept` edits the doc, and
  `main` takes changes by pull request, so blessing one afterwards costs a pull request per
  feature — the per-feature review this feature exists to remove, relocated to acceptance.
  Leaning: accept in rounds too, one small pull request touching only `status:` lines for
  several features at once. Worth confirming before the protocol says it.

## Log
- 2026-08-15 — measured `kit next` against a real project that adopted the kit: five drafts
  on five branches, board reports nothing planned. Scoped from that.
