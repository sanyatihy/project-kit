# v1 compatibility and upgrades

## Supported use

Kit runs on macOS and Linux with git as its only runtime dependency. Windows is outside
the v1.0 support scope. Go and Python are the supported scaffolds.

The supported baseline is git 2.39.5, GNU make 3.81 for development gates, Go 1.26.4
for source builds and the Go scaffold, and Python 3.12.9 with uv 0.12.10 for the Python
scaffold. Older versions are not certified. Go projects record their toolchain in go.mod;
the linter is v2.12.2. Python selects the 3.12 interpreter line in .python-version.
Run make sync once and commit uv.lock before the first CI run. The lockfile records
resolved development dependencies; CI uses uv sync --locked and the pinned uv version.
The v1 candidate was exercised on macOS 15.7 and Debian 12, both arm64. Other architectures
need the same gates before claiming validation; CI supplies additional runner evidence.

Each agent uses its own checkout. Agents sharing a remote upgrade together. One binary
cannot enforce human review when agents can use the human's merge credentials.

## What stays compatible through v1.x

- Canonical command names, documented flags and exit meanings remain compatible.
  Exit 0 means success, 1 means a completed negative result, and 2 means refusal or an
  incomplete operation. For claim, 0 also includes safely resuming your recorded claim.
- Existing JSON fields retain their types and meanings. Additional fields may appear;
  consumers ignore unknown fields. Human-readable output is not a parsing interface.
- Audit identifiers remain stable. Additional checks and stronger validation of unsafe
  states may change whether a gate passes and are called out in release notes.
- Feature ids, frontmatter keys and parsed section headings remain supported. Keep the
  machine-readable keys, headings and statuses in English; body prose may use any language.
- Legacy Shipped/shipped: snapshots remain readable and frozen after integration.
  ship and reset remain aliases of submit and reopen with the current safety guards.
- init preserves application files, existing feature documents and customized stack files.
  Only the real Work protocol section is refreshed; house rules belong outside it.

Breaking removals or changed machine meanings require a new major release. Internal Go
packages, test implementation, branch display titles and prose wording are not public APIs.

## Upgrade from v0.7 or an untagged predecessor

1. Pause writers sharing the remote. Commit and push valuable work; record which checkout
   owns each in-flight claim. Keep those checkouts, including their git metadata.
2. Install the chosen published kit version on every machine and run kit version. Avoid
   mixing old writers with v1 claims: the old filename lock cannot coordinate the new
   immutable id lock. Merely renaming an old branch is not a migration.
3. Run kit init in each project. Review its diff and commit the protocol refresh through
   normal review. Existing Makefiles and CI are preserved; compare them with the current
   scaffold and apply relevant gate fixes explicitly. init is not a stack-file updater.
4. Run kit audit --strict online, fetching advertised branch objects when requested.
   Resolve duplicate ids and malformed plans. Never rewrite integrated feature history.
5. Resume existing legacy branches in their original checkouts. New claims use work/<id>
   and carry a unique empty claim commit. Local git refs retain acquisition evidence and
   confirmed ownership; keep them until release. Claim retries reconcile interruptions.

Changed operations from v0.7:

| Previous operation | Current operation |
| --- | --- |
| ship a completed feature | submit; status Submitted and submitted: date |
| reset review changes | reopen on the unmerged feature branch |
| reset an invalid/missing status | reopen --repair |
| revive a Dropped plan | draft a new feature for approval |
| release --force | release --discard; foreign claims and dirty files remain protected |
| drop an id with no local doc | drop --reservation for explicit unused-id cleanup |
| shipped-not-pushed / shipped-no-date / done-not-shipped | submitted-not-pushed / submitted-no-date / done-not-submitted |

An unknown push outcome preserves the attempt. Retry kit claim <id> from that clone
when the remote is reachable. A confirmed competing winner returns exit 1. Do not delete
refs by hand or decide ownership solely because two branches started at the same commit.
A worktree creation warning means the claim is already held: preserve any obstructing
files, then retry claim. Offline output labels cached observations; it cannot certify
current remote ownership, publication or integration.

## Release boundary

A Submitted feature is awaiting implementation review. After the human merges, the
release operator validates the merged trunk, publishes the intended version and checks
an installation of that exact version. Required remote checks and merge permissions are
release conditions. An unverified setting remains pending; local tests do not certify it.
