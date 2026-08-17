---
id: 0011
title: Features declare what they need
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
check:
checked:
related: 0010-make-the-board-honest-under-several-agents.md
---

## Problem
An agent decomposed a project into seven features and reasoned about dependencies in prose:
*"No rate limiting. That is 0005, and it depends on the key identity added here."* The kit
cannot read that sentence. `next` offers by status and lowest ID, so it hands an agent a
feature whose prerequisite does not exist, and it cannot say which features could be worked
at once. With a human sequencing merges that is invisible; with agents claiming unattended
it breaks first and silently.

## Non-goals
- **A scheduler.** No critical path, no assignment; `next` reports what is ready.
- **Auditing whether the declared needs are correct.** A missing dependency is a judgement call; the checks confirm the ids exist and the graph terminates.
- **Transitive display.** If A needs B and B needs C, A waits on B — saying so once is enough.
- **Blocking on anything but features.** INBOX questions are `kit block`; one blocking mechanism is enough.

## Design
**`needs:` in the frontmatter, a list of feature ids**, beside `related:` but stricter: this
cannot start until those shipped. Empty means it can start now, which is the common case and
stays free of ceremony.

**ID order was rejected.** An ID is an identity, and a prerequisite discovered late already
has a higher one, so encoding order there is unrepresentable exactly when it matters.

**The ready set is the parallelism answer** — one ready feature means a chain, three means
three agents. Derived from what the docs already say rather than declared separately.

**Shipped is the bar, not merged.** Reading merge state would make the board depend on how
fast somebody clicked; the cost is a short window where a dependency is done but not on the
trunk.

**Three checks, because there are three ways to be wrong**: a need that can never be
satisfied (missing, Dropped, Superseded), a cycle, and shipping ahead of a need. Nothing
beyond those — a feature nothing needs is independent, not dead, and a chain below a broken
link has one cause.

**The guidance goes where drafting happens**, in the template's `needs:` prompt: clean
boundaries hide couplings — a shared composition root, a shared router, numbered migrations
where two features each add the next number, merge clean and break at runtime.

## Tasks
- [x] `needs:` parsed from the frontmatter and written into the template beside `related:`
- [x] `kit next` reports a feature with unshipped `needs:` as blocked, naming what it waits on
- [x] Audit: a `needs:` that can never be satisfied, a cycle, and shipping ahead of what you need
- [x] Protocol and template: what belongs in `needs:`, and the couplings that hide behind clean boundaries

## Log
- 2026-08-14 — reopened after three deferrals. What changed is who sequences the work: a
  human merging in order made dependencies invisible; agents claiming unattended make them
  the first thing to break.
