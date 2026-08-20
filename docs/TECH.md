# TECH

The kit as built: the facts that outlive the feature that produced them. Feature docs in
[features/](features/) carry what is planned and what was decided while doing it; this
carries what is true now. It is read before every change, so it stays short — one fact,
one line, and the evidence only where it stops the rule being reversed. How a fact was
found stays in the feature doc that found it.

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
around deliberately. Measured: the prose rule against merging your own work was broken
twice by an agent that had read it, and the coded refusal that replaced it was obeyed the
first time it fired.

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
  so `claim` names both possibilities and `stalled` never ages a claim carrying no work. A
  name is read **whole**, namespace included: on the last path segment alone, a worktree
  reads its own checkout as a rival claim and the fix line deletes the work.
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
  remote's trunk carries no copy of the doc saying Shipped. **Status, never presence** — a
  doc reaches the trunk when it is *drafted*, so testing presence silences the check for
  every feature. And on **any** branch the repo counts as a trunk, since a clone whose
  `origin/HEAD` predates a default-branch rename points at one nothing merges to.
- **You never merge your own work: legibility here, enforcement on the remote.** The board
  marks what the reader built (the claim it stands on, or the branch's authorship) and what
  `review: human` reserves for a person — written by `kit accept --review`, read from the
  checkout, any non-empty value counting. Branch protection is the floor and the only real
  enforcement; none of this distinguishes anybody while agents share one credential.
- **An answer that cannot change inside one command is asked once** — the trunk, this
  checkout's branch, the remotes, the remote's branch list, every ref (`git show-ref`), and
  every feature doc's status on the trunk, which `ls-tree` and one `cat-file --batch` read in
  two processes. Per feature instead, a 30-feature board read cost 116 git processes against
  29 now, and 10 online. **Both are ceilings the gate fails on**, and both are measured,
  because a regression can leave the offline number identical. A measurement nobody is
  required to read is the prose tier. `make gitcount` prints them; raising one is allowed and
  the failure says so, but a count that grows with the board is the bug.
- **The board is read concurrently, capped at 8, in the order the paths were given.** What a
  board read spends is one `ls-remote` per project — 2.3s against GitHub over SSH where the
  rest is under a tenth of that — so a fleet costs one round trip rather than one each: four
  projects went from 11.2s to 2.5s. The round trip starts before the local walk for the same
  reason. The cap bounds connections to one forge, not CPU, and the order is what lets one
  board be compared with the last.
- **Claims are read from the remote, not from `refs/remotes/*`**, which git prunes on
  neither `fetch` nor `pull`, so a branch deleted on merge reads as a live claim for ever.
  One `ls-remote` per remote, bounded at 15s — 5 is inside a working remote's latency (5.58s
  to GitHub over SSH) and the board silently printed nothing. Expiry reported as an empty
  page is worse than a slow one, because absence and ignorance render identically, so
  `RemoteUnanswered` separates *asked and got nothing* from *nothing to ask*. `--offline`
  stays quiet, being an instruction; `claim` never reads the cache.
- **Liveness comes from the claim branch, and there is no status for it.** A status set on
  the feature branch never reaches the trunk, which is the only checkout anyone audits, so
  the doc there reads `Accepted` and `stalled` cannot fire. The log is read `--all` for the
  same reason; a doc's mtime is not a signal, since a merge resets it.
- **A claim finding needs positive evidence, not absence.** From the trunk an in-flight
  feature's doc is *always* missing, so absence would report every open feature while
  advising a fix that deletes the draft. A branch is an orphan only if its own tree says so;
  an unfetched branch is passed over rather than guessed at.
- **The trunk is asked for, not assumed:** `origin/HEAD`, then `main`, then `master`, then a
  HEAD that is not claim-shaped. Hard-coding the two names leaves a project whose default is
  neither with nothing to diff against, and every check built on it goes quiet.
- **Drafting is a plan round, and drafts live on the trunk.** One `plan/<something>` branch
  per session, merged once, at whatever statuses its docs reached — and its name must not
  mention a feature id *anywhere*, since any branch citing one is a claim and such a draft
  claims itself. Not only the first segment: `plan/fold-the-fix-into-0039` is a claim on
  0039. A branch carrying only its doc is a proposal, which is what `draft-claimed` diffs
  for. `kit drop` exists because a doc on the trunk cannot be retired by deleting a branch.
- **Accepting is a command, and only a human runs it.** It refuses a non-Draft, an unemptied
  `## Open questions`, and a doc a BLOCKING line names — via `BlockersFor`, so `accept` and
  `next` cannot disagree about what is blocked.
- **Blocking is an INBOX line naming the feature in its id field**, never the prose, which
  stops work invisibly; an unparseable line matches whole-line, which over-blocks — the safe
  direction. `next` skips at once, `audit` escalates after two days, because a line carries a
  date and no time. BLOCKING and FYI differ in severity too — HIGH and LOW — since `--fyi`
  promises the feature is not blocked, and HIGH breaks that promise through the `--strict`
  gate in front of `ship`.
- **The feature template's guidance lives in HTML comments, and that is load-bearing.**
  Comments are stripped before parsing, so guidance reaches the author and not the tool.
  Visible placeholder prose and two `- [ ] …` boxes make an unwritten doc parse as a finished
  plan, which `next` then hands an agent to build. `unfilled-template` is HIGH on the back of
  that, and cannot misfire because an unwritten section parses empty.
- **Three docs, three tenses:** VISION what the project is becoming, `features/` what is
  being done now, TECH what exists. **A file is the kit's to refresh when nothing reads the
  project's copy** — `TEMPLATE.md` and `features/README.md` are written from the embedded
  copy, so `init` replaces them and they cannot describe a format the tool no longer
  produces. Anything a project fills in is never touched; house rules go above the marker.
- **Every fact has one home, and the routing is decided at ship.** Still true afterwards and
  it belongs here, one line; a change to the plan and it belongs in that doc's `## Log`,
  naming the change; neither and it belongs in the pull request. Written twice is two facts
  to keep true, and a `## Log` that records every run is the transcript of the work rather
  than the result. The protocol states it, `kit ship` prints it where the choice is actually
  made, and `session-narration` catches at LOW the shapes of the failure a grep can see.
- **A feature is one pull request, and that decides the decomposition.** Too big is one
  nobody reviews in a sitting; too small changes nothing observable. Not "one agent's
  reach", which fits seven features and one whole project alike. Boundaries come from the
  vision's `## What done looks like`.
- **Every run of `next` ends with an instruction**, in all four states: an agent that reaches
  the end of the documented path decides sensibly and wrongly. `Plan.Guidance` is decided
  once and carried by both formats; a taken or blocked feature carries no `task`.
- **`protocol-stale` compares the project's protocol section with the kit's**, because
  shipping a protocol change does not deliver one — `AGENTS.md` is rewritten only by `init`.
  **Rules are cited by name, never by number**, with a test, because a later feature shifts
  the list and nothing else notices.
- **`secret-in-doc` matches shapes, not entropy** — seven formats whose issuer publishes a
  fixed prefix. HIGH, because a published credential is the one finding late discovery
  cannot fix. The patterns live in `audit.go` only; in a template they would trip the check.
- **Checks are a table, not a function.** `featureChecks` runs per feature and `repoChecks`
  once per project; each entry names the registry codes it can report, and a test fails the
  build if a registered check has no implementation or two claim one code. As one 362-line
  function, no check could be read or tested alone.
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
  INBOX line and shipping by hand produced findings a command makes impossible.
- **The commands refuse exactly what the board refuses** — `Survey`, `ClaimFeature` and
  `Ship` share `UnmetNeeds`, and `Ship` reuses `Repo.Mine`. Separately, `next` withholds a
  feature whose needs have not shipped and `claim` hands it over.
- **An `id:` disagreeing with its filename is HIGH, not tidiness.** The branch comes from
  the file and the claim is looked up by the id, so the board keeps offering a feature
  somebody holds.
- **One unreadable path must not blank the fleet board.** Failures are reported in place; a
  bad path exits 2, outranking `--strict`'s 1, because a project you asked about and did not
  get is the worse answer. Hence `audit --json` is `{"findings": […], "error": …}` — a bare
  list has nowhere to say why it is empty.
- **`unfilled-vision` closes the gap `unfilled-template` did, one file over:** `init` writes
  a stub, so a presence check fires only when the file is missing and the run that creates
  the stub silences it.
- **Exit codes are a contract**, pinned by a test: `0` worked, `1` only ever means
  `--strict` found something HIGH, `2` means the command could not run.
- **Dates are passed in, never read inside** — a clock inside leaves tests unable to catch a
  UTC-midnight off-by-one.
- **Standing guidance is embedded, never copied into projects.** `go-checklist` and
  `deployment` are practice, not documentation of the kit, and `audit` cannot check a word
  of either — a copy per repo would be drift with nothing to notice it.
- **Guidance must never use `- [ ]`**, which in a kit project means "a PR-sized unit of
  work"; source material using it for "a property to verify" is converted to prose. A test
  pins it.
- **Feature docs are agent context, and `doc-bloat` watches them at LOW.** 150 lines, an
  alarm rather than a target: the fix line offers "say in the Log why it earns the length",
  and `kit ship` asks for the compaction at the moment a working document becomes context.
- **The writing rule is per sentence, and the protocol owns it.** A sentence earns its place
  by deciding, constraining or measuring something, and evidence gets one clause. README,
  `kit help conventions` and the VISION and TECH templates cite it rather than restating it.
- **Enforce length per sentence, not per document.** A document ceiling is cleared most
  cheaply by cutting a paragraph; a sentence ceiling, by splitting, which deletes nothing.
  `long-sentence` at LOW, 45 words, over the docs rule 11 leaves editable.
- **Calibrate a doc threshold against a corpus written to the standard you want** — this
  repo just after a trim only confirms the trim. The corpus has to be checked for the
  standard it was actually written to: `localcode`'s were R&D working notes, so its `## Log`
  length was the defect and not the target, and only its length limits survive.
- **The words go in the wrong sections, which is what "too verbose" meant** — shorter docs
  overall, spent arguing the work rather than recording the decision. The `## Problem` ≥
  `## Design` ratio separates a corpus that argues from one that decides exactly, and is in
  BACKLOG awaiting a corpus written under the routing rule.
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
- **A narration marker is matched across the line wrap**, two lines at a time: the docs are
  hard-wrapped and every marker is several words, so a line-at-a-time scan misses whichever
  ones the wrap happens to split.
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
behind. In-tree, because an external script is deleted without anything noticing. Three
properties are load-bearing. `TestEveryRegisteredCheckIsExercised` compares the cases
against `kit.Checks` both ways, so a check with no test fails the build. Every audit case
starts from the same clean project, which must report nothing. Every test is independent —
own `t.TempDir()`, no `os.Chdir` or `os.Setenv`, binary built once behind `sync.Once` —
which is what lets them declare `t.Parallel()`. `PARALLEL` caps how many run
at once, 4 of 12 cores by default since the gate runs beside other work: 32.8s serial, 17.4s
capped, 8.8s uncapped. Coverage reads 4% — subprocess tests instrument nothing.

## State of play

Private repo, tagged `v0.2.0`, `go install …/cmd/kit@latest`; four projects run it.
