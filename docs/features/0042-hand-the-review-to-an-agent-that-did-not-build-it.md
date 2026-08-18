---
id: 0042
title: Hand the review to an agent that did not build it
status: Draft
created: 2026-08-18
shipped:
check:
checked:
review:
needs:
related: docs/features/0028-see-what-is-waiting-for-review.md, docs/features/0013-never-leave-an-agent-without-an-instruction.md
---

## Problem

The loop the vision describes runs unattended until its last step, which is performed by
improvisation. `next` tells an idle agent to review one somebody else built — 0013 closed
the vacuum of telling it to stop — and then leaves it there: it is the only instruction on
the board with no command behind it. `claim` prints the worktree to work in, `new`
scaffolds, `ship` verifies and stamps; a reviewer is told to review and has to invent which
branch, what to diff it against, what to check, and what to do when it is good.

The rule that keeps the two agents distinct is prose at the tier `docs/TECH.md` says prose
fails. Its own measurement: the rule against merging your own work was broken twice by an
agent that had read it, and `ship`'s coded refusal was obeyed the first time it fired, on
that same agent minutes later.

## Non-goals

- **Running anything.** Not the tests, not the build. The kit is not a test runner, and a
  verdict it produced would be one nobody could audit. It says what to check; the reviewer
  checks it.
- **Merging.** Branch protection is the floor and the forge owns the merge. `review` names
  the command for the forge `kit init` detected, the way `claim` names the worktree command.
- **Recording a verdict.** No new state and no field: approval is the merge, and the docs
  and branches stay the whole state.

## Design

**`kit review [<id>]`, the command the board's instruction has always lacked.** With no id
it takes the feature that has waited longest and that this checkout may review; with one it
takes that feature. It fetches the branch, prints the range to read — `<trunk>...<branch>`,
so the diff is the branch's own work and not the trunk moving underneath it — and states
what a review has to establish: every `## Tasks` box ticked, `kit audit` clean *on the
branch*, and the project's own gate green, run by the reviewer.

**It refuses the builder, and that is the point of it.** Exit 1, `claim`'s "the answer is
no", when this checkout holds the claim or the branch's commits carry this checkout's git
identity. The refusal names what to do instead rather than only saying no. The identity
half is weak and the message says so: agents sharing one credential are indistinguishable
here — measured, four agents' commits on this repo all carried the operator's name — so a
reviewer wrongly refused reviews from a checkout that did not build it, and the unambiguous
half, the claim this checkout stands on, is the one that carries the guarantee.

**`review: human` stops the command, not just the board.** 0029 made a person's review
legible; here it refuses to print a merge command at all and says a person must take it.
Any non-empty value counts, which is 0029's fail-safe direction.

## Tasks
- [ ] `kit review [<id>]`: fetch, the diff range, what a review must establish, and the
      merge command for the detected forge; conformance on each part
- [ ] Refusal at exit 1 when this checkout built it, naming what to do instead and what the
      shared-identity case means; conformance both ways
- [ ] `review: human` refuses the merge command and says a person takes this one
- [ ] `next`'s review guidance names `kit review`, so the instruction and the command are
      one thing; `kit help review` and README carry it

## Open questions
- What happens when a review says no? Today nothing does: the doc says Shipped, the branch
  sits unmerged, and the board cannot tell a feature nobody has reviewed from one that was
  rejected — so agents will pick it up forever. Leaning: `kit review <id> --changes "<why>"`
  puts the doc back to Accepted *on the branch* with the reason in its `## Log`, which
  returns it to the board as work its claim branch still holds. That is a second writer on
  someone else's branch and it needs deciding before it is built, not after.

## Log
- 2026-08-18 — scope settled before drafting: the kit is not a test runner, so `review` sets
  a review up and refuses the wrong reviewer rather than producing a verdict.
