# TECH

The kit as built. Feature docs in [features/](features/) carry what is planned;
this file describes what exists.

## What it is

One Go binary, `kit`, that bootstraps projects and tells you what to work on.
Plans live as markdown in each project's `docs/features/`; there is no database
and no queue. The feature docs and the git branches *are* the state.

```
cmd/kit/          CLI: dispatch, output, help
internal/kit/     parse, repo model, audit, next, scaffold, newdoc
templates/        what `kit init` writes, embedded via assets.go
```

Requires only git at runtime. Python is not used anywhere.

## The model

A project has one markdown file per feature, `docs/features/NNNN-name.md`, whose
`status:` frontmatter is canonical:

```
Draft -> Accepted -> In progress -> Shipped     (exits: Dropped, Superseded)
```

`status:` is also the autonomy dial. `Draft` is where a human settles the design,
and `kit next` never offers one as work; once `Accepted`, an agent can work it
unattended. Four rules govern the rest, and `kit help` prints them.

Mechanisms worth knowing before changing anything:

- **Ordering needs no syntax.** Task list order is dependency order; `next` returns
  the topmost unticked box. This replaced a dependency graph.
- **The claim is a branch name.** A branch citing a feature ID means that feature is
  taken. Zero new state, visible across worktrees and — once pushed — machines.
- **Blocking is an INBOX line naming the feature.** `next` skips it; `audit`
  escalates it after 7 days.
- **`audit` never writes.** An auditor that fixes things stops being read.

## Decisions and why

Recorded so they are not re-litigated. Each was considered and settled.

- **Go, not Python.** The kit began as Python scripts; the port is verified against
  the same behavioural suite. Chosen for `go install` distribution and because this
  is a Go shop. The cost is real: a binary cannot be read or patched in place by an
  agent the way a script could, and the installed binary drifts from source until
  `make install` runs.
- **`go install`, not per-project vendoring.** Earlier versions copied the commands
  into each project's `.kit/`. A binary cannot be vendored as source, so projects now
  depend on `kit` being installed, and generated CI installs it in a step.
- **PDCA was considered and rejected** as the organising frame. It comes from
  repeated manufacturing processes; software changes are mostly one-offs, so
  mandatory predictions became ceremony. What survived is an entirely **optional**
  `check:`/`checked:` pair, for features that are genuine bets. Never backfill a
  `check:` onto something already shipped — that is predicting a known result.
- **Deployment scaffolding is not generated.** `--stack go` produces what you need to
  write and verify code, never terraform/ansible/images. That is the fastest-rotting
  part and is not needed on day one. Going to production is a feature doc like any
  other.
- **One job for `audit`:** find where the docs disagree with the repo. Resist adding
  a second.

## Gotchas

Each of these has already caused a bug.

- **Files under `templates/` must not be valid Go package files or a nested module**,
  or `go vet ./...` compiles them and `go:embed` skips them. Hence `gomod`,
  `gitignore`, `main.go.tmpl`. A `go.mod` in there silently emptied the embed.
- **Templates are embedded through `assets.go` at the repo root**, because
  `go:embed` cannot reach a parent directory from `cmd/kit/`.
- **Go's `flag` stops parsing at the first positional.** `parseArgs` in
  `cmd/kit/main.go` loops it so `kit audit PATH --json` works. Any new subcommand
  taking paths must use it.
- **The `## Work protocol` section of a project's `AGENTS.md` is regenerated** by
  `kit init`. Edits inside it are lost; put anything durable above the marker.
- **`kit help checks` is generated from the same `Check` registry `audit` reports
  from**, so severity and meaning cannot drift. Add checks to that registry, never
  as bare strings.
- **The installed binary is not the source.** Run `make install` after changes, or
  test against `./kit`.

## Verifying

```sh
make check     # fmt-check, vet, go unit tests
make install   # go install ./cmd/kit
kit audit .    # the kit passes its own audit
kit next .     # its own plan
```

`make check` is currently weaker than it looks: the conformance suite that drove the
binary end to end was deleted while untracked and is tracked as work in
[features/0002-restore-the-conformance-suite.md](features/0002-restore-the-conformance-suite.md).
Unit tests cover parsing, doc creation, and help consistency; nothing yet exercises
`init`, `next`, or `audit` as a user runs them.

## State of play

**Nothing is committed.** The repo has no commits and no remote, which is why the
documented `go install github.com/sanyatihy/project-kit/cmd/kit@latest` cannot
resolve yet, and why the deleted suite was unrecoverable. That is
[features/0001-publish-the-kit.md](features/0001-publish-the-kit.md), and it is the
first thing `kit next` hands back.
