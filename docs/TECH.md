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
  depend on `kit` being installed on the machine that uses it.
- **The repo is private, and that has a price.** `go install` works, but only with
  `GOPRIVATE` covering `github.com/sanyatihy/*` and git credentials for the host —
  see the README. It is why generated CI does not run `kit audit --strict`: doing so
  means installing a private module, so every scaffolded project would carry a
  credential to store and rotate and would fail its first push until someone set one.
  Zero-setup scaffolding beat catching drift a few hours before `kit next` does. The
  kit's own CI *does* audit, because it has the source and `go run ./cmd/kit` needs
  nothing to authenticate. If the kit ever goes public, both costs vanish and the two
  steps can go back into the template.
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
- **Exit codes are a contract**, documented in `kit help` and the README and pinned
  by `TestExitCodesAreDistinct`: `0` worked, `1` only ever means `audit --strict`
  found something HIGH, `2` means the command could not run. An agent scripting the
  kit has to tell a typo from a finding.
- **Dates are passed in, never read inside.** Both `Audit` and `NewFeature` take
  `today` from `main`. One of them used to read the clock, and the cost was a test
  that could only assert "some date that looks like today" — which cannot catch an
  off-by-one at a UTC midnight boundary.
- **Standing guidance is embedded, never copied into projects.** `kit help
  go-checklist` and `kit help deployment` carry engineering practice and deployment
  patterns that are not documentation of the kit. They are `help` topics because
  `audit` cannot check a word of either — so a copy in each repo would be drift with
  nothing to notice it, which is the exact failure the kit exists to prevent. One
  copy, updated on upgrade.
- **Guidance must never use `- [ ]`.** In a kit project that syntax means "a
  PR-sized unit of work, do the topmost one". Source material that uses it for "a
  property to verify" gets converted to prose bullets, or an agent following protocol
  rule 3 works through it as a task list. `TestNoTopicLooksLikeATaskList` pins this,
  exempting the topics whose subject matter *is* the feature-doc format.

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
- **`go install` does not apply the Makefile's `-ldflags`,** so `main.Version` stays
  at its compiled-in default for anyone installing over the network. `resolveVersion`
  falls back to the module version Go embeds in the build info — the tag for
  `@v0.1.0`, a pseudo-version for `@latest` off an untagged branch. Without that, a
  tagged release reports `dev` and no one can tell what they are running.
- **`CGO_ENABLED=0` must not reach `make test`.** The race detector requires cgo on
  linux/amd64 but not on darwin/arm64, so a file-level `export` in the Makefile
  passes every local run and fails CI with `-race requires cgo`. The build flags are
  set per-recipe for this reason. More generally: a green `make check` on a Mac is
  not evidence about the Linux runner.
- **`GOPRIVATE` is a comma-separated list, and a shell `export` beats `go env -w`.**
  A machine with an existing company `GOPRIVATE` must append, not replace; the `-w`
  write is silently inert when the variable is exported in a profile. Go warns about
  the conflict, and the warning is easy to scroll past.

## Verifying

```sh
make check     # fmt-check, vet, unit tests, conformance suite
make vuln      # govulncheck against the module
make install   # go install ./cmd/kit
kit audit .    # the kit passes its own audit
kit next .     # its own plan
```

Builds are reproducible: `CGO_ENABLED=0`, `-trimpath`, and `-ldflags "-s -w"`, so
the output depends on the source and nothing about the machine. Module build info
survives the stripping, which matters because `kit version` falls back to it when
the ldflags stamp is absent.

`.github/workflows/check.yml` runs `golangci-lint`, `make check`, `make vuln`, and
`go run ./cmd/kit audit . --strict` on every PR and every push to `main`. The audit
step uses `go run` rather than an installed binary so it needs no credential.

**The conformance suite** is `cmd/kit/conformance*_test.go`. It builds the binary
once and drives it against throwaway projects, asserting on stdout, stderr, exit
codes, and the files left behind — `init`, `next`, `new`, and `audit` as a user runs
them, plus BOM/CRLF, missing frontmatter, unparseable dates, non-git directories,
and the flag orders people actually type.

It is a Go test in-tree rather than an external script. The previous suite was
Python driving the binary, untracked and outside the build, which is exactly how it
could be deleted without anything noticing.

Two things about it are load-bearing:

- **`TestEveryRegisteredCheckIsExercised`** compares the conformance cases against
  `kit.Checks` in both directions. AGENTS.md has always asked for a test that a new
  check fires and one that it stays quiet; this makes the first half a build
  failure rather than an honour-system rule.
- **Every audit case starts from the same clean project**, which must report
  nothing. A noisy baseline would make all sixteen firing cases vacuous.

`cmd/kit` coverage reads about 6% and will stay there: these tests drive a
subprocess, so nothing they exercise is instrumented. The number is not a signal
for this package.

## State of play

Published. `github.com/sanyatihy/project-kit` is a private repo tagged `v0.1.0`, and
`go install github.com/sanyatihy/project-kit/cmd/kit@latest` resolves to that tag —
verified from an empty `GOMODCACHE`, not just assumed. Setting up a new machine takes
the two settings in the README's install section.

The testing gap is closed. `make check` is now the whole gate: fmt, vet, unit tests,
and the conformance suite that drives the commands end to end. The kit's own CI runs
that plus `golangci-lint` and its own audit.

The backlog is empty: the findings from reviewing the kit against
`kit help go-checklist` were closed by
[features/0004-close-the-go-checklist-findings.md](features/0004-close-the-go-checklist-findings.md).
