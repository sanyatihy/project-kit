# TECH

Current contracts and implementation constraints. Plans live in [features/](features/);
command usage lives in `kit help <command>`.

## Structure

One Go binary; git is the only runtime dependency. Markdown and git hold all state.
There is no database, queue, service or runtime orchestration.

```
cmd/kit/       CLI dispatch, help, rendering and conformance tests
internal/kit/  feature parsing, git operations, lifecycle, board, audit, scaffolding
assets.go     embeds templates from the repository root
templates/   project documents, help topics and Go/Python scaffolds
```

`internal/kit` is one package, split by subject. All git subprocesses go through git.go.

## Enforcement

Prefer remote permissions, then command refusals, then prose. Agents can bypass kit
with raw git. A printed recovery command must preserve the refusal's guarantees:
never tell a losing claimant to delete the winner's branch. Runtime panes and restarts
belong to the agent runner, not kit.

## Documents and lifecycle

- One feature per `docs/features/NNNN-name.md`. Fields: id, title, status, created,
  submitted, needs. Frontmatter carries data only; guidance stays in HTML comments.
- Lifecycle: Parked ↔ Draft → Submitted; Dropped retires unbuilt work. Submitted on a working
  branch means awaiting review; Submitted on trunk means integrated and frozen.
  Reopen returns an unmerged submission to Draft and clears its date.
- Park and activate commit only the complete, unstarted target on a planning branch.
  They reject claims, dirty target files and unreadable remote state, and require the
  current trunk as an ancestor. Retries after a committed transition are no-ops.
  Parking preserves plan content; activation needs merge before claims become eligible.
  Parked remains document-validated, appears separately on next with its Log reason,
  and does not age as stalled work. It satisfies no implementation dependency.
  Next retains observed ownership; parked-claim fails strict audit when a claim races
  parking review. Unknown integration is reported without guessing shared availability.
- Legacy Shipped/shipped: remains readable; ship/reset alias submit/reopen with the same
  guards. Invalid-status repair requires `--repair`; Dropped cannot be reopened.
- Submit, drop, reopen, park and activate own status changes and commit them. Submit requires a clean
  tree and the feature's branch. A failed status commit names the file to preserve.
- Mutations require an unambiguous document: duplicate ids or frontmatter keys refuse.
  Read-only lookup may return the first match so diagnostics remain available.
- Next, claim and submit share plan validation. Malformed dependencies, cycles and
  ambiguous plans block work; submit also rejects blockers and unwritten sections.
  Problem, Design and at least one Tasks box are required to build. Comments and fenced
  examples do not count. Plan-completeness audits apply to committed, nonterminal docs.
- Task order is execution order; the first unticked box is next. One box is one commit.
  `needs:` declares prerequisites, satisfied only by Submitted documents on trunk and
  in the current checkout. Submission on a work branch does not unlock dependents.
  Shared files alone do not create dependencies; there is no global concurrency cap.
- A question is an unticked box under the feature's own `## Open questions`, so the doc
  is the id and nothing scans prose for one. Every open box blocks; a plain bullet is not
  a box and blocks nothing. Only the leading date is parsed, and an undated box blocks
  without ageing. `block` writes and commits under a lock beside the doc, replacing the
  file by rename: appending to a section is read-modify-write, and concurrent writers
  otherwise lose questions and expose a half-written doc. It refuses a terminal doc,
  which cannot be claimed and is frozen on trunk. Section writes locate a heading through
  the same fence and comment masking the parser uses, so a heading quoted in an example
  never receives one. Answer in the plan, tick the box and commit both before resuming. `docs/INBOX.md` is retired; `inbox-retired`
  reports one that survives, because `init` cannot delete what it did not write.
- VISION holds direction; TECH holds current facts; features hold plans; Log holds plan
  changes; PRs hold validation and handoff. Log may be empty; length is not a gate.
  Integrated feature history is frozen. The one-time prose compaction is recorded in 0073.

## Guided work

- Start records mode, outcome and base in an empty commit's Kit-Intent trailer. Plans
  use plan/<round>; direct work uses spike/<outcome>. Approved ids reuse claim guards.
- With no history, start creates and publishes an empty main through exclusive ref
  creation. The index and working files remain on the first review branch. Existing
  history, conflicting branches and unknown remote state refuse unsafe setup.
- Plan finish validates complete Draft or Parked plans with unticked tasks, commits only approved
  setup/document paths, audits and pushes. Implementation changes require their own route.
- Work finish harvests recorded direct work once, requests missing Problem/Design facts,
  and reuses a partial harvest after interruption, then requires complete tasks and a
  clean candidate. Planning branches cannot be harvested to bypass approval. Gate argv runs directly with a
  30-minute bound; failure, changed HEAD/branch or dirty output stops publication.
  Submission may change only lifecycle fields; hook-created commits or other edits refuse publication.
  Revisions after submission require reopen. Retrying finish reruns validation.
- Finish supplies pr_title from the feature title or Plan: <round>; next repeats the
  feature title for pending review. Agents set the PR title explicitly instead of
  commit autofill, which can select the claim receipt.
- Handoffs report the advertised base, candidate and confirmed publication. They do not
  certify PR creation, approval or merge authority. Disposable pre-receive fixtures
  test denied trunk writes; live hosted permissions still require authenticated evidence. Agents use their forge interface,
  then continue independent approved work from an updated trunk checkout.
- Next adds an action containing actor, kind, checkout and optional argv. Work merged
  as Draft with a retained claim produces merged-before-submit; unchecked tasks survive.

## IDs, claims and recovery

- New reserves `refs/kit/ids/<id>` remotely before writing the doc. Exclusive ref creation
  allocates the id; the disk scan only chooses a starting point. Reservations outlive docs.
- Reservation objects name the id, title and date over git's empty tree; legacy backfills
  may point directly to that tree. Ask git for its hash to support SHA-256 repositories.
  Init backfills existing ids. No remote means local allocation without cross-clone safety.
- Reservation reads fetch and mirror their namespace once per repo read. Disposal checks
  local and advertised branch trees, the age horizon and readable remote evidence, then
  leases the observed ref. Deleted or reverted historical docs alone do not hold an id.
- New claims use `work/<id>`, independent of title and filename. Legacy filename branches
  remain valid claims and prevent a second acquisition. Writers upgrade together.
- Acquisition pushes a unique empty receipt commit. `refs/kit/claims/<id>` preserves the
  attempt; `refs/kit/owned/<id>` preserves confirmed ownership across rebases. An unknown
  push outcome retains evidence; retry reconciles it with the remote.
- Only a confirmed competing winner permits cleanup of an unchanged, unoccupied losing
  attempt. A worktree failure cannot undo a won claim. Receipt commits and their empty
  rebased equivalents are bookkeeping, not product work.
- Claim requires a committed plan on trunk and HEAD equal to the advertised trunk tip
  online, or local trunk offline. Board eligibility uses the same PlanLanded predicate.
- Any branch name citing a four-digit id is a claim. One feature per checkout; new refuses
  a foreign claim branch. Planning uses `plan/<round>` without an id. A spike is neither
  trunk, a plan branch nor a claim; harvest assigns its id and feature branch afterward.
- Harvest commits the whole working tree with completed task boxes. With no history it
  first creates an empty trunk review base; staged and unstaged product files stay on
  the feature branch. Existing claims and detached HEAD refuse harvest.
- A won claim uses `.worktrees/<branch>` under the main worktree and adds `.worktrees/`
  to `.git/info/exclude`. An existing matching worktree is success; other files are preserved.
- Ownership for submit is exact branch equality, including the branch named by rebase
  state. Release instead requires a local branch created by this clone; a remote ref alone
  proves neither ownership nor permission to delete another agent's work.
- Release refuses uncommitted files and unmerged commits, including doc-only changes.
  `--discard` permits unmerged commits, never foreign ownership or dirty worktrees. The
  remote delete leases the observed tip. Current and custom worktrees are kept detached;
  local ref deletion leases the checked tip. Incomplete cleanup names a kit release retry.
  The local branch must contain the advertised remote tip before deletion.
- Reopen checks the published trunk before changing a submission; missing objects or
  unknown integration evidence refuse. Reopen, submit and drop reject foreign claims.

## Board, audit and git

- Next and audit read one repository, from any subdirectory. Every next state includes
  an instruction, shared by text and JSON. Missing `docs/features` exits 2.
- Claims and review states use advertised remote branch tips, not stale remote-tracking
  refs. Board progress and last-moved use a local head when it contains the advertised
  tip; behind or diverged heads keep the remote answer, and unfetched objects give no
  count. Other checkouts' uncommitted ticks are excluded. Review and audit retain the
  published anchor, so an unpushed submission does not enter awaiting review.
- Board items carry an optional `worktree` path relative to the surveyed repo root,
  from the batched NUL-delimited worktree listing. Taken entries print it; guidance
  counts held claims and names `kit release <id>` for an absent agent. A checkout path
  does not grant ownership or route another agent into that checkout.
  Read liveness from git history across branches, never document mtimes reset by merges.
  Orphan claims require positive evidence from the branch's own tree.
- Review distinguishes local revision, unpushed submission, published review and
  integration, including squash merges and retained feature branches.
- Author identity and current branch can label one's own work, but shared credentials
  defeat identity separation. Humans merge; remote permissions and required checks
  enforce that boundary against raw git. Kit can enforce only its own commands.
- Failed remote reads label cached claims and unknown handoffs. Explicit offline output
  labels its reduced coverage (`offline: true` in JSON). Neither establishes remote absence.
  Strict online audit fails on an unanswered remote; offline audit names omitted checks.
- Audit leaves project files unchanged; online reservation checks may refresh refs.
  Help is generated from the registry in audit.go.
- **Checks are a table, not a function.** `featureChecks` runs per feature and `repoChecks`
  once per project; each entry names the registry codes it can report, and a test fails the
  build if a registered check has no implementation or two claim one code.
- One defect produces one actionable finding. Severity reflects damage; age is advisory.
  Delete checks for states nobody may repair. Missing VISION and AGENTS.md share
  no-protocol; question checks skip frozen docs, whose questions nobody may correct;
  split-claim and id-mismatch are HIGH. Keep issuer patterns in audit.go, never templates.
  `needs-unstated` is LOW because a citation alone cannot prove a dependency. It reads
  Problem and Design only for nonterminal plans. Secret checks use issuer-specific shapes.
- Unreadable docs produce HIGH findings and skip other feature checks. Terminal docs
  retain frontmatter and secret checks. Frozen-doc compares against published trunk
  online or local history offline; deleted docs and unreadable evidence remain findings.
- Status-uncommitted compares working documents with HEAD using batched tree reads.
  Submitted dates must be present, parseable and not in the future; tech-drift ignores
  future dates. A completion finding requires actual Tasks boxes.
- JSON fields keep their types and meanings: empty board groups are `[]`; unavailable
  moved timestamps are omitted; read failures populate error and exit 2. Additive fields
  are allowed; human-readable wording is not a parsing interface.
- Exit codes: 0 success; 1 completed negative result (lost claim or strict HIGH finding);
  2 refusal or incomplete operation. An interrupted claim is not a confirmed loss.
- Git path output uses `-z`. Git reads are batched; adding unclaimed features must not add
  subprocesses. Remote listing and reservation refresh each run at most once per read.
  The process-growth comparison uses 30- and 130-feature boards with identical claims.
- **`exec.CommandContext` kills git and nothing git started.** `git-remote-https` outlives
  it holding the stdout pipe, so `cmd.Output()` blocks for ever: `next` against a
  black-holed remote came back in 75s against the 15s its help promises, and in 15.05s
  once every git command set `cmd.WaitDelay`. The delay comes out of the bound, not on
  top of it.
  Remote calls are bounded at 15 seconds. A 5s timeout excluded a working SSH remote
  measured at 5.58s. Inject dates into logic.

## Scaffolding and parsing

- Init exclusively creates missing files, preserving existing files and dangling symlinks.
  It owns the document layout; submit's TECH creation is a legacy fallback. Only the
  Work protocol section is refreshed; house rules and following sections survive.
  Virgin projects audit clean: unfilled-vision and no-protocol wait for features.
- AGENTS.md has a project title and refresh marker; CLAUDE.md is its symlink. Protocol
  detection ignores quoted markers, fences and HTML comments. Staleness reports which
  side is newer; never refresh a newer project protocol from an older binary.
- Stack metadata declares files, entry-point globs, keep directory and pipelines. Python
  uses `src/*/__main__.py`; generated entry-point tests run outside the source tree.
  Existing applications receive neither those tests nor an interpreter pin.
- Forge selection uses explicit choice, existing CI, remote host, then project path.
  Unknown forge writes no CI and reports it. Stack scaffolds include their gitignore;
  docs-only init ignores editor and OS debris. Deployment scaffolding is excluded.
- Templates cannot be valid Go packages or nested modules: use gomod, gitignore and
  main.go.tmpl. Embed from assets.go because go:embed cannot reach a parent directory.
- Frontmatter closes only on a line exactly `---`; writes stay inside it. Trailing comments
  require two spaces before `#`. Backtick and tilde fences hide headings and task examples.
  Preserve `title: Fix #123 crash`; a setext underline must not close frontmatter.
- CLI parsing accepts flags after positional arguments. Requested help uses stdout;
  bad-flag usage uses stderr. Shipped references cite rule names, not list numbers.
- Help topics and protocol paths must resolve through embedded content or scaffolded files.
  Checklists are review aids, not executable task boxes. Tests protect reachability,
  parsing and refusals, not prose length or wording counts.
  Feature outcomes and splitting rules live in the unchanged AGENTS protocol. Optional
  check/checked predictions were removed after zero of 82 docs used them; do not restore ceremony.

## Verification and release

```sh
make check    # fmt-check, pinned lint/vet, race tests and conformance
make gitcount # git subprocess growth across board sizes
make vuln     # pinned vulnerability scanner
make audit    # strict online audit using this checkout's source
make install  # reinstall the shared kit binary after merge
make release TAG=v1.0.0  # complete gate, validated trunk tag, verified push
```

CI runs Makefile CI_STEPS: check, vuln, audit. Missing or failed checks fail the gate.
The Go linter is pinned at v2.12.2; govulncheck at v1.7.0. Scaffolded CI is bounded at
15 minutes. Build with CGO_ENABLED=0, -trimpath and stripped ldflags.
Version falls back to Go build metadata when unstamped.

- **`CGO_ENABLED=0` must not reach `make test`.** The race detector needs cgo on
  linux/amd64 but not darwin/arm64, so a file-level export passes locally and fails CI.

**The conformance suite** (`cmd/kit/conformance*_test.go`) builds the binary once and drives
it with the test suite’s race mode against throwaway projects, asserting on stdout, stderr, exit codes and the files left
behind. In-tree, because an external script is deleted without anything noticing. Four
properties are load-bearing. `TestEveryRegisteredCheckIsExercised` compares the cases
against `kit.Checks` both ways, so a check with no test fails the build. Every audit case
starts from the same clean project, which must report nothing. Every test is independent —
own `t.TempDir()`, no `os.Chdir` or `os.Setenv`, binary built once behind `sync.Once`.
**And the binary under test gets `gitEnv()`, not `os.Environ()`**: it used to inherit the
developer's `~/.gitconfig`, so three tests — including the one written to catch the
`release` deletion bug — passed only on a machine whose `init.defaultBranch` is `main`.

Coverage of the driver process is not coverage of the tested binary; no percentage is reported.

The module is private: installation requires GOPRIVATE and credentials. Preserve existing
comma-separated GOPRIVATE entries; exported environment values override `go env -w`.
Generated CI omits kit audit to avoid distributing those credentials; this repo audits
its own source. After merge, reinstall and publish a tag: `@latest` resolves to a release.

Release binds the complete gate and annotated tag to one clean published trunk commit.
Retry reuses the same tag; conflicts and unreadable state refuse. Verify the remote tag
object and exact-version installation. Tag-stale currently reads local tags; its remote
visibility limitation remains in BACKLOG.

`kit help compatibility` owns supported versions, stable v1 interfaces and coordinated
upgrades. [RELEASE.md](RELEASE.md) owns the post-merge publication procedure.
