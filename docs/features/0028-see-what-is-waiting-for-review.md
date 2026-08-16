---
id: 0028
title: See what is waiting for review
status: Draft
created: 2026-08-16
shipped:
check:
checked:
needs: 0027
related: 0013-never-leave-an-agent-without-an-instruction.md
---

## Problem
`kit next` answers *what do I build*. Nothing answers *what needs review*, so the far
side of `ship` is unobserved: a shipped feature leaves the board and the kit never
mentions it again.

Both failures this session live in that blind spot. Four features were shipped and
never pushed, with `audit --strict` green throughout — rule 13 says push and nothing
checks it. And in `observer`, five drafts sat on unmerged branches while the board
reported *"nothing is planned"*. Same gap, reached from opposite ends: work that has
left the builder and not arrived anywhere.

There is also no role for a coordinator to hold. Rule 13 says *"reviewing and merging
are not yours"*, which reads as human-only — but the invariant it protects is that
**the builder does not approve its own work**. A second agent can hold that role; a
human never being involved is a separate decision.

Reading the protocol against the code turned up three more defects in the same
neighbourhood:

- **Rule 2's account of exit 1 is now false.** It says *"exit 1 means somebody was
  first — run `kit next` and take something else."* Since 0023 and 0024 it also means
  unmet `needs:`, a doc nobody wrote, or an id that disagrees with its filename. An
  agent that believes the protocol concludes the feature is taken when it may be
  claimable by nobody, runs `next`, finds it *blocked* instead, and has no account of
  what happened.
- **Nobody opens the pull request.** Rule 5 says *"you do not open the PR"* and rule 13
  says *"push the branch and stop"*. Between them it is never opened.
- **Rule 12's gate is incomplete.** `make check` and `audit --strict` can both be green
  and `ship` still refuse — unmet `needs:`, or the feature claimed by another checkout.
  Unannounced, that reads as a malfunction.

Minor, and worth fixing while the section is open: rule 3 says `next` *"will not offer
you another to start"*. It does list them, under **free, but not in this checkout**,
pointing `kit claim` elsewhere. The behaviour is right and the sentence is not.

## Non-goals
- **A new status.** `Shipped` already means the builder is finished, which is the
  builder's own statement about its own work. Whether it landed is not its call and
  git already answers it — a `Reviewed` status would be a field an agent could set
  about a decision it does not own.
- **Merging, or reading approvals.** No forge API, unchanged from 0018 and 0020. The
  kit reports what is waiting; a coordinator merges with its own tools. This is what
  keeps it identical on GitHub and on a self-hosted GitLab.
- **Judging whether work is any good.** `audit` reports what is checkable.
- **Assigning reviewers.** The kit does not know who exists.
- **Holding the review policy.** Whether a human must see a change before it lands is a
  per-project decision, and it already has a home the controlled party cannot rewrite:
  branch protection on the trunk. A `.kit.toml` saying `review: human` would be a
  control an agent can edit, and a second copy of a fact the forge already enforces —
  the drift `Repo`'s "no config" line exists to prevent. The kit reports; the forge
  enforces.
- **Classifying risk.** "Low risk, let it self-merge" is a real judgement and not a
  checkable one. Protection set per project *is* that judgement, made once by whoever
  is entitled to make it.

## Design
Three git facts already available decide the state, and no new one is stored:

| `status:` | branch on remote | on trunk | state |
|---|---|---|---|
| In progress | yes | no | being built |
| Shipped | no | no | **shipped, never pushed** |
| Shipped | yes | no | **awaiting review** |
| Shipped | — | yes | landed — nothing to say |

**`kit next` gains an `awaiting review` section**, after the board and before the
guidance. It lists the feature, its branch, and how long it has waited. It is the
coordinator's board and it is the same command everyone else runs, so there is one
place to look rather than a second tool to remember.

The closing guidance accounts for it, per 0013: a project with nothing free and three
features awaiting review must not say *"stop"* — the work to do is reviewing.

**`shipped-not-pushed`**, HIGH. A feature whose doc says Shipped, whose branch is not
on the remote, and whose work is not on the trunk exists only on one machine. HIGH
because it is silent, and because the fix is one push — the same reasoning that makes
`stalled` HIGH.

Both read `remoteHeads`, which `Claims` already calls, and both degrade the way the
rest of the kit does: with `--offline`, or a remote that cannot be reached, neither
reports rather than guessing. A claim read from a stale mirror is already handled by
`RefsAreAuthoritative`, and this uses the same answer.

**Each entry says whether the reader built it.** A coordinator that is itself an agent
needs to know which of these it may not merge, and the claim already answers it —
`Repo.Mine` for the checkout it is standing in, the claim branch's authorship
otherwise. Marking it is the kit making the invariant legible, which is all a
well-behaved agent needs; a badly-behaved one is the forge's problem.

**Rule 13 is reworded** from "reviewing and merging are not yours" to the invariant it
means: *you do not merge your own work; a different checkout — another agent, or a
human — reviews it.* The protocol gains the three roles, the barrier that separates
them, and two things that must be true for any of it to hold:

- **Agents need distinct git identities.** Measured on this repo: every commit from
  all four agents in the last round is authored by the operator, so nothing — not the
  kit, not a forge rule requiring a non-author reviewer — can tell builder from
  coordinator. Separation of duties is unverifiable until the identities differ, and
  the credential matters more than the checkout: an agent pushing with the operator's
  token *is* the operator to every check that exists.
- **The policy lives in branch protection.** Enforcement has to sit where the agent
  has no authority. Everything the kit does here is legibility, and it says so rather
  than implying a guarantee it cannot make.

**Enforce, do not instruct — the general rule this session earned.** A constraint in
prose is one an agent can skim, misread, or reason its way past; a refusal in the tool
is one it has to deliberately route around. Measured here: rule 13 was violated twice
by an agent that had read it, and `ship`'s coded refusal was obeyed the first time it
fired — on that same agent, minutes later.

Three tiers, and a rule should sit at the highest one it can reach:

| tier | example | an agent gets past it by |
|---|---|---|
| the remote | branch protection, credentials | nothing it can do |
| the kit | `claim` refusing unmet `needs:` | not using the kit |
| prose | "push the branch and stop" | reading carelessly |

So a protocol sentence is a last resort, not a first one: whenever a rule is added
here, the question is what would make it unnecessary. Where prose is genuinely the
only option — judgement, taste, what to write down — it says so plainly rather than
sounding like a guarantee.

## Tasks
- [ ] `Repo.AwaitingReview()` — Shipped, branch on the remote, not reachable from the
      trunk — reusing `remoteHeads` and `trunkRef`
- [ ] `kit next` prints an `awaiting review` section, in text and `--json`
- [ ] Closing guidance names reviewing as the work when that is all there is
- [ ] `shipped-not-pushed` at HIGH, quiet offline and quiet when the work is on the
      trunk; conformance for each
- [ ] Each awaiting-review entry says whether the reader built it, so a coordinator
      agent knows what it may not merge
- [ ] Rule 13 reworded around separation of duties, with the three roles, the landing
      barrier, distinct agent identities, and the policy living in branch protection,
      in `templates/AGENTS-section.md`
- [ ] Rule 2 states what exit 1 now means, rule 12 warns that `ship` has refusals the
      gate does not cover, rule 3 describes what `next` actually does, and somebody is
      named as opening the pull request
- [ ] "Enforce, do not instruct" in the protocol and `docs/TECH.md`, with the three
      tiers, so a future rule is weighed against making itself unnecessary
- [ ] README and `kit help next` describe the review board and who reads it

## Log
- 2026-08-16 — scoped after shipping four features without pushing any of them, with
  every check green the whole way.
- 2026-08-16 — read the whole protocol against the code and folded the findings in:
  rule 2's account of exit 1 was falsified by 0023 and 0024, rule 12's gate is not the
  whole gate, and nobody opens the pull request.
- 2026-08-16 — the review policy stays out of the kit. It is optional per project and
  enforceable only on the remote: anything the kit checks, an agent skips by calling
  git directly, so a config flag would be a control the controlled party can rewrite.
  Legibility here, enforcement in branch protection.
