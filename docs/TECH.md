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

`status:` is the autonomy dial, and it **orders the board rather than gating it**.
`next` offers `In progress`, then `Accepted`, then `Draft`; accepting says a human has
looked and blessed the design, which promotes the work rather than unlocking it. Four
rules govern the rest, and `kit help` prints them.

Mechanisms worth knowing before changing anything:

- **Ordering needs no syntax.** Task list order is dependency order; `next` returns
  the topmost unticked box. This replaced a dependency graph.
- **The claim is a branch name, and the push is what decides.** A branch citing a
  feature ID means that feature is taken. Zero new state, visible across worktrees
  and — once pushed — machines. `kit claim` reads `git push --porcelain` and treats
  only a *created* ref as a win, because a push to a branch the remote already has
  at the commit being pushed says "Everything up-to-date" and exits 0; trusting the
  exit code hands one feature to two agents. Creating the ref is a compare-and-swap
  server-side, so simultaneous claimants produce exactly one winner.
- **The checkout is the unit of isolation.** A worktree holds one feature. `kit next`
  reports the claim on its own HEAD as *yours* rather than taken, and will not offer
  a different feature to start in a checkout that is already claimed. Creating and
  removing worktrees is git's job and a launcher's; the kit only refuses to hand out
  work where there is already some.
- **The kit owns the work, not the runtime.** What to build, who holds it, how far
  along, what drifted. Where agents actually run — worktrees on disk, panes, starting
  and prompting them, noticing one is blocked — belongs to a runtime like `herdr`,
  which does all of it and persists across a reboot. Neither needs to know about the
  other: an agent composes them, `kit claim` then the runtime's own commands. This is
  what keeps the kit usable when the harness underneath changes, which is the one
  thing it can be sure of.
- **`needs:` is the only ordering the kit has, and it is declared, not inferred.**
  A list of feature ids in the frontmatter; a feature is blocked until every one of
  them is `Shipped`. ID order was considered and rejected — an id is an identity, so
  a prerequisite discovered late already has a higher one and the ordering becomes
  unrepresentable at the moment it matters. Shipped is the bar rather than merged,
  which leaves a short window where a dependency is done but not on the trunk;
  reading merge state instead would make the board depend on how fast somebody
  clicked. Parsing is lenient and the checks are strict: junk in the field is
  dropped so the board never changes shape over a stray comma, and `needs-unknown`,
  `needs-cycle` and `shipped-early` report the three ways the graph can be
  impossible. None of them judges whether the declared needs are the *right* ones —
  that is the same judgement call `audit` has always refused.
- **The ready set is the parallelism answer.** One ready feature means the work is a
  chain and three means three agents can take one each. Nothing declares that
  separately, so nothing can drift from it — it falls out of `needs:` and `Shipped`.
- **A branch holding only its feature doc is a proposal, not a claim.** Agents draft
  and leave the doc for a human, and nothing is committed to the default branch, so
  the proposal waits on a branch. `draft-claimed` compares the branch against trunk
  and fires only when something other than the doc changed. The fact was already in
  git, so neither the doc nor the tool needed a new field to carry it.
- **`kit next` reports ticked-over-total.** A count of boxes and never an estimate —
  boxes are not equal sizes. It is in `--json` as `done` and `total` because that is
  the interface anything watching the plan from outside has to read.
- **A feature is one pull request, and that is what decides the decomposition.** The
  agent pushes a branch and a human reads it, so the feature is the unit of review:
  too big is a PR nobody can review in one sitting, too small is one that changes
  nothing observable and exists only so the next can start. This was the missing
  tie-breaker — "one agent's reach" is satisfied both by seven features and by one
  covering a whole project, which is how the same prompt produced both. Boundaries
  come from the vision's `## What done looks like`, and bootstrapping means producing
  the whole set rather than the first one.
- **Three docs, three tenses, one owner each.** `docs/VISION.md` is what the project
  is becoming, `docs/features/` is what is being done now, `docs/TECH.md` is what
  exists. The vision and the as-built doc are both mutable and carry no status; they
  differ in that the as-built doc must be true *now* while the vision is allowed to
  be aspirational and is expected to move. `kit init` writes all of them and
  overwrites none. `no-vision` is LOW and fires only once a project has feature docs,
  since a project that has been set up and not yet planned has nothing to decompose.
- **A file is the kit's to refresh when nothing reads the project's copy.** That test,
  not who wrote it first, sorts the scaffold: `docs/features/TEMPLATE.md` and
  `docs/features/README.md` are read by nothing — `kit new` writes from the embedded
  template — so `kit init` replaces them on every run and they cannot describe a
  format the tool no longer produces. Everything a project fills in is the only copy
  there is and is never touched; house rules have their own space above the marker in
  `AGENTS.md`. A refresh that changes nothing reports as kept, or the word stops
  meaning anything. Schema versioning was considered and deferred: every format change
  so far has been additive, and the trigger to revisit is the first one that cannot be.
- **The feature template's guidance lives in HTML comments, and that is load-bearing.**
  `ParseSections` strips comments and `forEachBox` never sees them, so anything inside
  `<!-- -->` reaches the author and not the tool. Moving guidance out of comments would
  silently restore the worst behaviour this kit has had: with visible prose in every
  section and two `- [ ] …` boxes under `## Tasks`, a doc nobody had written parsed as a
  finished plan — `kit next` counted `0/2` and handed an agent `…` as the box to build,
  `kit audit` said the docs and the repo agreed, and the only finding went to the author
  who tidied the placeholder out of `## Log`. The gate was anti-correlated with the work.
  `unfilled-template` is HIGH on the back of that: because an unwritten section now
  parses as empty, the check cannot misfire, and `kit claim` does not consult `audit`, so
  `--strict` is the only thing standing between an unwritten doc and an agent acting on
  it. `missing-section` keeps only `## Log` and only at `Shipped`, since an Accepted
  feature has nothing to log yet.
- **Blocking is an INBOX line naming the feature.** `next` skips it immediately;
  `audit` escalates it after two days. Thresholds follow the cadence of what they
  watch, not an operator's calendar, and an idle agent costs minutes. Two days rather
  than one because a line carries a date and no time, so ages run from midnight UTC on
  the day of filing and a one-day fuse could burn out an hour after a question was
  asked.
- **The BLOCKING/FYI marker decides the severity, not just whether work stops.** An
  unanswered BLOCKING question is `inbox-stale` at HIGH; an undrained FYI note is
  `fyi-stale` at LOW after a fortnight. One severity cannot serve both: `--fyi` promises
  the feature is not blocked, and reporting it at HIGH broke that promise by way of the
  `--strict` gate the protocol puts in front of `kit ship` — a note explicitly cleared
  to wait stopped everything from shipping.
- **`audit` never writes.** An auditor that fixes things stops being read.
- **Accepting is a command, and only a human runs it.** `Draft -> Accepted` is the one
  transition that belongs to a person, and it was a hand edit — which is what `kit ship`
  exists to stop at the other end and what `unticked-shipped` and `shipped-no-date` are
  the wreckage of. `kit accept` refuses a non-Draft, a doc whose `## Open questions` is
  not empty, and one a BLOCKING inbox line names — the last via `BlockersFor`, so
  `accept` and `next` cannot disagree about what is blocked. It stamps no date:
  `created:` already says when the feature was proposed.
- **The gate is unresolved questions, not `status:`.** Accepting used to block every
  agent until a human transitioned a field, which added a stop before a review rather
  than a review. `kit claim` takes a `Draft`; what still refuses it is a BLOCKING inbox
  line and a doc nobody wrote.

  That second one is load-bearing. `unfilled-template` exempts `Draft`, and the
  exemption was only safe while a Draft could not be claimed — without a replacement,
  `claim` would hand out a doc straight from `kit new` with three empty sections and
  every command exiting 0. `Feature.Unwritten()` is the single answer to "nobody wrote
  the plan", shared by `audit`, `claim` and `next`, so the board never offers what the
  next command refuses.

  `draft-claimed` moved with the gate: it now reports a branch building on a doc whose
  `## Open questions` are open, which is the defect it was always reaching for. Left
  pointed at `Draft` it would have fired on the normal flow.
- **Drafting is a plan round, and drafts live on the trunk.** A session's docs go on
  one `plan/<something>` branch and merge once, at whatever statuses they reached. The
  branch is never named after a feature, because a branch leading with a four-digit ID
  *is* a claim — which is what made a draft on its own branch claim itself, and what
  0014 had to patch with a delete-the-branch rule that is now gone. `kit drop` exists
  because a doc on the trunk cannot be retired by deleting a branch.
- **`next` reports a stale protocol; `audit` alone was not enough.** The protocol tells
  agents to run `next` at the start of every session and says nothing about running
  `audit`, so a project following superseded rules had no moment at which anything would
  say so. It prints above the board — an agent that has read the plan and started work has
  walked past anything printed after it — and never changes the exit code, because the plan
  is still right and it is the rules around it that drifted. `--json` carries it as a field
  instead, since that is the format agents parse.
- **A status rewrite is confined to the frontmatter.** `(?m)^status:.*$` over the whole
  document also rewrote a `status:` line inside a fenced example, and feature docs for a
  tool whose subject is frontmatter quote frontmatter constantly. `SetField` edits only
  between the `---` fences and is shared by `accept` and `ship`, so neither can drift from
  the other.
- **Both output formats carry the same conclusion.** `Plan.Guidance` is decided once
  in `internal/kit` and rendered by the text output and carried by `--json`, so they
  cannot drift. `--json` is the format documented for agents, and it previously held
  a board with no instruction — and attached the next box to features it had reported
  as somebody else's. A feature that is taken or blocked now carries no `task` in
  either format: `task` means "the box to do now", which for work that is not on offer
  is a category error. Copyable commands stay out of the guidance sentence, because it
  is wrapped and a wrapped command cannot be copied.
- **A leftover branch is indistinguishable from a fresh claim.** Both are identical to
  the trunk, so no inspection can separate them, which is why `kit claim` names both
  possibilities instead of asserting a claim it cannot verify. It is also why a claim
  carrying no work is never aged: `stalled` would be inventing a date. Plan rounds
  removed the common source — a draft no longer sits on a feature-named branch — but a
  claim released by hand still leaves one.
- **Claims are read from the remote, not from `refs/remotes/*`.** Those refs are a cache
  of the last fetch and git prunes them on neither `fetch` nor `pull`, so a branch deleted
  when its pull request merged stayed there and read as a live claim — permanently, on
  every feature, in the exact loop the kit exists to run. `next` said to stop and wait,
  `claim` reported a lost race against a branch that existed nowhere, `audit` said the
  docs and the repo agreed, and the remedy printed was two commands that both error. One
  `ls-remote` per remote answers it: no objects transferred, nothing written, bounded by
  its own short timeout because `kit next ~/code/*` walks every project. `--offline`
  keeps the old path for a network that is not there, and the cost is stated rather than
  hidden. `claim` never takes it — a claim decided against a cache is not a claim.
- **A claim must carry the plan.** The push is `HEAD:refs/heads/<branch>`, so anything
  uncommitted does not travel with it. Without a check, claiming an uncommitted plan
  reported a won claim, pushed a branch holding nothing, and sent the agent to a worktree
  where `kit next` said the project was not set up — four steps, all exiting 0, and the
  agent's whole context gone. `git cat-file -e HEAD:<doc>` before the push costs nothing
  and leaves nothing behind.
- **Liveness comes from the claim branch, not from `status: In progress`.** That status is
  set on the feature branch and never reaches the trunk, so from the only checkout a human
  audits the doc reads `Accepted` — and `stalled`, the one check for an agent that died
  holding a feature, could not fire anywhere anyone was looking. The log is read with
  `--all` for the same reason: every commit in a pull-request flow is on a branch that has
  not merged. The doc's mtime is deliberately *not* a signal — for a tracked file it
  records when git last wrote it out, which a merge resets to now, so a freshly merged
  draft masked a claim that had been dead for a week.
- **A claim finding needs positive evidence, not absence.** The protocol puts a draft on a
  branch of its own and commits nothing to the trunk, so from the trunk the doc is *always*
  missing while a feature is in flight — and `orphan-claim` reported every open feature as
  a branch citing a doc nobody wrote, advising a fix that deletes the draft, while
  `no-plan` said the project had commits and no plan. Both now reason from what can be
  proved: a branch is an orphan only if its own tree can be read and confirms the doc is
  absent, and `no-plan` stands down when any claim-shaped branch exists. A branch nobody
  has fetched is passed over rather than guessed at — missing a real orphan until someone
  fetches is far cheaper than telling an agent to delete work it cannot see.
- **The trunk is asked for, not assumed.** `origin/HEAD`, then `main`, then `master`, then
  a HEAD that is not claim-shaped. Hard-coding the two conventional names meant a project
  whose default branch is neither had no trunk to diff against, so `BranchBuildsOn`
  answered "nothing changed" for every branch and every check built on it went silently
  quiet.
- **`protocol-stale` compares the project's work-protocol section with the kit's.**
  Shipping a protocol change does not deliver one: `AGENTS.md` is only rewritten by
  `kit init`, so a project runs whatever rules it was scaffolded with until someone
  re-runs it, and nothing inside that project can tell. The check is passed the
  expected text because `internal/kit` cannot embed a parent directory.
- **The protocol cites rules by name, never by number**, and a test enforces it.
  Two cross-references pointed at the wrong rules for months because later features
  inserted rules above them. Numbers in prose do not move when the list does.
- **Every run of `kit next` ends with an instruction.** Four states — something is
  yours, something is free, nothing is free, nothing exists — and each closes with
  what to do. The invariant is the point rather than any sentence: three vacuums have
  now been fixed the same way (who merges, where a draft lives, what to do when the
  board is full but none of it is yours), and each time an agent reached the end of
  the documented path, found nothing, and decided sensibly and wrongly. `next` also
  never tells an agent to accept a Draft — a tool that instructs past the project's
  own gate makes the gate advisory.
- **Shipping is a handoff.** `kit ship` ends the agent's part: push the branch and
  stop, because reviewing and merging are not the agent's. It says so at the moment
  of the decision rather than relying on the protocol alone, which is read at the
  start of a session and not the end. With no remote it says there is nowhere to
  hand off to — silence there is what let a first real run merge itself to `main`.
- **`secret-in-doc` matches shapes, not entropy.** Seven formats whose issuer
  publishes a fixed prefix or envelope, over the documents the repo model already
  opens — the feature docs, the inbox, the vision, the as-built doc, the protocol, and
  `BACKLOG.md`, which rule 6 makes as agent-writable as any of them. Not the source, not
  history. An eighth, a URL carrying userinfo, was
  removed after it failed a real project's doc for correctly explaining that such
  URLs are rejected: it was structural rather than issuer-anchored, so it never met
  the admission rule the other seven do. HIGH because `--strict` is the only lever and
  a published credential is the one finding that late discovery cannot fix. The
  finding names the shape and the line and never the value, or the report becomes a
  second leak. The patterns live in `audit.go` and nowhere else: written into a
  template or a help topic they would make the kit trip its own check, the same trap
  that keeps `- [ ]` out of the guidance topics.

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
- **The forge is detected at `init`, materialised as a file, and never consulted
  again.** `DetectForge` (`internal/kit/forge.go`) takes the first rung with positive
  evidence: the project's own `.gitlab-ci.yml` or `.github/workflows/`, then the
  remote's host, then the host segment `ModulePath` extracts from the directory
  layout — and only when that segment names a recognised forge, so an unrecognised
  host falls through instead of defaulting. `ciLayout` keys the CI file by forge;
  everything else in a stack is forge-independent.

  Unknown writes **no CI file** and says so with both ways out. A missing pipeline is
  visible and one for the wrong forge is not: a GitHub workflow in a GitLab repo never
  runs and no check can report it, because a repo mirrored to both forges carries both
  files legitimately.

  This is deliberately not configuration. There is no settings file and no per-host
  profile, so nothing can drift — the answer becomes a file on disk at init, and
  `next` and `audit` never detect anything. Per-instance CI values (runner tags,
  internal registry mirrors) are left in the generated file for the project to fill
  in, because they differ between two instances of the same forge.
- **One job for `audit`:** find where the docs disagree with the repo. Resist adding
  a second.
- **Prevent before detecting.** `kit block` and `kit ship` exist because writing an
  INBOX line and shipping a feature by hand were where four findings came from. A
  command that makes the mistake impossible beats a check that reports it, and
  `audit` stays the pressure valve rather than the destination.
- **A claim is a branch name, and only for live work.** Local branches, remote
  branches and worktrees all count, so a stale fetch hides someone else's claim.
  The claim checks apply only to features still being worked: once a feature is
  Shipped its branches are merged leftovers, and reporting those fires on every
  project where nobody deletes branches, which is most of them.
- **Blocking matches the id field of an INBOX line, never the prose.** A question
  that merely mentions another feature used to stop work on it, invisibly. A line
  outside the documented format falls back to matching the whole line, which
  over-blocks — the safe direction, since missing a blocker sends an agent into work
  a human has not cleared. `inbox-unparsed` makes those lines visible.
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
  property to verify" gets converted to prose bullets, or an agent following the
  topmost-box rule works through it as a task list. `TestNoTopicLooksLikeATaskList`
  pins this, exempting the topics whose subject matter *is* the feature-doc format.
- **Shipped text has line ceilings** (`cmd/kit/budget_test.go`), because nothing else
  stops a doc growing: a paragraph is easier to add than to justify, and no reviewer
  counts lines. Three tiers by how often the text is read — the protocol tightest,
  since it is written into every project and re-read every session, so its length is
  the one that compounds; command help next; topics loosest, because `go-checklist`
  and `deployment` are guidance whose substance is the point.

  Ceilings only. There is no minimum, because a length floor rewards padding and
  under-writing is already caught by `unfilled-template` and by review. The failure
  message names the rule rather than the number, and says to raise the ceiling with a
  reason — a test that only says "shorter" teaches half the rule.

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
- **Guidance the binary does not print may as well not exist.** An outside review
  reached four wrong conclusions about this tool, every one a fair reading of
  `kit help`: the INBOX format was enforced by the parser and documented nowhere
  reachable. `TestHelpAnswersWhatAnOutsideReviewGotWrong` and
  `TestEveryTopicIsReachableFromTheOverview` keep that shut.
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
