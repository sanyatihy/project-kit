# TECH

The kit as built: the facts that outlive the feature that produced them. Feature docs in
[features/](features/) carry what is planned; this carries what is true now. It is read
before every change, so one fact goes on one line, and evidence appears only where it
stops the rule being reversed.

## What it is

One Go binary, `kit`, that bootstraps projects and says what to work on. The feature docs
and the git branches *are* the state: no database, no queue, no config file. `git` at
runtime is the only dependency.

```
cmd/kit/          CLI: dispatch, output, help
internal/kit/     feature model, git plumbing, claims, audit, next, scaffold
templates/        what `kit init` writes, embedded via assets.go
```

`internal/kit` is one package split by subject: `feature.go` the document, `git.go` every
subprocess, `claims.go` which branch holds what, `repo.go` discovery, then one file per
command. It was one 1416-line file, which is why nothing noticed that two functions ran
git two different ways.

## Enforce, do not instruct

Prose can be skimmed, misread or reasoned past; a refusal in the tool has to be routed
around deliberately.

| tier | example | an agent gets past it by |
| --- | --- | --- |
| the remote | branch protection, credentials | nothing it can do |
| the kit | `claim` refusing unmet `needs:` | not using the kit |
| prose | "push the branch and stop" | reading carelessly |

A rule belongs at the highest tier it can reach, so a protocol sentence is a last resort:
ask what would make it unnecessary. **A printed command is the same tier as the refusal it
sits next to.** The refusal for an already-claimed feature named `git push --delete`, and
a losing agent that ran it took the winner's branch and claimed the feature on the next
attempt — mutual exclusion defeated using only what the kit printed.

## The model

One markdown file per feature, `docs/features/NNNN-name.md`, with `status:` canonical:

```
Draft -> Shipped     (exit: Dropped)
```

- **Three statuses, and only `ship` and `drop` write one.** There is no status for work in
  flight — the claim branch says that. `Accepted` was removed after 82 docs across five
  projects reached it zero times; every feature went `Draft` to `Shipped`. `Superseded` was
  removed because no command could write it, so reaching it required the hand edit the
  protocol forbids.
- **Six frontmatter fields.** `id`, `title`, `status`, `created`, `shipped`, `needs`.
  `check:`, `checked:` and `review:` went with the commands that wrote them. `related:`
  went because 77 docs filled it in and no line of Go ever read it.
- **What refuses a claim is a list, and it is stated in full wherever it is stated at
  all:** an unmet `needs:`, a BLOCKING inbox line, an unwritten `## Problem`, `## Design`
  or `## Tasks`, an `id:` disagreeing with the filename, a duplicate id, and a plan that
  has not reached the trunk. Saying "a doc nobody wrote and a BLOCKING line" as if that
  were the whole list taught readers that `needs:` does not gate claims. It does.
- **The claim is a branch name, and the push is what decides.** Only a *created* ref is a
  win: pushing to a branch the remote already holds at that commit prints "Everything
  up-to-date" and exits 0, which hands one feature to two agents. Ref creation is a
  server-side compare-and-swap, so exactly one claimant wins. A name is read **whole**,
  namespace included: on the last path segment alone, a worktree reads its own checkout as
  a rival claim.
- **A claim carries the plan, and branches from the trunk.** The doc must be in `HEAD`'s
  tree and in `trunkRef()`'s, or the claim is a child of the plan branch and only its own
  author can integrate it.
- **The checkout is the unit of isolation.** A worktree holds one feature; this checkout's
  own claim reads as *yours*.
- **Destroying a claim asks who made it, and the test is a local head.** `Mine` asks
  whether HEAD *is* the claim branch, which is right for `ship` — shipping happens in the
  worktree doing the work — and wrong for `release`, because `claim` tells you to make that
  worktree, so between claiming and moving there you could not release what you had just
  taken. A local ref means this clone created the branch: the same answer for every
  worktree sharing it, a different one for an agent that only saw the claim over the wire.
- **An unknown answer must not be the permissive one in a command that deletes.**
  `BranchBuildsOn` reads an unresolvable trunk as "carries no work", so `release` deleted
  branches full of commits at exit 0 on any project whose default branch was neither `main`
  nor `master`. The trunk ladder is `origin/HEAD`, local `main`, local `master`, the
  remote-tracking copy of either, then a HEAD that is not claim-shaped — and `release`
  refuses outright when none of them resolves.
- **Ordering needs no syntax.** Task order is dependency order and `next` returns the
  topmost unticked box. This replaced a dependency graph.
- **`needs:` is the only cross-feature ordering, and it is declared.** Blocked until every
  id listed is `Shipped` — not merged, which would make the board depend on how fast
  somebody clicked. ID order was rejected: a prerequisite found late already has a higher
  id. The ready set is therefore the parallelism answer, and nothing declares it separately.
- **Ship has a far side, and the board reports it.** `next` ends with **awaiting review**: a
  branch the remote has, carrying a doc that says Shipped, not yet on the trunk. The status
  is read *from the branch* — the only place it exists, since a coordinator stands on the
  trunk where the doc still says what it said when the round landed. **Status, never
  presence** — a doc reaches the trunk when it is *drafted*, so testing presence silences
  the check for every feature.
- **You never merge your own work: legibility here, enforcement on the remote.** The board
  marks what the reader built — the claim it stands on, or the branch's authorship. The
  second half is worth exactly as much as the identities are distinct, and on this repo
  four agents shared the operator's, so it separated none of them. Branch protection is
  the only real enforcement while that is true.
- **An answer that cannot change inside one command is asked once** — the trunk, this
  checkout's branch, the remotes, the remote's branch list, every ref (`git show-ref`), and
  every feature doc's status on the trunk. Per feature instead, a 30-feature board read
  cost 116 git processes against 29 now, and 10 online. Both are ceilings the gate fails
  on, and both are measured, because a regression can leave the offline number identical.
  `make gitcount` prints them.
- **One project per invocation.** `next` and `audit` read the repo you are in and take no
  paths. The fan-out that read several at once keyed its payload by directory basename, so
  two projects called `proj` reported as one, silently, at exit 0 — and it had no `recover`,
  so a panic in one blanked the board for all. Four days across five projects invoked it
  zero times. A shell loop replaces it, and the two defects were deleted rather than fixed.
- **Claims are read from the remote, not from `refs/remotes/*`**, which git prunes on
  neither `fetch` nor `pull`, so a branch deleted on merge reads as a live claim for ever.
  One `ls-remote` per remote, bounded at 15s — 5s is inside a working remote's latency
  (5.58s to GitHub over SSH) and the board silently printed nothing. `RemoteUnanswered`
  separates *asked and got nothing* from *nothing to ask*, because absence and ignorance
  render identically. `--offline` stays quiet, being an instruction; `claim` never reads
  the cache.
- **Liveness comes from the claim branch, and there is no status for it.** A status set on
  the feature branch never reaches the trunk, which is the only checkout anyone audits. The
  log is read `--all` for the same reason; a doc's mtime is not a signal, since a merge
  resets it.
- **A claim finding needs positive evidence, not absence.** From the trunk an in-flight
  feature's doc is *always* missing, so absence would report every open feature while
  advising a fix that deletes the draft. A branch is an orphan only if its own tree says so.
- **Drafting is a plan round, and drafts live on the trunk.** One `plan/<something>` branch
  per session, merged once — and its name must not mention a feature id *anywhere*, since
  any branch citing one is a claim and such a draft claims itself. Not only the first
  segment: `plan/fold-the-fix-into-0039` is a claim on 0039. `kit drop` exists because a
  doc on the trunk cannot be retired by deleting a branch.
- **Blocking is an INBOX line naming the feature in its id field**, never the prose, which
  stops work invisibly; an unparseable line matches whole-line, which over-blocks — the safe
  direction. `next` skips at once, `audit` escalates after two days, because a line carries
  a date and no time. FYI lines never block, and are LOW where BLOCKING is HIGH.
- **The feature template's guidance lives in HTML comments, and that is load-bearing.**
  Comments are stripped before parsing, so guidance reaches the author and not the tool.
  Visible placeholder prose and two `- [ ] …` boxes make an unwritten doc parse as a
  finished plan, which `next` then hands an agent to build.
- **The plan checks gate on *committed*, not on a status.** `unfilled-template` and
  `no-tasks` were gated on "not a Draft", which dropping `Accepted` would have made dead.
  Committed-and-unwritten is the truer defect anyway, and a doc fresh out of `kit new` is
  untracked and stays quiet.
- **Three docs, three tenses:** VISION what the project is becoming, `features/` what is
  being done now, TECH what exists. **A file is the kit's to refresh when nothing reads the
  project's copy** — `TEMPLATE.md` is written from the embedded copy, so `init` replaces it
  and it cannot describe a format the tool no longer produces. Anything a project fills in
  is never touched; house rules go above the marker.
- **`kit init` writes an `AGENTS.md` with a title and the marker note.** It used to write
  the bare protocol section, opening on an `##` heading with no project name and nothing
  saying the section is regenerated — so the promise that house rules survive above the
  marker reached nobody.
- **Every fact has one home, and the routing is decided at ship.** Still true afterwards and
  it belongs here, one line; a change to the plan and it belongs in that doc's `## Log`,
  naming the change; neither and it belongs in the pull request.
- **A feature is one shipped increase in what the project can do, or can be trusted to do.**
  Named in one sentence an operator would recognise, or it is a task box or a chore commit;
  a sentence needing an "and" is two features. This replaced "records one decision", which
  measured the document rather than the value and admitted a pure refactor while excluding
  an increment nobody had to decide anything about. Size stays the sanity check: too big is
  one nobody reviews in a sitting, and the split is by value, never by layer.
- **Every run of `next` ends with an instruction**, in every state. The newest was a doc
  straight out of `kit new`: it lands in `blocked`, and the closing line said stopping was
  the right outcome when the work was to write the plan just created.
- **`protocol-stale` compares the project's protocol section with the kit's**, because
  shipping a protocol change does not deliver one — `AGENTS.md` is rewritten only by `init`.
  **Rules are cited by name, never by number**, in every shipped string as well as in the
  protocol. A number is valid only inside the one list that carries it, and a refusal that
  cited one sent the reader to a list they were not holding.
- **`secret-in-doc` matches shapes, not entropy** — seven formats whose issuer publishes a
  fixed prefix. HIGH, because a published credential is the one finding late discovery
  cannot fix. The patterns live in `audit.go` only; in a template they would trip the check.
- **Checks are a table, not a function.** `featureChecks` runs per feature and `repoChecks`
  once per project; each entry names the registry codes it can report, and a test fails the
  build if a registered check has no implementation or two claim one code.
- **`audit` has one job:** find where the docs disagree with the repo. It never writes — an
  auditor that fixes things stops being read — and it counts boxes rather than estimating.
- **A check that reports a state nobody may fix is deleted, not tuned.** A Shipped doc is
  frozen, so a finding whose only remedy edits one is a standing instruction to break the
  protocol. `missing-section` demanded a `## Log` the routing rule says is optional;
  `draft-claimed` read any `## Open questions` text as unsettled, including answered ones;
  `doc-bloat` offered an escape hatch that existed in no line of code; `session-narration`
  skipped Shipped, which was 13 of the 18 docs in the only corpus written under the rule;
  `long-sentence` cost a 130-line sentence parser for one LOW finding on frozen docs. All
  five are gone, and `needs-unknown` now skips a doc that is itself terminal.
- **The kit owns the work, not the runtime.** Where agents run — panes, restarts, noticing
  one is blocked — belongs to a runtime like `herdr`; an agent composes the two.

## Decisions and why

- **Go, not Python**, for `go install` distribution. The cost is real: a binary cannot be
  patched in place by an agent, and it drifts from source until `make install` runs.
- **The repo is private, and that has a price.** `go install` needs `GOPRIVATE` and
  credentials, which is why generated CI does not run `kit audit --strict`: every scaffolded
  project would carry a credential to rotate. The kit's own CI audits, because
  `go run ./cmd/kit` authenticates to nothing.
- **PDCA was rejected** as the frame — it comes from repeated processes and software changes
  are mostly one-offs, so mandatory predictions became ceremony. The optional
  `check:`/`checked:` pair that survived it was itself removed: zero of 82 docs used it.
- **Deployment scaffolding is not generated**, and the topic that documented it is gone.
  `--stack go` produces what you need to write and verify code; the rest is the
  fastest-rotting part, and no project ever read the topic.
- **The forge is detected at `init`, written as a file, and never consulted again.** First
  rung with positive evidence: the project's CI directory, the remote's host, then the
  module path's host segment. Unknown writes **no CI file** and says so — a missing pipeline
  is visible, one for the wrong forge is not.
- **Prevent before detecting.** `block`, `ship`, `release` and `drop` exist because writing
  an INBOX line, a status, or a branch deletion by hand each cost something a command makes
  impossible. `ship` refuses a tree with anything uncommitted in it, the debris being the
  feature's own.
- **The commands refuse exactly what the board refuses** — `Survey`, `ClaimFeature` and
  `Ship` share `UnmetNeeds`, and `Ship` reuses `Repo.Mine`.
- **An `id:` disagreeing with its filename is HIGH, not tidiness.** The branch comes from
  the file and the claim is looked up by the id, so the board keeps offering a feature
  somebody holds.
- **One path per document kind.** Three feature directories and four spellings of the
  as-built doc were accepted for projects the kit never scaffolded, and `createTechDoc`
  hard-coded one of the four anyway — so a project using a different spelling got a second
  as-built doc rather than an updated one.
- **Exit codes are a contract**, pinned by a test: `0` worked, `1` it ran and the answer is
  no (`claim` lost the race, `audit --strict` found HIGH), `2` it could not run or refused.
- **Dates are passed in, never read inside** — a clock inside leaves tests unable to catch a
  UTC-midnight off-by-one.
- **Standing guidance is embedded, never copied into projects.** `go-checklist` is practice,
  not documentation of the kit, and `audit` cannot check a word of it — a copy per repo
  would be drift with nothing to notice it. It must never use `- [ ]`, which in a kit
  project means a PR-sized unit of work; a test pins that.
- **Shipped text has line ceilings** (`cmd/kit/budget_test.go`), because nothing else stops
  a doc growing and no reviewer counts lines. Tiered by how often it is read: the protocol
  tightest, then command help, then topics, and now the README. Ceilings only — a floor
  rewards padding.

## Gotchas

Each of these has already caused a bug.

- **Files under `templates/` must not be valid Go package files or a nested module**, or
  `go vet ./...` compiles them and `go:embed` skips them. Hence `gomod`, `gitignore`,
  `main.go.tmpl`.
- **Templates are embedded through `assets.go` at the repo root** — `go:embed` cannot reach
  a parent directory from `cmd/kit/`.
- **Go's `flag` stops parsing at the first positional.** `parseArgs` loops it so a flag
  after an argument still parses.
- **The `## Work protocol` section of a project's `AGENTS.md` is regenerated** by `init`.
  Edits inside it are lost; durable house rules go above the marker.
- **`kit help checks` is generated from the registry `audit` reports from.** Add checks
  there, never as bare strings.
- **The installed binary is not the source.** `make install` after anything lands, or test
  against `./kit`. A real project once ran two features behind and nothing said so.
- **`go install` does not apply the Makefile's `-ldflags`**, so `resolveVersion` falls back
  to the module version in the build info.
- **`CGO_ENABLED=0` must not reach `make test`.** The race detector needs cgo on
  linux/amd64 but not darwin/arm64, so a file-level export passes locally and fails CI.
- **Guidance the binary does not print may as well not exist.** An outside review reached
  four wrong conclusions, every one a fair reading of `kit help`. Two tests keep that shut.
- **`linkRe` accepts any run of non-`)` characters**, so `[x](   )` yields no fields at all.
  Indexing that slice panicked inside the command that gates every ship and every CI run.
- **`GOPRIVATE` is a comma-separated list, and a shell `export` beats `go env -w`.** Append
  to an existing company value; the `-w` write is inert when the variable is exported.

## Verifying

```sh
make check     # fmt-check, vet, unit tests, conformance suite — the whole gate
make vuln      # govulncheck against the module
make install   # go install ./cmd/kit
```

Builds are reproducible: `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w"`. CI runs the same
gate plus `golangci-lint`, `make vuln` and `go run ./cmd/kit audit --strict` — `go run` so
it needs no credential.

**The conformance suite** (`cmd/kit/conformance*_test.go`) builds the binary once and drives
it against throwaway projects, asserting on stdout, stderr, exit codes and the files left
behind. In-tree, because an external script is deleted without anything noticing. Four
properties are load-bearing. `TestEveryRegisteredCheckIsExercised` compares the cases
against `kit.Checks` both ways, so a check with no test fails the build. Every audit case
starts from the same clean project, which must report nothing. Every test is independent —
own `t.TempDir()`, no `os.Chdir` or `os.Setenv`, binary built once behind `sync.Once`.
**And the binary under test gets `gitEnv()`, not `os.Environ()`**: it used to inherit the
developer's `~/.gitconfig`, so three tests — including the one written to catch the
`release` deletion bug — passed only on a machine whose `init.defaultBranch` is `main`.

**No coverage number is reported.** `-cover` measured the test process rather than the
binary the suite drives, printing 2.9% for a package covered end to end; instrumenting it
is `go build -cover` plus `go tool covdata`, and no threshold reads the result.

## State of play

Private repo, tagged `v0.2.0`, `go install …/cmd/kit@latest`; four projects run it.
