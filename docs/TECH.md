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
  protocol forbids. Both end states are terminal in both directions: `ship` refuses a
  Dropped doc as `drop` refuses a Shipped one.
- **A write needs one unambiguous doc.** `WriteTarget` refuses an id two docs declare and a
  key one doc declares twice; the first match `FeatureByID` returns is for reads, where
  refusing to render would help nobody.
- **Six frontmatter fields.** `id`, `title`, `status`, `created`, `shipped`, `needs`.
  `check:`, `checked:` and `review:` went with the commands that wrote them. `related:`
  went because 77 docs filled it in and no line of Go ever read it. The template writes no
  prose on any of them, and a test holds it to that: two spaces is all that separates a
  comment from a value, so guidance there is one keystroke from data. `needs-unparsed`
  reports what leaks anyway — a `needs:` is ids, commas and spaces, and anything else in
  it is not a dependency `needs-unknown` could ever have asked about.
- **What refuses a claim is a list, and it is stated in full wherever it is stated at
  all:** an unmet `needs:`, a BLOCKING inbox line, an unwritten `## Problem`, `## Design`
  or `## Tasks`, an `id:` disagreeing with the filename, a duplicate id, and a plan that
  has not reached the trunk. Saying "a doc nobody wrote and a BLOCKING line" as if that
  were the whole list taught readers that `needs:` does not gate claims. It does.
- **The claim is a branch name, and the push is what decides.** Only a *created* ref is a
  win: pushing to a branch the remote already holds at that commit prints "Everything
  up-to-date" and exits 0, which hands one feature to two agents. Ref creation is a
  server-side compare-and-swap, so exactly one claimant wins. The local branch is made
  before the push and deleted if it loses: a claim is held in both places or in neither.
  A name is read **whole**, namespace included, and from `for-each-ref` — `git branch`
  writes a page for a reader, and `(no branch, rebasing 0002-x)` read as a rival claim.
- **An id is handed out once, by the same compare-and-swap.** `kit new` pushes
  `refs/kit/ids/<id>` before writing the doc; only a *created* ref is the number, and the
  disk scan says where to start rather than what is free. The ref outlives the file, so
  deleting a doc no longer frees its id, and `kit init` reserves the ids a project already
  holds — nothing reports an unreserved one, because until that ran every doc on every
  board was one. The ref points at a commit naming the id and the title, over the empty
  tree asked of git because that value differs under SHA-256 — a backfilled one is the bare
  tree and records nothing. A project with no commits can still reserve; no remote means no
  reservation, and `kit new` says which guarantee it could not give.
- **A reservation is accounted for, or released.** `kit drop` on an id no doc holds deletes
  the ref instead of writing a status, and `audit` reports `orphan-reservation` at MEDIUM
  for one past the stalled horizon. Whether a doc holds it is asked of every branch's
  *tree*: a round that was drafted and reverted keeps its docs in `git log` for ever, and
  those are the ids there is anything to free. One `fetch` mirrors `refs/kit/ids/*` here,
  because `ls-remote` answers with SHAs and the record is in the object — a list that could
  not be refreshed reports nothing and releases nothing.
- **A claim carries the plan, branches from the trunk, and carries nothing else.** The doc
  must be in `HEAD`'s tree and in `trunkRef()`'s, or the claim is a child of the plan
  branch and only its own author can integrate it. The push is of `HEAD`, so a checkout
  the trunk does not contain sends another feature's commits under the new name: from the
  worktree that had just shipped 0001, `kit claim 0002` pushed 0001's work at exit 0.
- **The board offers only what a claim would accept.** `next` reported a feature as ready
  and `claim` exited 2 on it — where every new project stood at step 9. One condition read
  by both, `PlanLanded`, is what stops the two answers drifting apart again.
- **The checkout is the unit of isolation, and ownership is branch equality.** A worktree
  holds one feature, and its claim reads as *yours* while HEAD *is* the branch named after
  the doc — `notes-0001-alpha` mentions the id and confers nothing. A rebase detaches HEAD,
  so the name is read from git's rebase state instead.
- **One checkout takes one claim, and the branch name is what refuses.** `claim` refuses
  while HEAD's name cites any four-digit id: the ahead-of-the-trunk check sees a second
  claim only once the first branch carries a commit, so `kit claim 0008` inside 0007's
  worktree pushed both at exit 0.
- **A branch citing an id claims that feature, so `new` refuses to draft on one.**
  `plan/fold-in-0002-fix` made the board report 0002 as taken while `audit` exited 0. The
  draft is refused rather than `Claims` narrowed, which would shrink the guarantee; a
  feature's own claim branch is exempt, so a doc discovered mid-build still lands there.
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
  refuses outright when none of them resolves. A branch name strips *any* configured
  remote's prefix: with `origin/` alone, a clone whose remote is `upstream` resolved
  `upstream/main`, which matches no ref.
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
  the check for every feature. It leaves the groups above while it waits, or one finished
  feature reads as two.
- **You never merge your own work: legibility here, enforcement on the remote.** The board
  marks what the reader built — the claim it stands on, or the branch's authorship. The
  second half is worth exactly as much as the identities are distinct, and on this repo
  four agents shared the operator's, so it separated none of them. Branch protection is
  the only real enforcement while that is true.
- **An answer that cannot change inside one command is asked once** — the trunk, this
  checkout's branch, the remotes, the remote's branch list, every ref (`git show-ref`), and
  every feature doc's status on the trunk, and `HEAD`'s tree and the trunk's. Per feature
  instead, a 30-feature board read cost 116 git processes against 30 now, and 12 online. A memo whose answer can be empty
  needs a flag of its own: the trunk's used `""` for *not asked yet*, so a checkout where
  none resolves re-ran the ladder per caller — 747 processes against 28. Each is a ceiling
  the gate fails on, and each is measured, because a regression can leave the others
  identical. `make gitcount` prints them.
- **One project per invocation.** `next` and `audit` read the repo you are in and take no
  paths. The fan-out that read several at once keyed its payload by directory basename, so
  two projects called `proj` reported as one, silently, at exit 0 — and it had no `recover`,
  so a panic in one blanked the board for all. Four days across five projects invoked it
  zero times. A shell loop replaces it, and the two defects were deleted rather than fixed.
- **Claims are read from the remote, not from `refs/remotes/*`**, which git prunes on
  neither `fetch` nor `pull`, so a branch deleted on merge reads as a live claim for ever.
  That `ls-remote` also advertises each branch's tip, and the tip is the claim: a local
  head is only the last push. `release` refuses unless the tip is contained in the copy it
  deletes; progress, *last moved* and `stalled` read it, and one never fetched counts
  nothing rather than the trunk's zero. One `ls-remote` per remote, bounded at 15s — 5s is
  inside a working remote's latency (5.58s to GitHub over SSH) and the board silently
  printed nothing. `RemoteUnanswered` separates *asked and got nothing* from *nothing to
  ask*, because absence and ignorance render identically. `--offline` stays quiet, being
  an instruction; `claim` never reads the cache.
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
  a date and no time. An FYI line never blocks and is never aged: nothing is owed it.
- **The feature template's guidance lives in HTML comments, and that is load-bearing.**
  Comments are stripped before parsing, so guidance reaches the author and not the tool.
  Visible placeholder prose and two `- [ ] …` boxes make an unwritten doc parse as a
  finished plan, which `next` then hands an agent to build.
- **The plan checks gate on *committed*, never on a terminal doc, and read a boxless
  `## Tasks` as unwritten** — so a doc out of `kit new` stays quiet, `claim` refuses what
  the board declines, and a stub `drop` retired stops refusing every ship in the project.
- **Three docs, three tenses:** VISION what the project is becoming, `features/` what is
  being done now, TECH what exists. **`kit init` writes the whole document layout and
  nothing else creates a project file**, so there is one answer to where a file came from;
  `kit ship`'s as-built creation is a fallback for projects that predate it. Every document
  is the project's the moment it is placed, and only the protocol section is refreshed.
- **`kit init` writes an `AGENTS.md` with a title and the marker note.** It used to write
  the bare protocol section, opening on an `##` heading with no project name and nothing
  saying the section is regenerated — so the promise that house rules survive above the
  marker reached nobody.
- **Every fact has one home, and the routing is decided at ship.** Still true afterwards and
  it belongs here, one line; a change to the plan and it belongs in that doc's `## Log`,
  naming the change; neither and it belongs in the pull request.
- **A feature is one shipped increase in what the project can do, or can be trusted to do.**
  Named in one sentence an operator would recognise, or it is a task box or a chore commit.
  **A separate doc earns its place when it holds a decision someone must read before
  building, not recoverable from the code or the commit.** It has a cost on both sides; the
  "and" test it replaced had none and so only split, producing eleven docs where five were
  right. Too big is one nobody reviews in a sitting, split by value never by layer.
- **Every run of `next` ends with an instruction**, in every state. The newest was a doc
  straight out of `kit new`: it lands in `blocked`, and the closing line said stopping was
  the right outcome when the work was to write the plan just created.
- **`protocol-stale` compares the project's protocol section with the kit's**, because
  shipping a protocol change does not deliver one — `AGENTS.md` is rewritten only by `init`.
  It reports which side is behind: `kit init .` writes the binary's copy over the section,
  so offered to the newer side it reverts it, and the only project that can be the newer
  side is the one carrying `templates/AGENTS-section.md`. `no-protocol` covers a file whose
  `## Work protocol` heading is gone as well as a missing file; nothing else looks for it.
  **Rules are cited by name, never by number**, in every shipped string as well as in the
  protocol. A number is valid only inside the one list that carries it, and a refusal that
  cited one sent the reader to a list they were not holding.
- **`secret-in-doc` matches shapes, not entropy** — eleven formats whose issuer publishes a
  fixed prefix. HIGH, because a published credential is the one finding late discovery
  cannot fix. The patterns live in `audit.go` only; in a template they would trip the check.
- **Checks are a table, not a function.** `featureChecks` runs per feature and `repoChecks`
  once per project; each entry names the registry codes it can report, and a test fails the
  build if a registered check has no implementation or two claim one code.
- **`audit` has one job:** find where the docs disagree with the repo. It never writes — an
  auditor that fixes things stops being read — and it counts boxes rather than estimating.
- **A check that reports a state nobody may fix is deleted, not tuned.** Nine went that
  way: five demanded prose nobody owed, four reported a state `ship` refuses to create or
  a note `--fyi` cleared to wait. What a terminal doc is still asked is the frontmatter
  the kit writes — repairable without reopening the plan — and a leaked credential.
- **One defect is one finding; the code names the category, not one member.** Seven pairs
  merged, each survivor's `Means` restated. A missing `VISION.md` or `INBOX.md` is
  `no-protocol`: one command writes all three, so one fix clears them.
- **Severity is the damage.** `split-claim` is HIGH — two agents owning one feature is the
  worst failure here, and it gated nothing at MEDIUM. `id-mismatch` is HIGH because the
  branch comes from the filename and the claim is looked up by the id, so the board keeps
  offering a feature somebody holds. Nothing is LOW, so no level prints empty.
- **`audit` is silent only about what it checked.** A doc it cannot open is
  `unreadable-doc` at HIGH, carrying its filename and its still-reserved id, and no other
  feature check runs on it — they all fire on the empty struct. `--offline` names the
  checks it dropped, from `OfflineChecks`. A directory with no `docs/features` exits 2 from
  `next` and from `audit`: nothing was read, so there is no clean bill to give either.
- **Every git read that yields a path asks for `-z`.** git C-quotes anything outside
  printable ASCII, and each reader compares the result against a name `os.ReadDir`
  produced: one accented character disabled the plan checks on that doc, reported pushed
  work as unpushed, and named `ship`'s debris under a name nothing on disk has.
- **A fix line has to be one the reader can run.** `done-not-shipped` wants boxes before it
  calls a doc finished, and names the edit where there is no `id:` for `kit ship` to take.
  `shipped:` is validated — absent, unparseable and not yet arrived are one finding — and
  `tech-drift` ignores a day that has not happened, which nothing could be newer than.
- **`exec.CommandContext` kills git and nothing git started.** `git-remote-https` outlives
  it holding the stdout pipe, so `cmd.Output()` blocks for ever: `next` against a
  black-holed remote came back in 75s against the 15s its help promises, and in 15.05s
  once every git command set `cmd.WaitDelay`. The delay comes out of the bound, not on
  top of it.
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
- **One path per document kind.** Three feature directories and four spellings of the
  as-built doc were accepted for projects the kit never scaffolded, and `createTechDoc`
  hard-coded one of the four anyway — so a project using a different spelling got a second
  as-built doc rather than an updated one.
- **Exit codes are a contract**, pinned by a test: `0` worked, `1` it ran and the answer is
  no (`claim` lost the race, `audit --strict` found HIGH), `2` it could not run or refused.
- **Dates are passed in, never read inside** — a clock inside leaves tests unable to catch a
  UTC-midnight off-by-one.
- **`kit init` writes the whole layout, and nothing else creates a project file.** One
  command, one answer to where a file came from; `ship`'s lazy `TECH.md` path is a fallback
  for projects that predate it. Splitting creation across `init`, `ship` and `block` was
  tried and reversed — it moved four files between three commands to silence one finding,
  and that finding was a gate: `unfilled-vision` now waits for features, as `no-protocol`
  already did, so a virgin project audits clean while carrying everything it will need.
- **A `.gitignore` is a refusal expressed as a file**, so `init` writes one — editor and OS
  droppings only, because anything else guesses what the project is. A stack's copy replaces
  it rather than extending it.
- **What the protocol does not carry, it points at**, and `cmd/kit/layout_test.go` holds
  every file it names against `docLayout` or a help topic. Rule 14 named `BACKLOG.md` while
  nothing wrote or described it, and the first version of that test read only `docs/` paths
  and so found nothing to check.
- **Standing guidance is embedded, never copied into projects.** `go-checklist` is practice,
  not documentation of the kit, and `audit` cannot check a word of it — a copy per repo
  would be drift with nothing to notice it. It must never use `- [ ]`, which in a kit
  project means a PR-sized unit of work; a test pins that.
- **Shipped text has line ceilings** (`cmd/kit/budget_test.go`), because nothing else stops
  a doc growing and no reviewer counts lines. Tiered by how long the reader is expected to
  sit with it: command help tightest, then the protocol, then topics, then the two whole
  documents. Ceilings only — a floor rewards padding.
- **The binary names nothing it does not have.** Tests hold `help.go` against the check
  registry, against `main.go`'s dispatcher and against the topic map, both ways —
  `kit ship --help` advertised `doc-bloat` for a release after the check was deleted.
- **A refusal list is complete or it is not a list.** `cmd/kit/refusals_test.go` names every
  reason `ship`, `release` and `claim` turn work away and counts the refusals in the source,
  so a new one reaches neither the code nor the help alone. The protocol points at
  `kit ship --help` rather than carrying a second copy.
- **`--json` is a contract and its shape is pinned.** The board's four groups are `[]` when
  empty, `moved` is absent rather than a year-1 timestamp — `omitempty` does nothing for a
  struct — and a directory nothing surveyed reports through `error` at exit 2.
- **A claim's worktree lives in the project, at `.worktrees/<branch>`.** The parent stays a
  list of repositories, and a dot directory is skipped by ripgrep and the Go toolchain on the
  name alone. The printed path is absolute, anchored at the main worktree, so it is right
  from a worktree as well as from the trunk. `claim` writes `.worktrees/` to
  `.git/info/exclude` rather than asking, because declining the exclude is not legitimate and
  `git clone` carries none — which puts the write in the command that runs in every clone.
- **A release is part of shipping, because `@latest` is whatever the newest tag says.**
  `make release TAG=…` gates, cuts an annotated tag and pushes it, refusing a dirty tree or
  a HEAD that is not `origin`'s default-branch tip; `audit` reports `tag-stale` when the
  newest tag falls further than `TagStaleCommits` behind the trunk.

## Gotchas

Each of these has already caused a bug.

- **Files under `templates/` must not be valid Go package files or a nested module**, or
  `go vet ./...` compiles them and `go:embed` skips them. Hence `gomod`, `gitignore`,
  `main.go.tmpl`.
- **Templates are embedded through `assets.go` at the repo root** — `go:embed` cannot reach
  a parent directory from `cmd/kit/`.
- **The frontmatter closes on a line of exactly three dashes, is written only inside those
  bounds, and takes a comment only after two spaces.** Each is a bug already had: a setext
  underline closed the block early, a document-wide replace stamped the ship date into a
  `title:` quoting a status, and `\s+#` cut `title: Fix #123 crash` down to `Fix`.
- **Go's `flag` stops parsing at the first positional.** `parseArgs` loops it so a flag
  after an argument still parses. It also writes usage to the stream it writes a
  bad-flag error to, so `newFlagSet` picks the stream from the args: help asked for is
  output, help printed after an error is a diagnostic.
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
- **`GOPRIVATE` is a comma-separated list, and a shell `export` beats `go env -w`.** Append
  to an existing company value; the `-w` write is inert when the variable is exported.

## Verifying

```sh
make check     # fmt-check, vet, lint, unit tests, conformance suite — the whole gate
make vuln      # govulncheck against the module
make install   # go install ./cmd/kit
make release TAG=v0.3.0   # gate, then cut and push the tag `@latest` resolves to
```

Builds are reproducible: `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w"`. **CI runs the
Makefile's `CI_STEPS` and nothing else**, and `cmd/kit/ci_test.go` fails when the workflow
drifts from it. **The linter is pinned and its set committed** to `.golangci.yml`, or a
release changes what the gate means with no commit behind it; `make lint` skips when it is
absent, which is safe only while CI installs it.

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

**No coverage number is reported.** `-cover` measured the test process, not the binary the
suite drives — 2.9% for a package covered end to end — and no threshold reads the result.

## State of play

Private repo, tagged `v0.3.0`, `go install …/cmd/kit@latest`; four projects run it.
