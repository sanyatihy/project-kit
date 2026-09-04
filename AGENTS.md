# project-kit

This repo is the kit itself. **Read this routing rule before anything else.**

- **Asked to bootstrap or set up a project at another path?** Follow
  [Bootstrapping another project](#bootstrapping-another-project). Change nothing in this
  repo, and ignore the `## Work protocol` below — it governs work on the kit, not on the
  project you were asked about. You do not need to be in this directory: `kit` is on
  `$PATH`.
- **Asked to change the kit?** See [Changing the kit](#changing-the-kit), and run
  `kit next` here for the current plan.

## Bootstrapping another project

```sh
kit init /path/to/project              # docs only
kit init /path/to/project --stack go   # also Makefile, go.mod, cmd/, CI
```

**Never delete or clear the target directory first.** `kit init` creates what is missing
and overwrites nothing you wrote, so it is safe on a directory that already has files and
safe to re-run. Emptying a target to "start clean" destroys work and is never required.

Then work **inside the new project**, not here. Run `kit next` there: it names the first
step, and the `## Work protocol` in *that* project's `AGENTS.md` governs the rest. Never
invent product decisions on the human's behalf — ask in `docs/INBOX.md`.

## Changing the kit

```sh
make check     # fmt-check, vet, lint, unit tests, conformance suite — the whole gate
make install   # go install ./cmd/kit
make release TAG=v0.3.0   # gate, then cut and push the tag `@latest` resolves to
```

**Run `make install` after anything lands on `main`.** `kit` on `$PATH` is one binary
shared by every project, so a merged change reaches none of them until it is reinstalled.
`docs/TECH.md` records what that cost once.

**Cut a release tag when a round lands on `main`.** An install resolves to the newest tag,
so a round that merged and was never tagged reaches nobody. This is a fact about a Go
module installed with `@latest`, not about every project, so it lives here rather than in
the protocol below.

Three constraints that are easy to trip over:

- Files under `templates/` must not be valid Go package files or a nested module, or
  `go vet ./...` compiles them. Hence `gomod`, `gitignore`, `main.go.tmpl`.
- Templates are embedded through `assets.go` at the repo root, because `go:embed` cannot
  reach a parent directory from `cmd/kit/`.
- A new check goes in the registry in `internal/kit/audit.go`, never as a bare string.
  `kit help checks` prints from it, and `TestEveryRegisteredCheckIsExercised` fails the
  build for a check with no conformance case.

<!-- The section below is written by `kit init` from the kit's own template. Edits inside
     it are lost on the next run; durable house rules go above this comment. -->
## Work protocol

Run `kit next` at the start of every session. It names the feature and the one task to do.
`kit audit` says what has drifted. `kit help` explains the model, and `kit help template`
prints the doc format.

### Planning

Do this only when you are asked to plan.

1. **Read `docs/VISION.md` first.** A feature that cannot be traced to it is the wrong
   feature. Split by `## What done looks like`: each observable outcome is at least one
   feature. `## Constraints` and `## What it is not` generate work as well as reject it —
   a constraint the project does not yet honour is a feature.
2. **A feature is one shipped change in what the project can do, can be trusted to do, or
   can be trusted not to do.** Adding a capability is one shape and not the privileged one:
   a failure that stops happening, an exposure that closes, a number that crosses a line, a
   capability removed with its cost. Each needs a value somebody would notice and a trigger
   that can be observed, the two questions the backlog asks before an idea is promoted.
   Removing what already shipped is a new doc citing the frozen one, never an edit to it.
   Name it in one sentence an operator would recognise: "the board no longer hands one
   feature to two agents", not "refactor `Mine`". No such sentence means it is a task box in
   another feature, or a chore commit that rides on a pull request under a branch name
   carrying no id. A separate doc earns its place when it holds a decision someone must read
   before building, that is not recoverable from the code or the commit. A change nobody
   could review in one sitting is too big — split it by value, not by layer.
3. **Draft the whole set in one session, not the first one only.** Write every doc with
   `kit new "<title>"`. Never copy the template by hand.
4. **Fill `needs:` on every draft.** List the feature ids that must ship before this one
   can start. Leave it empty when the work could start today — empty ones are what let
   agents run at once. `kit help template` says how to judge it.
5. **Put the whole round on one branch named `plan/<something>`, and merge it once.**
   The branch name must contain no four-digit feature id, anywhere in it. A branch name
   containing an id is a claim on that feature.
6. **Leave every doc `status: Draft`.** `kit ship`, `kit drop` and `kit reset` are the only
   things that change a status. Never edit `status:` or `shipped:` by hand — a value the kit
   does not recognise wedges the doc, and `kit reset <id> "<why>"` is what clears one.
7. **Never invent a product decision on the human's behalf.** What you cannot settle goes in
   `## Open questions`; what only a human can answer goes to `kit block <id> "<question>"`.

### Taking work

8. **Take a feature with `kit claim <id>`.** It pushes the branch that claims it and makes
   the worktree to work it in. Read the exit code:
   - **0** — the feature is yours. Go to the worktree it names and work there.
   - **1** — it is not yours. Run `kit next` and take a different feature. Do not retry.
   - **2** — the command could not run. Read the error, fix it, claim again.

9. **Work that started without a plan reaches the board with `kit harvest`.** A branch that
   is neither a claim nor a plan round is a spike: nothing routes it, and `kit next` says so
   rather than refusing you. When it is done, `kit harvest "<title>" --box "<outcome>"`
   takes an id, writes the doc and commits the tree in one commit; `kit harvest --nothing
   "<what it showed>"` is the spike that did not survive. It writes a `Draft`, so
   everything below still applies.

### Doing the work

10. **Read the feature doc for the design, and nowhere else.** It is the one home for it.
11. **Do the topmost unticked `## Tasks` box, and only that one.** List order is the order
    of work.
12. **Tick the box in the commit that implements it.** One box, one commit. Nobody re-reads
    the branch to check, so the commit is what has to be honest. The subject names the box;
    the body stays empty, because the diff shows what changed and the doc says why. A body
    earns its place only where the diff cannot show what it says — a revert and its cause,
    the source a cherry-pick came from.
13. **Discovered work becomes a new box, a new doc, or a `BACKLOG.md` line — never a `TODO`
    in the code.** Append it, unless it blocks the boxes below.
14. **A box that turns out wrong may be rewritten, split or deleted.** Own commit, reason in
    `## Log`. Never silently drop one you could not finish. A whole feature that should not
    be built is `kit drop <id> "<why>"`.
15. **Stopping mid-feature is `kit release <id>`.** It refuses while the branch carries
    work: that is a pause, so push the branch and open the pull request unfinished instead.
16. **Never put a real secret in a doc.** Name the variable and where the value lives, never
    the value. A pushed commit cannot be unpublished.
17. **Never edit a doc whose status is `Shipped`.** It is frozen history.

### Finishing

18. **When the last box is ticked, run this project's own gate, then `kit audit --strict`,
    then `kit ship <id>`.** In that order: `ship` freezes the doc, so anything the other
    two would have made you change has to be found before it.
19. **Push the branch, open the pull request, and stop. You never merge your own work.**
    Opening a pull request is not approving it. With no remote, say so and leave the work
    on the branch.

### Where two agents can still collide

**Run `kit audit` after a merge that brings in docs drafted with no remote.** `kit new`
reserves an id by pushing `refs/kit/ids/<id>`, so two agents drafting off one remote are
never handed the same number; with no remote there is nothing to reserve against and both
can write `0009`. They merge cleanly, so `audit` reports `duplicate-id` at HIGH. Fix it
before claiming either: give one doc a free id, rename its file, and rename its branch.

### How to write

- **Think before coding.** State your assumptions. Name the readings of an ambiguous
  request rather than picking one silently. Stop when confused rather than guessing.
- **Write the minimum code that solves the problem.** Nothing speculative, no abstraction
  for a single use, no feature beyond the box you are on.
- **Touch only what the task requires.** Do not improve adjacent code. Match the style
  already there.
- **Say what "done" looks like before you start, then verify it yourself.** A box whose
  truth you cannot check was too big.
- **Prefer a refusal in the tool to a rule in prose.** Before adding a rule here, ask what
  would make it unnecessary. Prose is the weakest tier and is honest only for judgement.
- **A sentence earns its place by deciding, constraining or measuring something.** One
  claim per sentence. Evidence gets one trailing clause, never two.
- **A bold lead-in never restates the sentence after it.** Cut it, not the sentence.
- **Write the decision, not the route to it.** "Changed JSONL for SQLite", never "we
  researched the append-only design, found it would not scale, and chose a database".
- **A non-goal is one line: what is excluded, and why.** "A database. None is needed at
  four hundred rows a month." Not an argument, and never a thing nobody would build.
- **Cite another feature by id; never restate its reasoning.** It has one home.
- **A comment says what the code cannot, and stops.** The contract, and the constraint the
  next editor would otherwise break — never what the code already says, and never the
  incident that produced it.
- **Every fact has one home.** True after this ships → `docs/TECH.md`, one line. Changed
  this doc's plan → one `## Log` entry, and a plan that never moved has an empty Log.
  Neither → the pull request.

### Where the rest lives

These rules are the judgement no refusal can make. Everything else the kit knows, it
prints: `kit help checks` for what `audit` reports, `kit help template` for the doc format,
`kit help backlog` for what a deferred entry looks like, and **`kit help go-checklist` or
`kit help python-checklist` for the practice this project's language is held to** — read
that before writing code, not after.
