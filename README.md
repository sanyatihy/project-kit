# project-kit

Bootstrap projects and run them with agents:

```sh
kit init     # set a project up
kit new      # start a feature doc
kit next     # what should I work on?
kit audit    # where do the docs disagree with the repo?
```

Plans live in markdown in git. No database, no queue, nothing to keep in sync — the
feature docs and the git branches are the state. A single Go binary; `git` is the only
runtime dependency.

## Install

From a clone: `make install`. Nothing below applies.

Over the network it takes two settings, because the repo is private.

**1. Keep Go off the proxy and the checksum database for it.** Neither can see a private
repo.

```sh
go env -w GOPRIVATE='github.com/sanyatihy/*'
```

`GOPRIVATE` is a comma-separated list, so append to a company value rather than replacing
it. A shell `export GOPRIVATE=` beats `go env -w`, and `GONOPROXY`/`GONOSUMDB` override it
entirely. Run `go env GOPRIVATE` and confirm before blaming anything else.

**2. Give `git` a way to authenticate.** Go fetches over HTTPS, which needs credentials
for a private repo. Reuse the SSH key you have, scoped to the one namespace:

```sh
git config --global url."git@github.com:sanyatihy/".insteadOf "https://github.com/sanyatihy/"
```

Then `go install github.com/sanyatihy/project-kit/cmd/kit@latest`. Upgrading is the same
command.

The failure modes are distinguishable:

| symptom | missing |
| --- | --- |
| `410 Gone` from `proxy.golang.org` | setting 1 |
| a `sum.golang.org` `404` | setting 1, the checksum half |
| a git prompt, or `terminal prompts disabled` | setting 2 |

## Setup

```sh
kit init /path/to/project              # docs only
kit init /path/to/project --stack go   # also Makefile, go.mod, cmd/, CI
kit init /path/to/project --forge gitlab   # when nothing says which
```

Run it from anywhere — `kit` is on your `$PATH`. Prompting an agent inside the kit's own
repo to set up some other project gives it write access to the tool it is using.

It creates the directory if missing, makes it a git repo, and writes `AGENTS.md`,
`docs/VISION.md`, `docs/TECH.md`, `docs/features/BACKLOG.md` and a
`.gitignore` covering what an editor and an operating system leave behind. It overwrites
nothing you wrote, so it is safe on a populated directory and safe to re-run.

With `--stack go` it also writes `Makefile`, `go.mod`, `cmd/<name>/`, `internal/`, a
`README.md`, and one CI pipeline running `make check` — and a `.gitignore` for Go in
place of the default rather than alongside it.

Which CI file depends on where the project is hosted, from the first thing that says so:

| | evidence |
|---|---|
| 1 | a `.gitlab-ci.yml` or `.github/workflows/` the project already has |
| 2 | the remote's host — `github.com`, or anything containing `gitlab` |
| 3 | the directory layout, when it names a forge: `~/code/gitlab.example.com/team/thing` |
| 4 | nothing — **no CI file is written**, and the run says so |

A hostname cannot tell a self-hosted GitLab from a Gitea, so a file the project already
carries outranks it. `--forge github|gitlab` settles it outright.

Re-running is how a project catches up after the kit is upgraded. One kit-owned thing is
refreshed: the `## Work protocol` section of `AGENTS.md`. Every document is yours the
moment it is placed and is never touched again; house rules live above the marker comment
in `AGENTS.md`. Nothing but `kit init` creates a project file, so there is one answer to
where a file came from, and `kit help vision`, `template` and `as-built` print the shape each
is waiting for.

Once set up you never run anything yourself. `AGENTS.md` tells the agent to run
`kit next` at the start of a session, so "what should I work on?" is enough.

## Compatibility and upgrades

Read `kit help compatibility` before upgrading a project with active work. The
[compatibility contract](templates/compatibility.md) covers supported platforms, stable
interfaces, legacy commands and coordinated upgrades across agent machines.
[Publishing kit](docs/RELEASE.md) gives the post-merge release and installation checks.

## Developing a project

There are two human review points for planned work: the plan and the implementation.
The agent handles the commands between them. One feature is one pull request; each
Tasks box is one implementing commit within it.

A feature makes the project observably better. Adding a capability, removing an expensive
one, closing a security exposure, eliminating a recurring failure and meeting a latency
or cost target all qualify. State the benefit and how to verify it.

Prefer one substantial, coherent outcome over a separate feature for every small step.
For example, retiring a legacy import format can include migrating callers, deleting the
parser, tests and documentation in one feature. Those are task boxes, not four features.
Split when outcomes stand on their own, approvals differ, or the combined change cannot
be reviewed safely. Parallel agents are a reason to find independent outcomes, not to
fragment a coherent one. Small directly requested changes use `start --work` and receive a harvested feature
record before review. Removing integrated work gets a new doc citing its frozen history.


| Step | Human | Agent |
| --- | --- | --- |
| Decide | Discuss the outcome and settle product choices | Record the vision and unresolved questions |
| Plan | Ask for a complete feature round | Run `kit start --plan "<round>"`, use `kit new`, fill designs, tasks and `needs:`, then `kit finish` and open one planning PR |
| Approve the plan | Review and merge the planning PR | Address plan feedback on that branch |
| Build | Let agents proceed | Update a trunk checkout, run `kit audit` and `kit next`, claim one feature, work in its worktree through every task |
| Submit | Review the implementation PR | Run `kit finish -- <gate argv>`, open the PR, then continue independent approved work from updated trunk |
| Revise | Request changes if needed | On the same feature branch, `kit reopen <id> "<why>"`, revise tasks, implement, validate, submit and push again |
| Integrate | Merge the implementation PR | Return to a trunk checkout, `git pull --ff-only`, then `kit audit` and `kit next` |

For a new repository, initialize it with `kit init`. Run `kit start --plan "<round>"`
or `kit start --work "<outcome>"`; it publishes an empty trunk and preserves all files
for the first review branch. Add a shared remote before parallel drafting so ids and claims can be reserved.
With no remote, PR review remains pending and independent clones cannot coordinate ids or claims.

Claims must start from the current published trunk. Kit refuses a stale checkout and
prints the repair; it does not pull into a working tree another agent may be using.
Each agent owns a separate checkout. Independent features can proceed together without
a global agent limit. `needs:` waits for merged prerequisites present in the checkout;
shared filenames alone do not require serial execution.

A task discovered during implementation becomes another box, feature or backlog entry.
The agent continues through the current feature until it is ready for review or blocked.
A human decision goes to `kit block <id> "<question>"`; the agent may take independent
work from another checkout. If none is available, it reports the blocker and stops.
When the human answers, record the decision in the plan and tick the question's box in
the same commit. Bring that commit into the paused checkout before resuming.

## Submission and integration

`kit submit` submits a completion snapshot; it does not approve, merge, deploy or certify
tests. It records `status: Submitted` and a `submitted:` date.
The branch's relationship to trunk determines the handoff:

| State | Next action |
| --- | --- |
| Parked | Wait for the operator to request activation; agent opens a planning PR |
| Draft with unfinished tasks | Implement the next box |
| All tasks complete | Gate, audit, submit |
| Completion stamp still local | Push and open the implementation PR |
| Submission pushed, unmerged | Human reviews; agent uses reopen for requested changes |
| Completion snapshot on trunk | Integrated and frozen; update trunk before taking more work |

`kit next` prints that distinction, including a retained branch after a squash merge.
An unmerged submission can be reopened and revised; an integrated doc is frozen history.
Later product changes use a new feature citing it. No post-merge status commit is needed.

Legacy `Shipped`/`shipped:` documents remain readable and frozen after integration.
`ship` and `reset` remain aliases for `submit` and `reopen`, with the same guards.
Unknown or missing statuses require `reopen --repair`; Dropped plans need a new approved
feature to return. `release --discard` permits deleting unmerged committed work from
this clone's claim, and always preserves uncommitted files and foreign claims.
`drop --reservation` explicitly releases an unused id; a missing doc alone deletes nothing.

To retain a complete plan for later, ask the agent to park it. On a `plan/<round>`
branch, `kit park <id> "<reason>"` preserves the plan as Parked; `kit activate <id>`
returns it to Draft when requested. Both changes require human review through merge.
Parking refuses ongoing claims. Use `block` for unanswered decisions, `release` for
untouched claims, and an unfinished PR to preserve ongoing work. `reopen --repair`
cannot activate parked work. Upgrade all writers before using Parked.

## Exploring first

For an MVP or experiment the human asks to explore, record the intent first and work
on the spike branch it makes. This route does not bypass approval of an already drafted
feature. When the result is worth keeping, finish it:

```sh
kit start --work "A long URL comes back short"
# ...implement and commit...
kit finish -- make check
```

Finish harvests the work into a feature doc on a branch named after that feature, asks
for Problem and Design, then gates, submits and pushes. A spike made without
`kit start --work` is recorded with `kit harvest "<title>" --box "<outcome>"` first. Fill Problem and Design from the decisions actually made; the completed
outcomes are already ticked. Review and revision then follow planned work. Do not create retrospective plans pretending to precede the code.

With no prior commits, harvest creates an empty trunk base and commits the entire MVP
on the feature branch. Publish the empty base before opening the first PR. Staged product
files stay off trunk, so the reviewer can see the entire implementation diff.

If the spike is discarded, `kit harvest --nothing "<what it showed>"` records the lesson
without creating a feature. Promote follow-up backlog entries on a later planning branch.

Kit can enforce its own commands. Required CI checks and remote merge permissions must
protect trunk against raw git bypasses. Use distinct agent identities and reserve final
approval for the human; an agent using the human's credentials has the same permissions.

## Commands

```sh
kit start <id> | --plan <round> | --work <outcome>  prepare recorded work
kit finish [--json] [-- <gate argv>]     validate, submit and publish for human review
kit new   <title...>                     next free ID, date filled in, status Draft
kit claim <id>                           the claim step of start; exit 1: not yours
kit harvest <title...> --box "..."       give an unrecorded spike a doc and a commit
kit pause "<why>"                        set work down, keeping the claim
kit release <id> [--discard]               give a claim back; refuses over work, and over
                                         a claim this clone never created
kit block <id> "<question>"              record a decision only a human can make
kit submit  <id>                           verify every box, stamp the date, submit
kit reopen <id> "<why>" [--repair]       reopen review; explicit invalid-status repair
kit park <id> "<why>"                   retain a complete plan for later
kit activate <id>                       propose activation through planning review
kit drop  <id> "<why>" [--reservation]   retire a plan; explicit unused-id cleanup
kit next  [--json] [--offline]           the next action for this checkout
kit audit [--json] [--strict] [--offline]  --strict exits 1 on HIGH, for CI
                                         both exit 2 where there is no docs/features
kit init  [PATH] [--stack S] [--forge F]
kit help  [protocol|template|checks|vision|as-built|go-checklist|python-checklist|
           compatibility|backlog]
kit version
```

The binary documents itself: `kit help` explains the model and the four rules, and
`kit help checks` lists every finding from the same registry `audit` reports from. An
agent never needs this repository to use the tool. `kit <command> --help` describes the
command, not just its flags. `kit help <command>` prints its usage and recovery guidance.

`kit help go-checklist` is standing engineering guidance rather than documentation of the
kit — embedded rather than written into projects, because `audit` cannot check a word of
it and a copy per repo would be drift nothing could detect.

`next` and `audit` read only and never write, and each reads the repo you are in, from any
subdirectory. For several projects, loop:

```sh
for d in ~/code/*; do (cd "$d" && kit next); done
```

Exit codes are a contract, so a script can tell a typo from a finding:

| Code | Means |
| --- | --- |
| `0` | it worked |
| `1` | it ran and the answer is no: the feature is not yours (lost, or refused over its plan), or `audit --strict` found HIGH |
| `2` | the command could not run, or refused: unknown command, bad flag, a refusal |

Diagnostics go to stderr and data to stdout, so `kit next --json` stays pipe-clean even
when something goes wrong.

`kit help checks` prints every finding `audit` can report, generated from the registry, so
it cannot drift. Thresholds are constants in `internal/kit`.

## The feature doc

One file per feature in `docs/features/`, named `NNNN-name.md`. The directory listing is
the board. Frontmatter carries `id`, `title`, `status` (`Parked ↔ Draft → Submitted`, exit
`Dropped`), `created`, `submitted` and `needs`. Sections: `Problem`, `Non-goals`, `Design`,
`Tasks`, `Open questions` (optional), `Log`.

Only `kit submit`, `kit drop`, `kit reopen`, `kit park` and `kit activate` write status;
`kit submit` writes the submission date. There is no status for work in flight — the claim branch says that, and a second
copy in the doc could only disagree with it.

`needs:` lists the feature ids whose implementation must merge before this one can
start. Empty means no prerequisite, so independent features can proceed together once
their plans are merged and other blockers are resolved. `kit help template` explains it.

Two things keep a doc honest. Every fact has one home: still true after the feature ships
and it belongs in `docs/TECH.md`, one line; a change to this doc's plan and it belongs in
`## Log`, one entry naming the change; neither and it belongs in the pull request. And
blockers belong in `## Open questions`, not in prose — a task box that says "blocked on X"
is invisible to `kit next` and will be handed to an agent that cannot do it.

## Several agents

One worktree each, one feature each. The worktree is what keeps them apart; two agents in
one checkout overwrite each other. Merging is serial. Two or three at once is the
realistic ceiling — the limit is your review bandwidth.

## Layout

```
cmd/kit/          the CLI
internal/kit/     parsing, repo model, audit, next, scaffolding
templates/        what `kit init` writes, embedded into the binary
```

## Changing the kit

```sh
make check     # fmt-check, vet, lint, unit tests, conformance suite
make install   # after anything lands on main
```

`make install` is not optional housekeeping. `kit` on your `$PATH` is one binary shared by
every project, so a merged change reaches none of them until you reinstall.

The conformance suite is `cmd/kit/conformance*_test.go`. It builds the binary once and
drives it against throwaway projects, so the commands are covered as a user runs them. It
is a Go test in-tree on purpose: the suite it replaced was an external script outside the
build, which is how it came to be deleted without anything noticing.

If you add a check to `audit`, add it to the registry, add a test that it fires and a test
that it stays quiet. The first is enforced — `TestEveryRegisteredCheckIsExercised` fails
the build for a check with no case.
