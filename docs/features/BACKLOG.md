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
observed reporting different progress for it in the same instant; since 0056 the board
reads the tip `ls-remote` advertises and a checkout that cannot reach it shows no count at
all, which is silence rather than liveness.

The fix is that the push becomes the heartbeat — the claim branch gets one when a box is
ticked — after which `stalled` can age from the remote branch's commit date and be true from
any machine. It has to degrade for `--offline` and for projects with no remote.

**Value:** a board that means the same thing on every machine.
**Promote when:** a second machine joins, or a claim's work is lost with the machine holding it.

## Source comments cite protocol rules that have moved

`internal/kit/audit.go` and `cmd/kit/conformance_audit_test.go` still cite rule 21,
which does not exist in the current protocol. Other comments cite a number whose
meaning changed. Removing the former rule-number test did not correct those references.
Replace the references with the constraint they mean; no numbering gate is required.

**Value:** source comments identify the actual constraint a reader must preserve.
**Promote when:** fired — obsolete references remain in the 0076 review.

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

## `kit new` help omits some refusal conditions

Help describes branch and write-collision guards but omits rejected titles, missing
setup, ambiguous remotes, failed reservation and exhausted ids. The former refusal
registry is gone; that does not make the missing command guidance obsolete.
Document the invocation conditions without restoring a refusal-count or prose-format gate.

**Value:** a caller can understand the command's preconditions before attempting it.
**Promote when:** fired — the 0076 review found these conditions absent from help.

## Nothing tells an agent which language a project's docs are written in

Kit generates no prose: the templates and what you typed are its only sources. The language
of a doc body is therefore the agent's choice, remade each session. A project that wants
Russian gets English unless its own house rules say otherwise, and that sentence is written
by hand in every project that wants one.

`kit init --lang ru` would seed it above the marker in `AGENTS.md`, which `init` never
rewrites and the project owns afterwards — any language, with no translated templates and
no model in the binary. The seeded line has to name what stays English whatever the
language: the headings the parsers read, the frontmatter keys, and `Draft | Submitted |
Dropped`. Without that an agent translates `## Tasks` and the board sees no tasks at all. A
re-run under a different language would not update the line, and `kit new` still takes an
ASCII title — `Slug` strips to `[a-z0-9]`.

**Value:** a project says what language its docs are written in once, and every agent is told.
**Promote when:** a second project needs non-English docs, or an agent writes English into
one that asked for Russian.

## A reservation cannot be released before the horizon, and nothing else can reach it

`kit drop <id> --reservation` refuses a reservation younger than the `stalled` horizon, which is right for
a draft in flight and wrong for one whose branch was deleted minutes ago. There is no
`--force`, so the only way past is `git push origin --delete refs/kit/ids/<id>` — the raw
command 0046 gave `release` a flag to avoid, for the reason it gave there: an agent that
cannot do it through the kit reaches for git, and the kit stops being the record.

**Value:** the kit stays the record for the one disposal path that leaves it.
**Promote when:** fired — 0074 was released by hand on 2026-09-01.

## The comments already written carry the incidents their feature docs hold

27% of non-test source is comment — 1,969 lines of 7,239 — and 105 blocks of six lines or
more hold 927 of them. `OnlyHere` spends 20 lines, seven of which restate 0039's
`## Problem`. 0073 added the rule and cut nothing, so every block written before it still
carries the incident beside the constraint.

**Value:** a reader of the code gets the constraint without the archaeology.
**Promote when:** fired — measured 2026-09-01, and the rule that judges it now exists.

## Compress battle-tested language checklists only with paired evidence

Keep the original Go/Python checklist wording. Baseline: 237ab912c288dc9722768132d17eb35e8ffafce2;
the shorter 0080 candidates are at 33ee31f. The operator deferred their compression.
Compare unchanged task fixtures and runner settings in fresh, repeated A/B sessions;
review rule coverage and task quality before considering context savings.

**Value:** smaller instruction context without losing established engineering guidance.
**Promote when:** paired behavioral evidence supports a candidate and the operator chooses
checklist compression; smaller word counts or passing kit tests alone do not qualify.
