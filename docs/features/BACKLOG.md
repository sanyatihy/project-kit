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

## A merged claim holds its slot until the trunk ref catches up

`AwaitingReview` skips a branch the trunk contains, comparing against `trunkRef`, which
prefers the local `refs/heads/<trunk>`. A checkout that has not pulled since the merge keeps
listing the feature as awaiting review — and the protocol tells every other agent not to
touch it. Most forges do not delete a source branch on merge, and nothing says to run
`kit release` afterwards, whose work guard passes on a merged branch.

**Value:** a finished feature stops occupying a slot on everyone's board.
**Promote when:** a merged feature is seen holding a slot, or a second machine reads a board
this one has already cleaned.

## One agent that plans and builds in the same session has no path

The protocol has two: a drafting round lands on the trunk, then a builder claims from it. An
agent asked to do both in one session can do neither cleanly — `claim` refuses until the
round lands, and `ship` needs a claim, so the work ends as a plan branch carrying docs whose
boxes are ticked and whose status is still `Draft`. It has happened twice.

Either the protocol names this as the coordinator's case and says what to do, or `ship`
learns a path that does not require a claim branch.

**Value:** the common single-operator session stops ending in a state the tool reports as wrong.
**Promote when:** it happens a third time, or a second operator hits it.

## A doc with no `id:` is invisible to every command that takes one

Nothing checks for a missing `id:`. `id-mismatch` needs both halves present, so a doc whose
frontmatter has no id at all is reported by nothing, appears on the board as `????`, and
`kit claim` cannot be pointed at it — the id is the argument. Found while fixing the fix
line `done-not-shipped` printed for one; that line now names the edit, but the doc is still
unclaimable and nothing says so.

0067 made it worse in one place: `no-status` builds its fix from the doc's id, so a doc with
neither prints `run: kit reset  "<why>"` — a command that cannot be typed. `unfilled-template`
already carries the branch for an id-less doc that `no-status` does not.

The fix is a check, at the severity `id-mismatch` already carries: the same guarantee is
broken, and by the same amount.

**Value:** a doc nobody can claim says so, instead of sitting on the board being skipped.
**Promote when:** a second doc reaches this state, or an agent is seen unable to claim one.

## A scaffolded project's CI has the flaws 0053 fixed here

`templates/stacks/go/check.yml` is where the kit's own workflow came from, so it carries
both: `golangci-lint-action` at `version: latest`, and no `timeout-minutes` under a
six-hour default. Every project `kit init --stack go` has written holds a copy, and
nothing updates a workflow after the first run.

Pinning in a template means the template names a version that ages, which is the reason
this is not simply the same two-line fix.

**Value:** a scaffolded project stops going red on a release nobody made, and a hung job
stops costing six hours.
**Promote when:** a scaffolded project's CI turns red with no commit behind it, or a job
there hangs.

## A comment citing a rule by number misdirects the next reader

`TestNothingUserFacingCitesARuleByNumber` reads string literals only, and says why: a
comment misdirects nobody but the next reader of the source. Two went wrong anyway —
`release.go` cited rule 8 for the stopping rule and `audit.go` rule 10 for the secret one,
both left behind when 0048 renumbered the protocol, and both found by reading rather than
by the build. Widening the test to comments is one line; the cost is that a note genuinely
about the numbered list can no longer name a number.

Two more are wrong now: `audit.go` and `conformance_audit_test.go` both cite rule 21, and the
protocol ends at 20. 0067 added three citations of rule 6, one of them in `docs/TECH.md` —
which is prose the guard cannot reach either way, because it reads Go string literals.

**Value:** the reasoning in the source stops pointing at rules that have moved.
**Promote when:** fired — two citations name a rule that does not exist.

## `tag-stale` reads this machine's tags, and `@latest` reads the remote's

`TagDistance` describes against the local refs, so a tag cut and never pushed silences the
check while an install still resolves to the one before it. It is `shipped-not-pushed` for
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

## `kit new`'s refusals are named nowhere

`refusals` in `cmd/kit/refusals_test.go` holds `ship`, `release`, `claim` and, since 0067,
`reset` — so the list 0055 made whole still does not cover `new`. It turns work away for six
reasons — an empty title, a title that slugs to nothing, a project not set up, several
remotes and no origin, a push
that answered nothing, and an id range the remote has exhausted — and `kit new --help` names
none of them. The last two arrived with 0058.

**Value:** an agent that meets a `kit new` refusal is not told the command could not have
made it.
**Promote when:** fired — `reset` was the fourth command to join the list.

## `next` prints a claim a checkout ahead of the trunk cannot make

Since 0059 `kit claim` refuses when HEAD carries commits the trunk does not have, and
`next` does not ask that question. Standing in the worktree that claims 0001, with 0002
free, the board prints `kit claim 0002` under **free, but not in this checkout**, and
running it there exits 2. The prose beside it already says to take the work in a worktree
of its own and never here, so the command is addressed elsewhere — which is why this is a
line rather than a box: the board is literal, not wrong.

The fix is the shape 0059 gave the trunk question: one condition read by `Survey` and by
`ClaimFeature`, and the worktree step printed ahead of the claim in that state.

**Value:** every command the board prints runs where it is printed.
**Promote when:** an agent is seen running it there, or a third claim refusal the board
cannot see is added.

## Two protocol rules restate a refusal `kit claim` now makes

0062 shipped while 0061 was being built: `ClaimFeature` refuses a checkout that already
claims something, and `kit new` refuses to draft on a branch citing an id that is not this
checkout's claim. 0061 kept the two rules because the refusal did not exist yet and cut
seven others on exactly that test; 0062's own `## Design` says the two collapse into it.
Four lines read at the start of every session in every project, saying what the command now
says at the moment it matters. They were rules 11 and 12 when this was written and are 9 and
10 now, which is the entry above happening to this one.

**Value:** the protocol keeps only what no refusal can make.
**Promote when:** fired — 0067 rewrote rule 6, and these two were not looked at.

## Nothing tells an agent which language a project's docs are written in

Kit generates no prose: the templates and what you typed are its only sources. The language
of a doc body is therefore the agent's choice, remade each session. A project that wants
Russian gets English unless its own house rules say otherwise, and that sentence is written
by hand in every project that wants one.

`kit init --lang ru` would seed it above the marker in `AGENTS.md`, which `init` never
rewrites and the project owns afterwards — any language, with no translated templates and
no model in the binary. The seeded line has to name what stays English whatever the
language: the headings the parsers read, the frontmatter keys, and `Draft | Shipped |
Dropped`. Without that an agent translates `## Tasks` and the board sees no tasks at all. A
re-run under a different language would not update the line, and `kit new` still takes an
ASCII title — `Slug` strips to `[a-z0-9]`.

**Value:** a project says what language its docs are written in once, and every agent is told.
**Promote when:** a second project needs non-English docs, or an agent writes English into
one that asked for Russian.

## `claim`'s two refusals still send an agent to a sibling directory

0063 moved a claim's worktree to `.worktrees/<branch>` inside the project, but the two
refusals in `ClaimFeature` that tell an agent to stand somewhere else still print
`git worktree add --detach ../<repo>-trunk <trunk>`. It is a detached checkout rather than
a claim, so it has no branch to be named for, and the sibling path is the layout 0063
argued against.

**Value:** one layout for every checkout the kit tells an agent to make.
**Promote when:** an operator's parent directory collects trunk checkouts, or a third
message needs a path and there is still no one answer to give it.

## `docs/features/TEMPLATE.md` is a copy nothing reads

`kit init` stopped writing it and a conformance case pins that it must not, `notAFeature`
keeps it off the board, and `kit help template` prints the embedded copy. So the file is
read by nothing and has already drifted: it lost a paragraph two rounds before 0065
brought its frontmatter back in line. What it still does is sit in the features directory
looking like the template, where the next person to copy from it gets whatever it has
drifted to.

It drifted again in 0068, and into the worst version of itself: the copy still carries the
`needs:` guidance that round deleted for pre-answering its own question, so a doc copied from
it is handed the advice that cost a chain a session.

**Value:** one template, so a doc copied by hand is the doc `kit new` writes.
**Promote when:** fired — the drift is the misleading sentence 0068 removed.

## Nothing notices an agent merging its own work

`kit claim` says a project with no remote is one no other agent can see, and `ship` says
there is nowhere to hand the work off and that merging is not yours. An unattended chain
read both and merged four of its own feature branches into the trunk anyway, because with no
remote there is no pull request to stop at and no reviewer to wait for. `ship` knows the
claim branch name and could say when the trunk already contains it, which reports the
breach after the fact rather than preventing it.

**Value:** an operator can tell whether a branch was reviewed or merged by the agent that
wrote it.
**Promote when:** a project with a remote is found doing it, or the report is wanted for a
chain the operator intends to run unattended.
