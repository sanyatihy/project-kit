---
id: 0014
title: Reach the agent, not just the reader
status: In progress
created: 2026-08-14
needs:
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem
0013 closed the state where nothing is free, and a second agent took the claimed
feature's boxes anyway. Rather than chase that one bug, the whole protocol was read
against every state an agent can reach. Seven defects, in four kinds.

**A feature drafted the documented way cannot be claimed.** 0010 settled that a draft
waits on a branch and nothing is committed to `main`. That branch is named after the
doc, which is exactly what 0006 made the claim. Measured end to end: draft on the
branch, human accepts and merges, then

```
  not claimed: 0001 A drafted feature is already claimed by 0001-a-drafted-feature
  taken   0001  0/2  A drafted feature — 0001-a-drafted-feature
  nothing is free
```

The feature is permanently unstartable, shown as taken by a branch nobody is on, and
since 0013 every agent is now told to stop. Following the flow exactly bricks the
work. The kit cannot fix this by inspection: before the merge a draft branch carries
the doc and a fresh claim branch carries nothing, so they differ — but *after* the
merge both are identical to the trunk. Content cannot tell them apart, so the branch
has to be deleted when the draft is accepted.

**The instruction is not in the format agents read.** `--json` is documented "for
agents" and carries no conclusion at all — none of 0013's closing line survives into
it. Worse, it attaches the next unticked box to a feature it has just reported as
somebody else's, which the text output has never done:

```
"taken": [{ "id": "0001", "task": "internal/config: typed Config, env loading …" }]
```

**The protocol never reached the project.** The `## Work protocol` section is only
rewritten by `kit init`, so a project keeps whatever rules it was scaffolded with.
The project where this happened has none of 0013's rules and nothing said so.
Shipping a protocol change does not deliver one.

**And the protocol contradicts itself in three places**, each from a change that
never swept its references:

- Rule 5 says tick a box "in the same PR that implements it"; rule 13 says opening
  the PR is not the agent's. Added by 0009, and rule 5 was never revisited.
- Rule 12 says set `status: Shipped` and the date by hand. `kit ship` exists to
  remove exactly that mistake — 0005 built it because doing it by hand is where
  `unticked-shipped` and `shipped-no-date` came from — and the protocol never
  mentions it. Nor `make check`, nor `kit audit --strict`: the gate an agent must
  pass before shipping is named nowhere.
- Two cross-references point at the wrong rules. "Rule 8 is where a question goes"
  means rule 9; "rule 4's business" means rule 6. Both were right until 0007 and
  0009 inserted rules above them. Numbered references in prose break silently on
  every insertion, which is the real defect — the two wrong numbers are symptoms.

## Non-goals
- **Making the kit enforce any of it.** `audit` never writes and `next` never
  refuses. Every fix here is a clearer answer or a truer report.
- **Auto-refreshing a project's protocol.** Editing `AGENTS.md` under a running agent
  is worse than a stale rule. Report it and name the command.
- **Refusing to report taken features.** Knowing what is taken is how an agent knows
  not to wait for it. The board keeps what it shows; it stops attaching a task to it.
- **Teaching `kit claim` to delete branches.** Deleting somebody's branch to take
  their work is the one thing a claim must never do. It says what is wrong instead.
- **Versioning the protocol.** Comparing the text answers the only question asked.

## Design
**The claim namespace is freed when a draft is accepted.** The branch that held the
draft is deleted at merge, which is what a normal merge does anyway. `kit claim` gains
the case it cannot currently explain: an existing branch that is identical to the
trunk is either a fresh claim or a leftover draft, and since the tool cannot tell, it
says so and names both fixes rather than reporting a claim that may not exist.

**`--json` carries the conclusion.** A `guidance` field holding the same sentence the
text output ends with, produced from one decision so the two cannot drift. And a
feature that is not yours carries no `task` in either format: `task` means "the box to
do now", which for somebody else's feature is a category error. `done` and `total`
stay — progress is a fact, not an invitation.

**`audit` reports a stale protocol.** Both halves are already in hand: the project's
section and the kit's embedded copy. This is the `tech-drift` shape — not a defect in
the project, a signal that something upstream moved. MEDIUM, because agents following
superseded rules is how the last two failures happened. It fires after every protocol
change, which is the truth about something that matters.

**Cross-references stop using numbers.** Rules are cited by their bolded name, so
inserting a rule cannot silently misdirect the next reader. This removes the class,
not the two instances.

**The protocol names the commands that exist.** Shipping is `kit ship`, and the gate
before it is `make check` and `kit audit --strict`. A protocol that describes the
manual path for something the tool does is a protocol that teaches the mistake.

## Tasks
- [ ] `kit claim` explains an existing branch it cannot attribute, and the protocol frees the name on acceptance
- [ ] `--json` carries the conclusion, and no format attaches a task to a feature that is not yours
- [ ] Audit: the project's protocol section differs from the installed kit's
- [ ] Protocol: ship with `kit ship`, pass `make check` and `kit audit --strict` first
- [ ] Protocol: cite rules by name, and fix the three contradictions

## Open questions
<!-- none -->

## Log
- 2026-08-14 — read the whole protocol against every state rather than fixing the
  reported bug. Four of the seven were invisible from the failure that prompted it,
  and the worst — a drafted feature that can never be claimed — had not been hit yet
  only because no project had reached that step with a second agent waiting.
- 2026-08-14 — the recurring shape is a change that lands correctly and does not sweep
  its references: 0009 added a rule and left rule 5 contradicting it, 0010 settled
  where drafts live and left the branch colliding with the claim, 0013 wrote the
  instruction into one of the two output formats. The fixes here are individual; the
  lesson is that the protocol is a system and needs reading as one.
