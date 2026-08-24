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
`docs/VISION.md`, `docs/INBOX.md`, `docs/TECH.md`, `docs/features/BACKLOG.md` and a
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
where a file came from, and `kit help vision`, `inbox` and `as-built` print the shape each
is waiting for.

Once set up you never run anything yourself. `AGENTS.md` tells the agent to run
`kit next` at the start of a session, so "what should I work on?" is enough.

## Starting from nothing

Say the job is a URL shortener and no code exists.

```sh
kit init ~/code/github.com/you/url-shortener --stack go
cd ~/code/github.com/you/url-shortener
kit next                         # nothing planned — start a feature
kit new "Shorten URLs"           # writes docs/features/0001-shorten-urls.md
```

Fill the doc in:

- **Problem** — one or two sentences.
- **Non-goals** — where scope goes to die. *Custom domains, click analytics, accounts.*
- **Design** — the decisions. `POST /` stores a URL and returns a code; `GET /<code>`
  redirects. Postgres, base62 of the row id.
- **Open questions** — anything you cannot answer yet. *Do codes expire?* What only a
  human can settle goes to `docs/INBOX.md` via `kit block`.
- **Tasks** — the part that matters, in the order they should be done:

```markdown
- [ ] Schema + migration for `links(id, url, created_at)`
- [ ] POST / — validate the URL, insert, return base62 code
- [ ] GET /<code> — look up and 301, 404 when missing
```

Each box is one PR you can finish and verify without asking anyone. `- [ ] Build the
backend` is not a task. If you cannot say how you would check a box is done, split it.

A drafting session is one **plan round**: every doc it produces goes on a single branch
named `plan/<something>` — never one citing a feature id, since a branch that names one
*is* a claim — and merges once.

Land the round, and `kit next` hands you the first box and the command to start:

```
  start here
    0001  0/3   Shorten URLs
    → Schema + migration for links(id, url, created_at)
       docs/features/0001-shorten-urls.md
       kit claim 0001
```

`kit claim` pushes the branch that takes the feature and prints the worktree to work in.
Creating that ref on the remote is a compare-and-swap, so exactly one agent wins; exit 1
means you lost and should take something else.

While an agent works, the board reads its progress off the claim branch — the boxes are
ticked there, and the trunk's copy of the doc would say `0/3` however far along it is:

```
  taken
    0002  1/3   Resolve a short link — 0002-resolve-a-short-link, 20 minutes ago
```

Do one box, tick it in the commit that implements it, and open a PR. Run `kit next` again
for the next one. When the last box is ticked, `kit ship 0001` verifies every box, stamps
the date and freezes the doc.

Then push and stop, because **you never merge your own work**. `kit next` closes with
what is waiting on the other side of that handoff:

```
  awaiting review
    0001  Shorten URLs
          0001-shorten-urls · waiting 2 days · you built this — not yours to merge
```

Shipped on the branch, pushed, and not on the trunk. The status is read from the branch,
since on the trunk it still says what it said when the round landed — so a coordinator
standing on the trunk sees the whole queue in the command everyone else already runs.

Both that and `shipped-not-pushed` ask the remote which branches exist, bounded at 15
seconds, and **both say so when it does not answer** rather than printing a page with
nothing on it. An unreachable remote and a project with nothing waiting look identical
otherwise, and the board is read as good news. `--offline` is an instruction rather than
a failure, so `next` stays quiet about it; `audit` names the checks it dropped, since a
skipped check and a passing one print the same page.

Branch protection on the trunk is the only thing an agent cannot route around. What the
kit does is make the invariant legible; agents also need git identities of their own,
since one pushing with the operator's credential *is* the operator to every check that
exists.

**When you get stuck on something only a human can decide**, run
`kit block 0001 "Should codes expire? Leaning: no."` and move on. That marks 0001 blocked,
so `kit next` stops offering it and points you elsewhere.

## Commands

```sh
kit new   <title...>                     next free ID, date filled in, status Draft
kit claim <id>                           push the branch; exit 1 means you lost it
kit release <id> [--force]               give a claim back; refuses over work, and over
                                         a claim this clone never created
kit block <id> "<question>" [--fyi]      park a decision only a human can make
kit ship  <id>                           verify every box, stamp the date, freeze
kit drop  <id> "<why>"                   retire a feature that will not be built
kit next  [--json] [--offline]           warns above 2 features in progress
kit audit [--json] [--strict] [--offline]  --strict exits 1 on HIGH, for CI
                                         both exit 2 where there is no docs/features
kit init  [PATH] [--stack S] [--forge F]
kit help  [protocol|template|checks|inbox|vision|as-built|go-checklist]
kit version
```

The binary documents itself: `kit help` explains the model and the four rules, and
`kit help checks` lists every finding from the same registry `audit` reports from. An
agent never needs this repository to use the tool. `kit <command> --help` describes the
command, not just its flags.

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
| `1` | it ran and the answer is no: `claim` lost the race, or `audit --strict` found HIGH |
| `2` | the command could not run, or refused: unknown command, bad flag, a refusal |

Diagnostics go to stderr and data to stdout, so `kit next --json` stays pipe-clean even
when something goes wrong.

`kit help checks` prints every finding `audit` can report, generated from the registry, so
it cannot drift. Thresholds are constants in `internal/kit`.

## The feature doc

One file per feature in `docs/features/`, named `NNNN-name.md`. The directory listing is
the board. Frontmatter carries `id`, `title`, `status` (`Draft → Shipped`, exit
`Dropped`), `created`, `shipped` and `needs`. Sections: `Problem`, `Non-goals`, `Design`,
`Tasks`, `Open questions` (optional), `Log`.

Only `kit ship`, `kit drop` and `kit reset` write a status, and only `kit ship` writes a
date. There is no status for work in flight — the claim branch says that, and a second
copy in the doc could only disagree with it.

`needs:` lists the feature ids that must ship before this one can start, and is the only
field that changes what `kit next` offers. Empty means it can start now, which is the
common case — and several empty ones at once is what lets agents work in parallel.
`kit help template` says how to judge it.

Two things keep a doc honest. Every fact has one home: still true after the feature ships
and it belongs in `docs/TECH.md`, one line; a change to this doc's plan and it belongs in
`## Log`, one entry naming the change; neither and it belongs in the pull request. And
blockers belong in `INBOX.md`, not in prose — a box that says "blocked on X" is invisible
to `kit next` and will be handed to an agent that cannot do it.

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
