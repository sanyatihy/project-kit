---
id: 0014
title: Reach the agent, not just the reader
status: Shipped
shipped: 2026-08-14
created: 2026-08-14
needs:
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem
0013 closed the "nothing is free" state and a second agent took the claimed feature's boxes
anyway, so the whole protocol was read against every state an agent can reach. Seven
defects, in four kinds.

**A feature drafted the documented way cannot be claimed.** A draft waits on a branch named
after its doc — which is what makes a claim. After the merge the feature is permanently
unstartable and shown as taken by a branch nobody is on. Content cannot fix it: before the
merge a draft branch carries the doc and a fresh claim carries nothing, but after it both
are identical to the trunk.

**The instruction is not in the format agents read.** `--json` carried no conclusion, and
attached the next unticked box to features it had just reported as somebody else's.

**The protocol never reached the project.** `## Work protocol` is rewritten only by
`kit init`, so a project keeps whatever it was scaffolded with. Shipping a protocol change
does not deliver one.

**The protocol contradicts itself in three places**, each a change that never swept its
references: rule 5 said tick a box "in the same PR" while rule 13 says opening the PR is not
the agent's; rule 12 described the hand edit `kit ship` exists to remove, and named neither
`make check` nor `audit --strict`; and two cross-references pointed at the wrong rules after
0007 and 0009 inserted rules above them.

## Non-goals
- **Making the kit enforce any of it.** Every fix here is a clearer answer or a truer report.
- **Auto-refreshing a project's protocol.** Editing `AGENTS.md` under a running agent is worse than a stale rule.
- **Teaching `kit claim` to delete branches.** Deleting somebody's branch to take their work is the one thing a claim must never do.
- **Versioning the protocol.** Comparing the text answers the only question asked.

## Design
**The claim namespace is freed when a draft is accepted** — the branch is deleted at merge,
which is what a normal merge does anyway. `kit claim` gains the case it cannot attribute: a
branch identical to the trunk is either a fresh claim or a leftover draft, and it names both
fixes rather than reporting a claim that may not exist.

**`--json` carries the conclusion**, produced from one decision so the formats cannot drift,
and no format attaches a `task` to a feature that is not yours: "the box to do now" for
somebody else's feature is a category error. `done` and `total` stay — progress is a fact.

**`audit` reports a stale protocol**, the `tech-drift` shape: not a defect in the project, a
signal that something upstream moved. MEDIUM, because agents following superseded rules is
how the last two failures happened.

**Cross-references stop using numbers** — rules are cited by name, which removes the class
rather than the two instances.

## Tasks
- [x] `kit claim` explains an existing branch it cannot attribute, and the protocol frees the name on acceptance
- [x] `--json` carries the conclusion, and no format attaches a task to a feature that is not yours
- [x] Audit: the project's protocol section differs from the installed kit's
- [x] Protocol: ship with `kit ship`, pass `make check` and `kit audit --strict` first
- [x] Protocol: cite rules by name, and fix the three contradictions

## Log
- 2026-08-14 — read the whole protocol rather than fixing the reported bug: four of the seven
  were invisible from the failure that prompted it, and the worst — a drafted feature that
  can never be claimed — had not been hit only because no project had reached that step.
- 2026-08-14 — the recurring shape is a change that lands correctly and does not sweep its
  references. The protocol is a system and needs reading as one.
