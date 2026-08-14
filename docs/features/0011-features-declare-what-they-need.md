---
id: 0011
title: Features declare what they need
status: In progress
created: 2026-08-14
shipped:
check:
checked:
related: 0010-make-the-board-honest-under-several-agents.md
---

## Problem
An agent decomposed a project into seven features unattended, and the decomposition
was good: boundaries clean, each doc carryable by one agent, dependencies reasoned
about in prose. `0004` even records *"No rate limiting. That is 0005, and it depends
on the key identity added here."*

The kit cannot read that sentence. `kit next` offers work by status and lowest ID, so
it will hand an agent a feature whose prerequisite has not been built, and it has no
way to say which of several features could be worked at the same time. With a human
sequencing merges that is invisible. With agents claiming work unattended it is the
thing that breaks first, and it breaks silently — the agent starts, builds against a
foundation that is not there, and finds out at merge.

Ordering by ID is not the answer. An ID is an identity, not a position: a prerequisite
discovered late already has a higher number, and renaming to fix it breaks the rule
that IDs are stable and files never move. It also conflates when something was thought
of with when it must be built, which are different facts that drift apart immediately.

0005, 0006 and 0010 each deferred this, every time on the grounds that the signal
would be real usage rather than anticipation. Seven features drafted from one prompt,
with a human about to stop sequencing them, is that signal.

## Non-goals
- **A scheduler.** No critical path, no automatic assignment, no fan-out planning.
  `next` reports what is ready; who picks it up is `kit claim` and the operator.
- **Auditing whether the declared needs are correct.** A missing dependency is a
  judgement call and no heuristic has it. The check confirms the ids exist and the
  graph terminates, nothing more.
- **Transitive display.** If A needs B and B needs C, A waits on B. Saying so once is
  enough; expanding chains in the output makes the board harder to read than the docs.
- **A second ordering rule.** `related:` stays what it is — reading material. `needs:`
  is the only field that changes what `next` offers, and the two must not blur.
- **Blocking on anything but features.** Not INBOX questions, not external events.
  Those are already `kit block`, and one blocking mechanism is enough.

## Design
**`needs:` in the frontmatter, a list of feature ids.** It sits beside `related:` and
means something stricter: this cannot start until those have shipped. A feature with
no `needs:` can start now, which is the common case and stays free of ceremony.

**Unmet needs make a feature blocked, in the group that already exists.** `next`
already separates ready from blocked for INBOX questions, and this is the same fact
from a different source: work that cannot start yet, with a reason attached.

**The ready set is the parallelism answer, and that is the point.** No prose about
serial versus parallel is needed and none should be added: one ready feature means the
work is a chain, three means they can be worked at once. It is derived from what the
docs already say rather than declared separately and left to rot.

**Shipped is the bar, not merged.** `kit ship` is the moment a feature's work is done
and the kit knows it; merging is the human's and happens right after. This leaves a
window where a dependency is Shipped but not yet on the trunk — small, and worth
accepting rather than teaching the kit to read merge state, which would make the
answer depend on how quickly somebody clicked a button.

**Three checks, because there are three ways to be wrong.** A `needs:` naming an id
that does not exist, or one that is Dropped or Superseded, can never be satisfied —
the feature waits forever and `next` never says why. A cycle means nothing in it can
ever start, which is stuck work rather than a typo, so it reads at the severity that
fails a build. And a feature Shipped while something it needs has not is the docs
contradicting themselves: either the dependency was wrong or the work was built on a
foundation that is not there, and both are worth knowing.

Nothing beyond those three. A feature that needs nothing and that nothing needs is not
a dead element, it is an independent feature — the case this whole field exists to make
visible. And a chain hanging off an unsatisfiable need resolves when the one broken
link is fixed, so reporting every doc downstream of it would be noise with one cause.

**The guidance goes where the drafting happens.** The template's `needs:` prompt is
what an agent reads while writing the doc, so that is where the hard part belongs: a
feature is independent only if it can start now, and clean boundaries hide couplings
that make it not — a shared composition root, a shared router or registry, and
numbered files in a shared sequence. Migrations are the canonical case, and two
features each adding the next number merge clean and break at runtime.

## Tasks
- [x] `needs:` parsed from the frontmatter and written into the template beside `related:`
- [x] `kit next` reports a feature with unshipped `needs:` as blocked, naming what it waits on
- [ ] Audit: a `needs:` that can never be satisfied, a cycle, and shipping ahead of what you need
- [ ] Protocol and template: what belongs in `needs:`, and the couplings that hide behind clean boundaries

## Open questions
<!-- none -->

## Log
- 2026-08-14 — added the third check after asking what a dead path in this graph would
  be. Most candidates are not defects: a feature nothing needs is independent, which is
  the point, and a chain below a broken link has one cause and one fix. Shipping ahead
  of a dependency is the only remaining case where the docs disagree with themselves.
- 2026-08-14 — reopened after three deferrals. What changed is not the design but who
  sequences the work: a human merging in order made dependencies invisible, and agents
  claiming unattended makes them the first thing to break.
- 2026-08-14 — rejected ID order as the ordering rule. An ID is an identity and a
  prerequisite found late already has a higher one, so encoding order there is
  unrepresentable at exactly the moment it matters.
