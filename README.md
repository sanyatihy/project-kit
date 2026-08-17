# project-kit

Bootstrap projects and run them with agents:

```sh
kit init     # set a project up
kit new      # start a feature doc
kit next     # what should I work on?
kit audit    # where do the docs disagree with the repo?
```

Plans live in markdown in git. No database, no queue, nothing to keep in sync — the
feature docs and the git branches are the state. A single Go binary, no runtime
dependencies.

## Install

From a clone, nothing special: `make install`. The source is already there, so none
of the below applies.

Installing over the network takes two settings, because the repo is private.

**1. Tell Go not to use the proxy or the checksum database for it.** Neither can
see a private repo.

```sh
go env -w GOPRIVATE='github.com/sanyatihy/*'
```

`GOPRIVATE` is a comma-separated list, and if you already have one — many company
setups do — this **appends**, it does not replace:

```sh
export GOPRIVATE="go.example.com,github.com/sanyatihy/*"
```

Two ways that write can silently do nothing. An `export GOPRIVATE=` in your shell
profile overrides `go env -w` completely; Go warns about the conflict, and the fix
is to edit the profile line instead. And `GOPRIVATE` is ignored outright if
`GONOPROXY` or `GONOSUMDB` is set, since those are what it is shorthand for. Run
`go env GOPRIVATE` and confirm you see what you expect before blaming anything else.

**2. Give `git` a way to authenticate to GitHub.** Go always fetches over HTTPS,
which needs credentials for a private repo. Reuse the SSH key you already have:

```sh
git config --global url."git@github.com:sanyatihy/".insteadOf "https://github.com/sanyatihy/"
```

Scoped to the one namespace on purpose — the unscoped form rewrites every
`github.com` URL on the machine, which is a large side effect for one module. If
you would rather not rewrite URLs at all, a `~/.netrc` entry or a credential helper
holding a token with `repo` scope works instead, at the cost of a secret to rotate.

Then:

```sh
go install github.com/sanyatihy/project-kit/cmd/kit@latest
```

Upgrading is the same command again.

The failure modes are distinguishable. `410 Gone` from `proxy.golang.org` means
setting 1 is missing. A `sum.golang.org` `404` — often quoting a `could not read
Username` from *its* server, not yours — also means setting 1, specifically the
checksum half. A local git prompt or `terminal prompts disabled` means setting 1
worked and setting 2 is what is missing.

## Setup

```sh
kit init /path/to/project              # docs only
kit init /path/to/project --stack go   # also Makefile, go.mod, cmd/, CI
kit init /path/to/project --stack go --forge gitlab   # when nothing says which
```

Run this from anywhere — `kit` is on your `$PATH`, so you never need to be inside
the kit's own repo. Prompting an agent there to set up some other project gives it
write access to the tool it is using, and the kit's `AGENTS.md` will be telling it
to work on the kit's own plan. Bootstrap from the new project's parent directory, or
from wherever you already are.

Creates the directory if it does not exist, makes it a git repo if it is not one,
and writes `docs/VISION.md`, `docs/features/{README,TEMPLATE}.md`, `docs/INBOX.md`,
and `AGENTS.md`.
It never overwrites or deletes, so it is safe on a directory that already has files.

With `--stack go` it also scaffolds the code layout: `Makefile`, `.gitignore`,
`go.mod` (module path inferred from where the project sits on disk), `cmd/<name>/`,
`internal/`, a `README.md`, and one CI pipeline running `make check`.

Which CI file that is depends on where the project is hosted, and the kit works it
out from the first thing that actually says so:

| | evidence |
|---|---|
| 1 | a `.gitlab-ci.yml` or `.github/workflows/` the project already has |
| 2 | the remote's host — `github.com`, or anything containing `gitlab` |
| 3 | the directory layout, when it names a forge: `~/code/gitlab.example.com/team/thing` |
| 4 | nothing — **no CI file is written** |

A hostname cannot tell a self-hosted GitLab from a Gitea, so the file the project
already carries outranks it, and a host the kit does not recognise falls through
rather than defaulting. Rung 3 is the only one that answers on a fresh `git init`
with no remote yet.

When nothing says, the run tells you so and names both ways to fix it: add a remote
and re-run, or pass `--forge github|gitlab`, which settles it outright. A missing
pipeline is visible; one for the wrong forge is not.

Runner tags and the build image in a generated `.gitlab-ci.yml` are left for you —
they differ between one GitLab instance and the next, and the file is never
overwritten once it exists.

That pipeline does not run `kit audit --strict`. Doing so would mean installing the
kit in CI, and since the kit is a private module that means a credential in every
project you scaffold — a secret to store and rotate per repo, and a first push that
fails until someone sets it. Drift is caught locally instead, where `kit next` is
already being run. Scaffolding stays zero-setup, which matters more than catching
drift a few hours earlier.

So every project answers the same three questions with the same three commands:
`kit next` (what to work on), `kit audit` (what drifted), `make check` (is it
green). That uniformity is the point — an agent's first five minutes are identical
in every repo.

Deployment scaffolding — terraform, ansible, images, dashboards — is deliberately
**not** generated. It is the fastest-rotting part and it is not needed to write
code. Add it when the project actually has something to deploy.

Re-running is safe, and it is how a project catches up after the kit is upgraded. Two
things are refreshed: the `## Work protocol` section of `AGENTS.md`, and the reference
docs the kit owns outright — `docs/features/TEMPLATE.md` and `docs/features/README.md`.
Nothing reads a project's copy of those, so replacing them takes nothing away and stops
them describing a format the tool no longer produces. Everything you fill in —
`VISION.md`, `TECH.md`, `INBOX.md`, `BACKLOG.md`, and every stack file — is yours and is
never touched. House rules live above the marker in `AGENTS.md`, which is also kept. No `TECH.md` — that describes what exists, so it waits until something
ships.

Once set up you never need to run anything yourself. `AGENTS.md` tells the agent to
run `kit next` at the start of a session, so "what should I work on?" is enough.

## Starting from nothing

Say the job is "build a URL shortener" and no code exists yet.

```sh
kit init ~/code/github.com/you/url-shortener --stack go
cd ~/code/github.com/you/url-shortener
kit next                         # nothing open — start a feature
kit new "Shorten URLs"           # writes docs/features/0001-shorten-urls.md
```

Fill in the doc. It stays `Draft` while you are still deciding:

- **Problem** — one or two sentences. "People need short links for long URLs."
- **Non-goals** — where scope goes to die. *Custom domains, click analytics, user
  accounts.* Writing these down is what stops the task being unbounded.
- **Design** — how it works. `POST /` stores a URL and returns a code; `GET /<code>`
  redirects. Postgres, base62 of the row id.
- **Open questions** — anything you cannot answer yet. *Do codes expire?* Leave the
  doc in `Draft` until these are settled, or move them to `docs/INBOX.md` and ask.
- **Tasks** — the part that matters, in the order they should be done:

```markdown
- [ ] Schema + migration for `links(id, url, created_at)`
- [ ] POST / — validate the URL, insert, return base62 code
- [ ] GET /<code> — look up and 301, 404 when missing
- [ ] Rate limit POST by IP
```

Each box is one PR you can finish and verify without asking anyone. `- [ ] Build the
backend` is not a task. If you cannot say how you would check a box is done, it is
too big — split it.

Write **as much as necessary and no more**. Cut restatement, hedging and padding;
never cut a decision, a constraint, or the evidence behind it. Both failures cost
real time: a doc nobody reads spends the context window every later session needs,
and one too thin to act on costs a session to reconstruct. Every template section
states its own budget, and the kit holds its own shipped text to line ceilings in
`cmd/kit/budget_test.go` — the protocol most tightly, since it is re-read at the
start of every session in every project.

A drafting session is one **plan round**: every doc it produces goes on a single branch
named `plan/<something>` — never after a feature, since a branch leading with a
four-digit ID *is* a claim — and merges once. One review for the round rather than one
per draft, and the docs reach the default branch at whatever statuses they reached,
`Draft` included.

Then accept whichever you have looked at. This is the one command an agent must never
run, but it is a **signal, not a gate**:

```sh
kit accept 0001                  # Draft -> Accepted
```

Accepting says you have blessed the design, which moves the feature up the board. It
does not unlock the work: agents claim Drafts too, and `kit next` simply offers anything
`Accepted` first. It refuses if `## Open questions` is still there or a `BLOCKING` inbox
line names the feature, because both say the design is not settled.

What still refuses a claim is a `BLOCKING` inbox line, and a doc nobody wrote — an empty
`## Problem`, `## Design` or `## Tasks` is turned away whatever its status, so a doc
straight out of `kit new` can never be handed to an agent to build.

Land the round, and `kit next` hands you the first box and the command to start:

```
  start here
    0001  0/4   Shorten URLs
    → Schema + migration for links(id, url, created_at)
       docs/features/0001-shorten-urls.md
       kit claim 0001
```

`kit claim` pushes the branch that takes the feature and prints the worktree to work in.
It refuses until the plan is on the default branch, for two reasons: the claim pushes
`HEAD`, so an uncommitted doc would not travel with the branch you were sent to, and a
doc that is only on the plan branch makes the claim a child of that branch — leaving a
merge into the round's own branch as the only way to integrate the work, performed by
whoever coordinated the round. Land the round, then claim. A project with no trunk to
land on — one branch, no remote — is unaffected.

Every other refusal `claim` makes is one `kit next` already makes — an unmet `needs:`, a
BLOCKING inbox line, a doc nobody wrote. Claiming by id without reading the board
cannot get you further than the board would. `kit ship` refuses the same way, and
also refuses a feature claimed by a branch that is not your checkout's: `Shipped` is
history nobody may edit, so shipping someone else's claim freezes their work.

Do that one box, tick it in the commit that implements it, and open a PR. Run `kit next`
again for the next one. When the last box is ticked, `kit ship 0001` verifies every box,
stamps the date, and freezes the doc.

Then push and stop, because **you never merge your own work**. `kit next` closes with
what is waiting on the other side of that handoff:

```
  awaiting review
    0001  Shorten URLs
          0001-shorten-urls · waiting 2 days · needs a human · you built this — not
          yours to merge
```

Shipped on the branch, pushed, and not on the trunk. The status is read from the branch,
since on the trunk it still says what it said when the round landed — so a coordinator
standing on the trunk sees the whole queue in the command everyone else already runs.
`shipped-not-pushed` catches the other half at HIGH: shipped, and on no remote at all.

**Who may merge it is two decisions, at two granularities.** Branch protection on the
trunk is the floor — the only thing an agent cannot route around — and its granularity is
the whole project: protect the trunk and a log-field rename waits for a person, leave it
open and the change to billing merges unseen. So
within whatever it allows, one feature at a time can be marked as a person's:

```sh
kit accept 0001 --review         # writes review: human into the doc
```

A marked feature shows as **needs a human** on the board above, and a coordinator agent
leaves it alone. It is set by whoever accepts the round — the moment the whole set is in
front of them, before anyone has sunk work into the outcome — and never by an agent.
Unmarked, and not your own, means a coordinator may merge it.

What the kit does here is make the invariant legible, and nothing more. An agent can
merge its own branch with git directly; what stops it is branch protection on the trunk,
which is the one control the controlled party cannot rewrite. Agents also need git
identities of their own — one pushing with the operator's credential is the operator to
every check that exists.

**When you get stuck on something only a human can decide**, add a line to
`docs/INBOX.md` with your leaning and move on:

```
- [ ] 2026-08-11 · 0001 · BLOCKING · Should codes expire? Leaning: no, keep forever.
```

That marks 0001 blocked, so `kit next` stops offering it and points you elsewhere.

## The four rules

1. `In progress`, then `Accepted`, then `Draft`, and lowest ID first within each.
   Finishing beats starting, and blessed beats unblessed. `status:` orders the
   board; it does not gate it.
2. Within a feature, do the topmost unticked box. List order is the order of work.
3. One agent per feature, one feature per checkout. `kit claim <id>` pushes the
   branch named after the doc, and creating that ref on the remote is what decides
   — exit 1 means another agent won it. Local branches, remote branches and
   worktrees all count as claims. Work in the worktree it prints, not where you
   stand.
4. A feature named in an open `INBOX.md` question is blocked. "Named" means the id
   field of the line, which is why `kit block` writes it for you. So is one whose
   `needs:` has not shipped — and what is left ready is how many agents can work at
   once.

## Commands

```sh
kit new   <title...>                     next free ID, date filled in, status Draft
kit accept <id>                          bless a Draft — the human's call, never an agent's
kit claim <id>                           push the branch; exit 1 means you lost it
kit block <id> "<question>" [--fyi]      park a decision only a human can make
kit ship  <id>                           verify every box, stamp the date, freeze
kit drop  <id> "<why>"                   retire a feature that will not be built
kit next  [PATH ...] [--json] [--offline] warns above 2 features in progress
kit audit [PATH ...] [--json] [--strict] [--offline]
                                         --strict exits 1 on HIGH, for CI
kit help  [protocol|template|conventions|checks|inbox|as-built|go-checklist|deployment]
kit version
```

The binary documents itself: `kit help` explains the model and the four rules,
`kit help checks` lists every finding from the same registry `audit` reports from,
and `kit help protocol|template|conventions` prints the embedded documents. An
agent never needs this repository to use the tool. `kit <command> --help` describes
the command, not just its flags.

Two topics are standing engineering guidance rather than documentation of the kit:
`kit help go-checklist` is a tiered checklist for Go projects, and
`kit help deployment` holds the patterns and traps behind taking a project to
production. Both are embedded rather than written into projects — `audit` cannot
check either, so a copy per repo would be drift nothing could detect. Neither
contains a `- [ ]`: in a kit project that syntax means work to do, and guidance that
looked like a task list would be worked through.

`next` and `audit` read only and never write. With no path they use the repo you are in, from any
subdirectory; `kit next ~/code/*` surveys every project at once. A path in that list
that cannot be read is reported in place and sets the exit code to 2, and every other
project still reports — one tidied directory must not blank the board.

Exit codes are a contract, so a script can tell a typo from a finding:

| Code | Means |
| --- | --- |
| `0` | it worked |
| `1` | `kit audit --strict` found something HIGH — only `--strict` ever exits 1 |
| `2` | the command could not run: unknown command, bad flag, missing path |

Diagnostics go to stderr and data to stdout, so `kit next --json` stays pipe-clean
even when something goes wrong.

`audit` names the file, the problem, and the fix. It reports:

- **HIGH** — `secret-in-doc`, `needs-cycle`, `unfilled-template`, `stalled`,
  `unticked-shipped`, `shipped-not-pushed`, `inbox-stale`, `unchecked`, `id-mismatch`
- **MEDIUM** — `no-plan`, `done-not-shipped`, `tech-drift`, `shipped-no-date`,
  `unfilled-vision`,
  `missing-section`, `inbox-unparsed`, `inbox-unknown`,
  `needs-unknown`, `shipped-early`, `protocol-stale`, `no-tasks`,
  `no-claim`, `split-claim`, `orphan-claim`, `draft-claimed`, `no-status`,
  `bad-status`, `no-protocol`
- **LOW** — `dead-link`, `no-inbox`, `no-tech`, `no-vision`, `fyi-stale`

`kit help checks` prints this list from the same registry `audit` reports from, so
it cannot drift.

Thresholds are constants in `internal/kit`.

## The feature doc

One file per feature in `docs/features/`, named `NNNN-name.md`. The directory listing
is the board. Frontmatter carries `id`, `title`, `status`
(`Draft → Accepted → In progress → Shipped`, exits `Dropped`/`Superseded`), `created`,
`shipped`. Sections: `Problem`, `Non-goals`, `Design`, `Tasks`, `Open questions`
(optional), `Log`.

`needs:` lists the feature ids that must ship before this one can start, and is the
only field that changes what `kit next` offers. Empty means it can start now, which
is the common case — and several empty ones at once is what lets agents work in
parallel. Judge it by whether the work could *start*, not by what it touches: a
shared composition root, a shared router, or numbered files in one sequence such as
migrations all make a feature dependent however separate its boundary looks.
`related:` is unaffected by any of this — it stays reading material.

Two things that keep it honest: `## Log` holds dated conclusions, never debate — the
argument lives in the PR. And blockers belong in `INBOX.md`, not in prose; a box that
says "blocked on X" is invisible to `kit next` and will be handed to an agent that
cannot do it.

`check:` is optional. Set it, with a date, only when a feature is a bet worth
revisiting — a pricing change, a growth experiment. After that date `audit` reports
`unchecked` until you record the outcome in `## Log` and set `checked:`.

## Several agents

One worktree each, one feature each. The worktree is what keeps them apart; two
agents in one checkout overwrite each other. Merging is serial. Two or three at once
is the realistic ceiling — the limit is your review bandwidth.

## Layout

```
cmd/kit/          the CLI
internal/kit/     parsing, repo model, audit, next, scaffolding
templates/        what `kit init` writes, embedded into the binary
```

## Changing the kit

```sh
make check     # fmt-check, vet, unit tests, conformance suite
make install   # after anything lands on main
```

`make install` is not optional housekeeping. `kit` on your `$PATH` is one binary
shared by every project, so a merged change reaches none of them until you reinstall.
The first real project built with the kit ran against a binary two features behind —
it was scaffolded without a `VISION.md`, got a protocol missing the rule about
drafting from one, and never ran the secret check. Nothing looked wrong from inside
that project, because nothing was.

The conformance suite is `cmd/kit/conformance*_test.go`. It builds the binary once
and drives it against throwaway projects, so the commands are covered as a user runs
them rather than only through their parts. It is a Go test in-tree on purpose: the
suite it replaced was an external script outside the build, which is how it came to
be deleted without anything noticing.

If you add a check to `audit`, add a test that it fires, a test that it stays quiet
when it should, and document the code above. The first of those is enforced —
`TestEveryRegisteredCheckIsExercised` fails the build for a check with no case.
