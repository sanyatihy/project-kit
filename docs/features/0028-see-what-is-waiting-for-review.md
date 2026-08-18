---
id: 0028
title: See what is waiting for review
status: Shipped
created: 2026-08-16
shipped: 2026-08-17
check:
checked:
needs: 0027
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem
`kit next` answers *what do I build*. Nothing answers *what needs review*, so the far side
of `ship` is unobserved. Both failures live there: four features were shipped
and never pushed with `audit --strict` green throughout, and in another project five drafts
sat on unmerged branches while the board reported *"nothing is planned"*.

There is also no role for a coordinator to hold. Rule 13's *"reviewing and merging are not
yours"* reads as human-only, but the invariant it protects is that **the builder does not
approve its own work** — which a second agent can hold.

Reading the protocol against the code turned up three more defects in the same
neighbourhood: rule 2's account of exit 1 was falsified by 0023 and 0024 and never updated,
so an agent refused for an unmet `needs:` concludes the feature is taken; nobody opens the
pull request, between rule 5's *"you do not open the PR"* and rule 13's *"push and stop"*;
and rule 12's gate is not the whole gate, since `ship` refuses past it.

## Non-goals
- **A new status.** `Shipped` is the builder's statement about its own work; whether it landed is not its call and git answers it.
- **Merging, or reading approvals.** No forge API — that is what keeps this identical on GitHub and self-hosted GitLab.
- **Holding the review policy.** Per-project, and it already has a home the controlled party cannot rewrite: branch protection. A `.kit.toml` saying `review: human` would be a control an agent can edit.
- **Classifying risk.** "Low risk, let it self-merge" is a real judgement and not a checkable one.

## Design
Three facts already available decide the state, and none is stored:

| `status:` | branch on remote | on trunk | state |
|---|---|---|---|
| In progress | yes | no | being built |
| Shipped | no | no | **shipped, never pushed** |
| Shipped | yes | no | **awaiting review** |
| Shipped | — | yes | landed — nothing to say |

**`kit next` gains an `awaiting review` section**, after the board and before the guidance:
the feature, its branch, how long it has waited. The same command everyone runs, so a
coordinator has one place to look. The closing guidance accounts for it — a board with
nothing free and three features waiting must not say *"stop"*.

**`shipped-not-pushed`, HIGH** — Shipped, no branch on the remote, not on the trunk: the
work exists on one machine. HIGH because it is silent and the fix is one push. Both read
`remoteHeads` and both go quiet offline rather than guessing.

**Each entry says whether the reader built it**, from the claim it stands on or the branch's
authorship — all a well-behaved coordinator needs; a badly-behaved one is the forge's problem.

**Rule 13 is reworded** to the invariant it means — you do not merge your own work — with the
three roles and two things it rests on: **distinct git identities** (measured: every commit
from four agents here carries the operator's name) and **branch protection**, where the agent
has no authority.

**Enforce, do not instruct**, the general rule this earned: rule 13 was violated twice by
an agent that had read it, and `ship`'s coded refusal was obeyed the first time it
fired, on that same agent minutes later. A rule belongs at the highest tier it can reach —
the remote, then the kit, then prose — so a protocol sentence is a last resort. The tiers
are in `docs/TECH.md`.

## Tasks
- [x] `Repo.AwaitingReview()` — Shipped, branch on the remote, not reachable from the
      trunk — reusing `remoteHeads` and `trunkRef`
- [x] `kit next` prints an `awaiting review` section, in text and `--json`
- [x] Closing guidance names reviewing as the work when that is all there is
- [x] `shipped-not-pushed` at HIGH, quiet offline and quiet when the work is on the
      trunk; conformance for each
- [x] Each awaiting-review entry says whether the reader built it, so a coordinator
      agent knows what it may not merge
- [x] Rule 13 reworded around separation of duties, with the three roles, the landing
      barrier, distinct agent identities, and the policy living in branch protection,
      in `templates/AGENTS-section.md`
- [x] Rule 2 states what exit 1 now means, rule 12 warns that `ship` has refusals the
      gate does not cover, rule 3 describes what `next` actually does, and somebody is
      named as opening the pull request
- [x] "Enforce, do not instruct" in the protocol and `docs/TECH.md`, with the three
      tiers, so a future rule is weighed against making itself unnecessary
- [x] README and `kit help next` describe the review board and who reads it
- [x] `remoteHeads` is asked once per command, not once per Shipped feature

## Log
- 2026-08-16 — the review policy stays out of the kit: it is optional per project and
  enforceable only on the remote, so a config flag would be a control the controlled party
  can rewrite.
- 2026-08-17 — the status is read from the claim branch, not this checkout. A coordinator
  stands on the trunk, where an unmerged feature's doc still says what it said when the
  round landed, so a board read from the working tree would be empty exactly when it matters.
- 2026-08-17 — `shipped-not-pushed` needs two independent negatives: the remote's live branch
  list cannot see a feature merged and then deleted, and the tracking refs are only as fresh
  as the last fetch.
- 2026-08-17 — a box added late: `remoteHeads` was unmemoised and `audit` on this repo went
  from under a second to 1m57s with 24 shipped features.
