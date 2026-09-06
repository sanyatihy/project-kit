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
- BLOCKING inbox lines match their id field; malformed lines conservatively match the
  whole line. FYI never blocks or ages. Concurrent questions append one complete line
  per write. Answer in the plan, remove its inbox line and commit both before resuming.
- VISION holds direction; TECH holds current facts; features hold plans; Log holds plan
  changes; PRs hold validation and handoff. Log may be empty; length is not a gate.
  Integrated feature history is frozen. The one-time prose compaction is recorded in 0073.

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
  remote delete leases the observed tip. Current or custom worktrees may need later cleanup.
- Reopen checks the published trunk before changing a submission; missing objects or
  unknown integration evidence refuse. Reopen, submit and drop reject foreign claims.

## Board, audit and git

- Next and audit read one repository, from any subdirectory. Every next state includes
  an instruction, shared by text and JSON. Missing `docs/features` exits 2.
- Claims and review states use advertised remote branch tips, not stale remote-tracking
  refs. Progress and last-moved dates come from that tip; unfetched objects give no count.
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
  `featureChecks` and `repoChecks` use the registry in audit.go. Help is generated from it.
  Every registered check must have one implementation and a conformance case.
- One defect produces one actionable finding. Severity reflects damage; age is advisory.
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
- Remote calls are bounded at 15 seconds. Set `exec.Cmd.WaitDelay` within that budget;
  killing git alone can leave a helper holding its output pipe. Inject dates into logic.

## Scaffolding and parsing

- Init exclusively creates missing files, preserving existing files and dangling symlinks.
  It owns the document layout; submit's TECH creation is a legacy fallback. Only the
  Work protocol section is refreshed; house rules and following sections survive.
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
- CLI parsing accepts flags after positional arguments. Requested help uses stdout;
  bad-flag usage uses stderr. Shipped references cite rule names, not list numbers.
- Help topics and protocol paths must resolve through embedded content or scaffolded files.
  Checklists are review aids, not executable task boxes. Tests protect reachability,
  parsing and refusals, not prose length or wording counts.

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
15 minutes. Build with CGO_ENABLED=0, -trimpath and stripped ldflags; do not export
CGO_ENABLED=0 into race tests. Version falls back to Go build metadata when unstamped.

Conformance builds the binary once and drives isolated temporary projects with controlled
gitEnv, not the user's environment. No shared cwd, globals or fixture ordering. Coverage
of the driver process is not coverage of the tested binary, so no percentage is reported.

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
