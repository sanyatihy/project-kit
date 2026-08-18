# TECH

The kit as built: the facts that outlive the feature that produced them. Feature docs in
[features/](features/) carry what is planned and what was decided while doing it; this
carries what is true now. It is read before every change, so it stays short — a fact that
only records that we thought about something does not belong here.

## What it is

One Go binary, `kit`, that bootstraps projects and says what to work on. The feature docs
and the git branches *are* the state: no database, no queue, no config file. Git at
runtime is the only dependency.

```
cmd/kit/          CLI: dispatch, output, help
internal/kit/     parse, repo model, audit, next, scaffold, newdoc
templates/        what `kit init` writes, embedded via assets.go
```

## Enforce, do not instruct

Prose can be skimmed, misread or reasoned past; a refusal in the tool has to be routed
around deliberately. Measured: the rule against merging your own work was broken twice by
an agent that had read it, and `ship`'s coded refusal was obeyed the first time it fired,
on that same agent minutes later.

| tier | example | an agent gets past it by |
| --- | --- | --- |
| the remote | branch protection, credentials | nothing it can do |
| the kit | `claim` refusing unmet `needs:` | not using the kit |
| prose | "push the branch and stop" | reading carelessly |

A rule belongs at the highest tier it can reach, so a protocol sentence is a last resort:
ask what would make it unnecessary. Where prose is the only option — judgement, taste — say
so rather than implying a guarantee.

## The model

One markdown file per feature, `docs/features/NNNN-name.md`, with `status:` canonical:

```
Draft -> Accepted -> Shipped     (exits: Dropped, Superseded)
```

`status:` **orders the board rather than gating it** — accepting says a human blessed the
design, not that work may start, and agents claim Drafts. What refuses a claim is a BLOCKING
inbox line and a doc nobody wrote; `Feature.Unwritten()` is the single answer to "nobody
wrote the plan", shared by `audit`, `claim` and `next`, so the board never offers what the
next command refuses.

- **The claim is a branch name, and the push is what decides.** Branches and worktrees
  citing an ID all count, while the work is live. Only a *created* ref is a win: pushing to
  a branch the remote already holds at that commit prints "Everything up-to-date" and exits
  0, which hands one feature to two agents. Ref creation is a server-side compare-and-swap,
  so exactly one claimant wins. A leftover branch is indistinguishable from a fresh claim,
  so `claim` names both possibilities and `stalled` never ages a claim carrying no work.
- **A claim carries the plan, and branches from the trunk.** The doc must be in `HEAD`'s
  tree — the push is `HEAD:refs/heads/<branch>` — and in `trunkRef()`'s, or the claim is a
  child of the plan branch and only its own author can integrate it. Exit 2, not `NotYours`:
  nothing on an unlanded round is claimable.
- **The checkout is the unit of isolation.** A worktree holds one feature; this checkout's
  own claim reads as *yours*, and `next` names no starting point where one is held.
- **Ordering needs no syntax.** Task order is dependency order and `next` returns the
  topmost unticked box. This replaced a dependency graph.
- **`needs:` is the only cross-feature ordering, and it is declared.** Blocked until every
  id listed is `Shipped` — not merged, which would make the board depend on how fast
  somebody clicked. ID order was rejected: a prerequisite found late already has a higher
  id. Parsing is lenient, checks are strict; whether the needs are the *right* ones is a
  judgement `audit` refuses. The ready set is therefore the parallelism answer, and nothing
  declares it separately.
- **Ship has a far side, and the board reports it.** `next` ends with **awaiting review**: a
  branch the remote has, carrying a doc that says Shipped, not yet on the trunk. The status
  is read *from the branch* — the only place it exists, since a coordinator stands on the
  trunk where the doc still says what it said when the round landed. `shipped-not-pushed` is
  the other half at HIGH, and needs two negatives: no branch on the remote names it, and the
  remote's trunk carries no copy of the doc saying Shipped. **Status, never presence** — a doc
  reaches the trunk when it is *drafted*, so testing presence silences the check for every
  feature — and on **any** branch the repo counts as a trunk, since a clone whose
  `origin/HEAD` predates a default-branch rename points at one nothing merges to, and
  reading only that one fires HIGH on every shipped feature. Reading the commit that last wrote the doc was the earlier answer and misfired the
  other way, once compaction made editing a shipped doc routine.
- **You never merge your own work: legibility here, enforcement on the remote.** The board
  marks what the reader built (the claim it stands on, or the branch's authorship) and what
  `review: human` reserves for a person — written by `kit accept --review`, read from the
  checkout, any non-empty value counting. Branch protection is the floor and the only real
  enforcement; none of this distinguishes anybody while agents share one credential, as four
  agents' commits here all carried the operator's name.
- **An answer that cannot change inside one command is asked once** — the trunk, this
  checkout's branch, the remotes, the remote's branch list, every ref (`git show-ref`), and
  every feature doc's status on the trunk, which `ls-tree` and one `cat-file --batch` read in
  two processes. Per feature instead, a 30-feature board read cost 116 git processes against
  28 now. `make gitcount` counts them **online as well as off**: the same regression hid for a
  release behind an offline-only measurement, where the count was identical either way.
- **The board is read concurrently, capped at 8, in the order the paths were given.** What a
  board read spends is one `ls-remote` per project — 2.3s against GitHub over SSH where the
  rest is under a tenth of that — so a fleet costs one round trip rather than one each: four
  projects went from 11.2s to 2.5s. The round trip starts before the local walk for the same
  reason. The cap bounds connections to one forge, not CPU, and the order is what lets one
  board be compared with the last.
- **Claims are read from the remote, not from `refs/remotes/*`**, which git prunes on
  neither `fetch` nor `pull`, so a branch deleted on merge read as a live claim for ever.
  One `ls-remote` per remote, bounded at 15s — it was 5, which a working remote did not fit
  inside (5.58s to GitHub over SSH), and the board silently printed nothing. The bound was
  half the bug: expiry reported as an empty page is worse, because absence and ignorance
  render identically, so `RemoteUnanswered` separates *asked and got nothing* from *nothing
  to ask*. `--offline` stays quiet, being an instruction; `claim` never reads the cache.
- **Liveness comes from the claim branch, and there is no status for it.** An
  `In progress` field was set on the feature branch and never reached the trunk, so from
  the only checkout anyone audits the doc read `Accepted` and `stalled` could not fire —
  and the protocol forbade the hand edit that was the only way to set it. Dropping it took
  `no-claim` with it, which existed only to reconcile the two copies. The log is read
  `--all` for the same reason; a doc's mtime is not a signal, since a merge resets it.
- **A claim finding needs positive evidence, not absence.** From the trunk an in-flight
  feature's doc is *always* missing, and `orphan-claim` reported every open feature while
  advising a fix that deletes the draft. A branch is an orphan only if its own tree says so;
  an unfetched branch is passed over rather than guessed at.
- **The trunk is asked for, not assumed:** `origin/HEAD`, then `main`, then `master`, then a
  HEAD that is not claim-shaped. Hard-coding the two names left a project whose default is
  neither with nothing to diff against, and every check built on it went quiet.
- **Drafting is a plan round, and drafts live on the trunk.** One `plan/<something>` branch
  per session, merged once, at whatever statuses its docs reached — never named after a
  feature, since a leading four-digit ID *is* a claim and such a draft claimed itself. A
  branch carrying only its doc is a proposal, which is what `draft-claimed` diffs for.
  `kit drop` exists because a doc on the trunk cannot be retired by deleting a branch.
- **Accepting is a command, and only a human runs it.** It refuses a non-Draft, an unemptied
  `## Open questions`, and a doc a BLOCKING line names — via `BlockersFor`, so `accept` and
  `next` cannot disagree about what is blocked.
- **Blocking is an INBOX line naming the feature in its id field**, never the prose, which
  used to stop work invisibly; an unparseable line matches whole-line, which over-blocks —
  the safe direction. `next` skips at once, `audit` escalates after two days, because a line
  carries a date and no time. BLOCKING and FYI differ in severity too — HIGH and LOW — since
  `--fyi` promises the feature is not blocked, and HIGH broke that promise through the
  `--strict` gate in front of `ship`.
- **The feature template's guidance lives in HTML comments, and that is load-bearing.**
  Comments are stripped before parsing, so guidance reaches the author and not the tool.
  Visible placeholder prose and two `- [ ] …` boxes once made an unwritten doc parse as a
  finished plan: `next` handed an agent `…` to build and `audit` agreed. `unfilled-template`
  is HIGH on the back of that, and cannot misfire because an unwritten section parses empty.
- **Three docs, three tenses:** VISION what the project is becoming, `features/` what is
  being done now, TECH what exists. **A file is the kit's to refresh when nothing reads the
  project's copy** — `TEMPLATE.md` and `features/README.md` are written from the embedded
  copy, so `init` replaces them and they cannot describe a format the tool no longer
  produces. Anything a project fills in is never touched; house rules go above the marker.
- **A feature is one pull request, and that decides the decomposition.** Too big is one
  nobody reviews in a sitting; too small changes nothing observable. "One agent's reach" was
  satisfied by seven features and by one covering a project. Boundaries come from the
  vision's `## What done looks like`.
- **Every run of `next` ends with an instruction**, in all four states: three vacuums were
  closed the same way, each an agent reaching the end of the documented path and deciding
  sensibly and wrongly. `Plan.Guidance` is decided once and carried by both formats; a taken
  or blocked feature carries no `task`.
- **`protocol-stale` compares the project's protocol section with the kit's**, because
  shipping a protocol change does not deliver one — `AGENTS.md` is rewritten only by `init`.
  **Rules are cited by name, never by number**, with a test: two cross-references pointed at
  the wrong rules for months after later features shifted the list.
- **`secret-in-doc` matches shapes, not entropy** — seven formats whose issuer publishes a
  fixed prefix. HIGH, because a published credential is the one finding late discovery
  cannot fix. The patterns live in `audit.go` only; in a template they would trip the check.
- **Checks are a table, not a function.** `featureChecks` runs per feature and `repoChecks`
  once per project; each entry names the registry codes it can report, and a test fails the
  build if a registered check has no implementation or two claim one code. They were one
  362-line function, where no check could be read or tested alone.
- **`audit` has one job:** find where the docs disagree with the repo. It never writes — an
  auditor that fixes things stops being read — and it counts boxes rather than estimating.
- **The kit owns the work, not the runtime.** Where agents run — panes, restarts, noticing
  one is blocked — belongs to a runtime like `herdr`; an agent composes the two.

## Decisions and why

- **Go, not Python**, for `go install` distribution; projects depend on `kit` being on the
  machine rather than vendored. The cost is real: a binary cannot be patched in place by an
  agent, and it drifts from source until `make install` runs.
- **The repo is private, and that has a price.** `go install` needs `GOPRIVATE` and
  credentials, which is why generated CI does not run `kit audit --strict`: every scaffolded
  project would carry a credential to rotate. The kit's own CI audits, because
  `go run ./cmd/kit` authenticates to nothing. Going public removes both costs.
- **PDCA was rejected** as the frame — it comes from repeated processes and software changes
  are mostly one-offs, so mandatory predictions became ceremony. An optional
  `check:`/`checked:` pair survives, for genuine bets. Never backfill one onto shipped work.
- **Deployment scaffolding is not generated.** `--stack go` produces what you need to write
  and verify code; the rest is the fastest-rotting part and not needed on day one.
- **The forge is detected at `init`, written as a file, and never consulted again.** First
  rung with positive evidence: the project's CI directory, the remote's host, then the
  module path's host segment when it names a known forge. Unknown writes **no CI file** and
  says so — a missing pipeline is visible, one for the wrong forge is not. Not
  configuration: the answer becomes a file at init, so nothing can drift.
- **Prevent before detecting.** `kit block` and `kit ship` exist because hand-writing an
  INBOX line and shipping by hand produced four findings. A command that makes the mistake
  impossible beats a check that reports it.
- **The commands refuse exactly what the board refuses** — `Survey`, `ClaimFeature` and
  `Ship` share `UnmetNeeds`, and `Ship` reuses `Repo.Mine`. They disagreed once: `next`
  withheld a feature whose needs had not shipped and `claim` handed it over.
- **An `id:` disagreeing with its filename is HIGH, not tidiness.** The branch comes from
  the file and the claim is looked up by the id, so the board keeps offering a feature
  somebody holds.
- **One unreadable path must not blank the fleet board.** Failures are reported in place; a
  bad path exits 2, outranking `--strict`'s 1, because a project you asked about and did not
  get is the worse answer. Hence `audit --json` is `{"findings": […], "error": …}` — a bare
  list has nowhere to say why it is empty.
- **`unfilled-vision` closes the gap `unfilled-template` did, one file over:** `init` writes
  a stub, so `no-vision` fired only when the file was missing, and the run that created the
  stub silenced the check.
- **Exit codes are a contract**, pinned by a test: `0` worked, `1` only ever means
  `--strict` found something HIGH, `2` means the command could not run.
- **Dates are passed in, never read inside** — a clock inside left tests unable to catch a
  UTC-midnight off-by-one.
- **Standing guidance is embedded, never copied into projects.** `go-checklist` and
  `deployment` are practice, not documentation of the kit, and `audit` cannot check a word
  of either — a copy per repo would be drift with nothing to notice it.
- **Guidance must never use `- [ ]`**, which in a kit project means "a PR-sized unit of
  work"; source material using it for "a property to verify" is converted to prose. A test
  pins it.
- **Feature docs are agent context, and `doc-bloat` watches them at LOW.** 100 lines a doc
  and 5 a `## Log` entry, measured from this repo's 38 docs after the 0037 trim rather than
  guessed: median 63 and 2. Both are alarms — the fix line offers "say in the Log why it
  earns the length" — and `kit ship` asks for the compaction at the moment a working
  document becomes context, worded to lead with what to keep.
- **Shipped text has line ceilings** (`cmd/kit/budget_test.go`), because nothing else stops
  a doc growing and no reviewer counts lines. Tiered by how often it is read: the protocol
  tightest, then command help, then topics. Ceilings only — a floor rewards padding — and
  the message says to raise the number with a reason rather than just "shorter".

## Gotchas

Each of these has already caused a bug.

- **Files under `templates/` must not be valid Go package files or a nested module**, or
  `go vet ./...` compiles them and `go:embed` skips them. Hence `gomod`, `gitignore`,
  `main.go.tmpl`.
- **Templates are embedded through `assets.go` at the repo root** — `go:embed` cannot reach
  a parent directory from `cmd/kit/`.
- **Go's `flag` stops parsing at the first positional.** `parseArgs` loops it so
  `kit audit PATH --json` works; any new subcommand taking paths must use it.
- **The `## Work protocol` section of a project's `AGENTS.md` is regenerated** by `init`.
  Edits inside it are lost; durable house rules go above the marker.
- **`kit help checks` is generated from the registry `audit` reports from.** Add checks
  there, never as bare strings.
- **The installed binary is not the source.** `make install` after anything lands, or test
  against `./kit`. A real project once ran two features behind and nothing said so.
- **`go install` does not apply the Makefile's `-ldflags`**, so `resolveVersion` falls back
  to the module version in the build info; without it a tagged release reports `dev`.
- **`CGO_ENABLED=0` must not reach `make test`.** The race detector needs cgo on
  linux/amd64 but not darwin/arm64, so a file-level export passes locally and fails CI.
  A green `make check` on a Mac is not evidence about Linux.
- **Guidance the binary does not print may as well not exist.** An outside review reached
  four wrong conclusions, every one a fair reading of `kit help`. Two tests keep that shut.
- **`GOPRIVATE` is a comma-separated list, and a shell `export` beats `go env -w`.** Append
  to an existing company value; the `-w` write is inert when the variable is exported.

## Verifying

```sh
make check     # fmt-check, vet, unit tests, conformance suite — the whole gate
make vuln      # govulncheck against the module
make install   # go install ./cmd/kit
```

Builds are reproducible: `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w"`. CI runs the same
gate plus `golangci-lint`, `make vuln` and `go run ./cmd/kit audit . --strict` — `go run`
so it needs no credential.

**The conformance suite** (`cmd/kit/conformance*_test.go`) builds the binary once and drives
it against throwaway projects, asserting on stdout, stderr, exit codes and the files left
behind. In-tree because the previous suite was an external script, which is how it got
deleted without anything noticing. Three properties are load-bearing:
`TestEveryRegisteredCheckIsExercised` compares the cases against `kit.Checks` both ways, so
a check with no test fails the build; every audit case starts from the same clean project,
which must report nothing; and every test is independent — own `t.TempDir()`, no
`os.Chdir` or `os.Setenv`, binary built once behind `sync.Once` — which is what lets them
declare `t.Parallel()`. `PARALLEL` caps how many run at once, 4 of 12 cores by default
since the gate runs beside other work: 32.8s serial, 17.4s capped, 8.8s uncapped. Coverage
reads 4% — subprocess tests instrument nothing.

## State of play

Private repo, tagged `v0.2.0`, `go install …/cmd/kit@latest`; four projects run it.
