# Backlog

Ideas that are not features yet. Nothing here is scheduled — the default state is **stays
frozen**. Promote one with `kit new` only when its trigger has actually fired.

An entry earns its place by naming a value somebody would notice and a trigger that can be
observed. An entry whose trigger has never fired, or whose fix costs more than the problem,
is deleted rather than kept as an option.

A **Promote when** line reading *fired* is waiting for a round rather than for a signal:
the observation that justifies it has already been made, and it is written beside it.

## The work is only on the remote once, at the end

`kit claim` pushes `HEAD`, so the branch that claims a feature carries the doc and no work,
and the kit's only other push is `release` deleting that same ref. Everything after that is
local until somebody pushes for review. Liveness is read from `git log --all`, which is this
machine's refs.

So the mutual visibility the multi-agent model rests on is the shared local object store,
not the remote. On one machine with worktrees that holds. Across two it does not, and it
fails silently: `stalled` never ages a claim carrying no commits beyond the doc, which is
exactly what a live agent looks like from anywhere else. Two checkouts of one feature were
observed reporting different progress for it in the same instant; since 0081 the board
reads the local head when it contains the advertised tip, and a checkout that cannot reach
the tip shows no count at all, which is silence rather than liveness.

The fix is that the push becomes the heartbeat — the claim branch gets one when a box is
ticked — after which `stalled` can age from the remote branch's commit date and be true from
any machine. It has to degrade for `--offline` and for projects with no remote.

**Value:** a board that means the same thing on every machine.
**Promote when:** a second machine joins, or a claim's work is lost with the machine holding it.

## `tag-stale` reads this machine's tags, and `@latest` reads the remote's

`TagDistance` describes against the local refs, so a tag cut and never pushed silences the
check while an install still resolves to the one before it. It is `submitted-not-pushed` for
tags, and that check exists because the branch version of it happened. `make release`
pushes what it cuts, so the flow the kit defines does not produce this; a hand-cut
`git tag -a` does. The remote's tags would cost no extra process — `audit` already makes an
`ls-remote` round trip and asks it for `--heads` alone.

**Value:** the check about what `@latest` resolves to reads what `@latest` reads.
**Promote when:** a tag is cut and left unpushed, or `audit` calls a project current that
`go install` disagrees with.

## `kit new` cannot draft while its remote is unreachable

The id is reserved by a push, so a project that has a remote and cannot reach it gets exit 2
after the push times out — where before 0058 `kit new` touched no network at all and always
worked. A project with *no* remote still drafts and says what it lost, which is the case the
design covers; an unreachable one is the case it does not. `next` and `audit` both take
`--offline`, and `Repo.Offline` already exists for it.

**Value:** drafting works on a plane, at the cost the operator asked for.
**Promote when:** an agent is seen unable to draft, or a round is delayed by a remote that
was down.

## A reservation cannot be released before the horizon, and nothing else can reach it

`kit drop <id> --reservation` refuses a reservation younger than the `stalled` horizon, which is right for
a draft in flight and wrong for one whose branch was deleted minutes ago. There is no
`--force`, so the only way past is `git push origin --delete refs/kit/ids/<id>` — the raw
command 0046 gave `release` a flag to avoid, for the reason it gave there: an agent that
cannot do it through the kit reaches for git, and the kit stops being the record.

**Value:** the kit stays the record for the one disposal path that leaves it.
**Promote when:** fired — 0074 was released by hand on 2026-09-01.

## The comments already written carry the incidents their feature docs hold

21% of non-test source is comment — 2,110 lines of 10,181 on 2026-10-29 — and 104 blocks
of six lines or more hold 914 of them. `OnlyHere` spends 20 lines, seven of which restate 0039's
`## Problem`. 0073 added the rule and cut nothing, so every block written before it still
carries the incident beside the constraint.

**Value:** a reader of the code gets the constraint without the archaeology.
**Promote when:** fired — measured 2026-09-01 and again 2026-10-29, and the rule that judges it now exists.

## Review complete changes locally before publishing

The operator wants to inspect a branch diff, give an agent comments and review the fixes
before any remote publication. The first round focuses on PR review in 0078. A later
local workflow needs an empty review base, an exact reviewed commit, a clear feedback loop
and human integration without inventing an approval marker the agent can grant itself.

**Value:** the operator can review all setup and implementation locally before publishing.
**Promote when:** the operator chooses to work without PR review or before adding a remote.

## Compress battle-tested agent guidance only with paired evidence

Keep the original Go/Python checklists, AGENTS.md and embedded Work protocol. Baseline: 237ab912c288dc9722768132d17eb35e8ffafce2;
the shorter 0080 candidates are at 33ee31f. The operator deferred their compression.
Compare unchanged task fixtures and runner settings in fresh, repeated A/B sessions;
review rule coverage and task quality before considering context savings. Protocol cases
must cover human approval, claim ownership, failed checks and safe recovery.

**Value:** smaller instruction context without losing established behavior or engineering guidance.
**Promote when:** paired behavioral evidence supports a candidate and the operator chooses
guidance compression; smaller word counts or passing kit tests alone do not qualify.

## Generated CI does not run `kit audit`

The stack CI templates skip `kit audit --strict` because installing a private module needed
a credential in every project; their own comment says adding it back costs nothing once the
kit is public. Drift then reaches review unchecked in every scaffolded project.

**Value:** every project's CI refuses the HIGH findings this repo's CI already refuses.
**Promote when:** the repository is public.

## Per-claim git reads are not batched

`next` spends about four git processes per live claim and `audit` about two, plus a history
walk per claim in `stalled`. Measured 2026-10-29: 2.4 s at 3 claims, about 6 s for `next` and
12 s for `audit` at 200. Batching through `cat-file --batch` and one `log --all` pass brings
200 claims to about 3 s.

**Value:** the board stays one round trip at fleet scale.
**Promote when:** a project holds 50 live claims at once.
