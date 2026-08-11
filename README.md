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

The repo is private, so the module proxy cannot serve it. Tell Go to skip the proxy
for it, once per machine:

```sh
go env -w GOPRIVATE=github.com/sanyatihy/*
```

Go then fetches it with plain `git`, so that has to be able to authenticate to
GitHub. If you use SSH keys, make `git` prefer them over HTTPS:

```sh
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

Then:

```sh
go install github.com/sanyatihy/project-kit/cmd/kit@latest
```

Or from a clone: `make install` — no `GOPRIVATE` needed, the source is already
there. Upgrading is the same command again.

A `410 Gone` or `404 Not Found` from `proxy.golang.org` means `GOPRIVATE` is not
set; a git authentication prompt or `terminal prompts disabled` means it is set
and the credentials are what is missing.

## Setup

```sh
kit init /path/to/project              # docs only
kit init /path/to/project --stack go   # also Makefile, go.mod, cmd/, CI
```

Run this from anywhere — `kit` is on your `$PATH`, so you never need to be inside
the kit's own repo. Prompting an agent there to set up some other project gives it
write access to the tool it is using, and the kit's `AGENTS.md` will be telling it
to work on the kit's own plan. Bootstrap from the new project's parent directory, or
from wherever you already are.

Creates the directory if it does not exist, makes it a git repo if it is not one,
and writes `docs/features/{README,TEMPLATE}.md`, `docs/INBOX.md`, and `AGENTS.md`.
It never overwrites or deletes, so it is safe on a directory that already has files.

With `--stack go` it also scaffolds the code layout: `Makefile`, `.gitignore`,
`go.mod` (module path inferred from where the project sits on disk), `cmd/<name>/`,
`internal/`, a `README.md`, and one CI workflow running `make check` and
`kit audit --strict`.

So every project answers the same three questions with the same three commands:
`kit next` (what to work on), `kit audit` (what drifted), `make check` (is it
green). That uniformity is the point — an agent's first five minutes are identical
in every repo.

Deployment scaffolding — terraform, ansible, images, dashboards — is deliberately
**not** generated. It is the fastest-rotting part and it is not needed to write
code. Add it when the project actually has something to deploy.

Re-running is safe: your files are kept, and only the `## Work protocol` section of
`AGENTS.md` is refreshed, so projects pick up protocol changes without losing their
own rules. No `TECH.md` — that describes what exists, so it waits until something
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

Then set `status: Accepted` and run `kit next`. It hands you the first box and the
command to start:

```
  start here
    0001  Shorten URLs
    → Schema + migration for links(id, url, created_at)
       docs/features/0001-shorten-urls.md
       git worktree add ../url-shortener-0001 -b 0001-shorten-urls
```

Do that one box, tick it, open a PR. Run `kit next` again for the next one. When the
last box is ticked, set `status: Shipped` and the `shipped:` date.

**When you get stuck on something only a human can decide**, add a line to
`docs/INBOX.md` with your leaning and move on:

```
- [ ] 2026-08-11 · 0001 · BLOCKING · Should codes expire? Leaning: no, keep forever.
```

That marks 0001 blocked, so `kit next` stops offering it and points you elsewhere.

## The four rules

1. `In progress` features come before `Accepted` ones. Finishing beats starting.
2. Within a feature, do the topmost unticked box. List order is the order of work.
3. One agent per feature; a branch named after the doc is the claim.
4. A feature named in an open `INBOX.md` question is blocked.

## Commands

```sh
kit new   <title...>                     next free ID, date filled in, status Draft
kit next  [PATH ...] [--json]            warns above 2 features in progress
kit audit [PATH ...] [--json] [--strict] --strict exits 1 on HIGH, for CI
kit help  [protocol|template|conventions|checks]
kit version
```

The binary documents itself: `kit help` explains the model and the four rules,
`kit help checks` lists every finding from the same registry `audit` reports from,
and `kit help protocol|template|conventions` prints the embedded documents. An
agent never needs this repository to use the tool. `kit <command> --help` describes
the command, not just its flags.

`next` and `audit` read only and never write. With no path they use the repo you are in, from any
subdirectory; `kit next ~/code/*` surveys every project at once.

`audit` names the file, the problem, and the fix. It reports:

- **HIGH** — `stalled`, `unticked-shipped`, `inbox-stale`, `unchecked`
- **MEDIUM** — `no-plan`, `done-not-shipped`, `tech-drift`, `shipped-no-date`,
  `missing-section`, `id-mismatch`, `no-status`, `bad-status`, `no-protocol`
- **LOW** — `dead-link`, `no-inbox`, `no-tech`

Thresholds are constants in `internal/kit`.

## The feature doc

One file per feature in `docs/features/`, named `NNNN-name.md`. The directory listing
is the board. Frontmatter carries `id`, `title`, `status`
(`Draft → Accepted → In progress → Shipped`, exits `Dropped`/`Superseded`), `created`,
`shipped`. Sections: `Problem`, `Non-goals`, `Design`, `Tasks`, `Open questions`
(optional), `Log`.

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
make check     # fmt-check, vet, go unit tests
```

If you add a check to `audit`, add a test that it fires, a test that it stays quiet
when it should, and document the code above.
