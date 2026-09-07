# v1 compatibility and upgrades

## Supported use

macOS and Linux; git is the only runtime dependency. Go and Python scaffolds are supported.
Windows is outside v1.0 support. Verified baseline:

- git 2.39.5; GNU make 3.81 for development gates.
- Go 1.26.4 for source builds and scaffolds; golangci-lint v2.12.2.
- Python 3.12.9 with uv 0.12.10. Run make sync and commit uv.lock before CI;
  CI uses uv sync --locked. .python-version selects the 3.12 line.
- Rehearsed on macOS 15.7 and Debian 12, arm64. Older tools and other architectures
  need validation; CI supplies additional runner evidence.

Agents use separate checkouts and upgrade together. Shared human credentials cannot
enforce human-only approval; reserve merge authority outside the agent.

## Stable through v1.x

- Canonical commands, flags and exit meanings: 0 success, 1 completed negative result,
  2 refusal/incomplete operation. Claim 0 includes recovery of recorded ownership.
- Existing JSON field types and meanings; additive fields are allowed. Ignore unknown
  fields. Human-readable wording is not a machine interface.
- Audit identifiers. New checks or stricter unsafe-state validation appear in release notes.
- Feature ids, frontmatter keys and parsed headings. Keep keys, headings and statuses
  English; body prose may use another language.
- Legacy Shipped/shipped: and ship/reset aliases, with current guards and frozen history.
- Init preserves application files, feature docs and custom stack files. Only the real
  Work protocol section refreshes; keep house rules outside it.

Breaking removals or changed machine meanings need a major release. Internal Go packages,
tests, prose and branch display titles are not public APIs.

## Upgrade from v0.7 or an untagged predecessor

1. Pause writers. Commit and push valuable work; record its owning checkouts. Preserve
   those checkouts and git metadata.
2. Install the chosen release everywhere and run kit version. Old filename claims and
   new work/<id> claims cannot safely coordinate mixed writers; renaming is not migration.
3. Run kit init per project and review the protocol diff normally. Compare preserved
   Makefiles and CI against current scaffolds; init does not update existing stack files.
4. Run kit audit --strict online; fetch missing advertised objects. Resolve duplicate ids
   and malformed plans without rewriting integrated history.
5. Resume legacy work on its existing branches. New claims use immutable ids and unique
   empty receipts. Preserve local acquisition/ownership refs until release.

Changed operations:

| Previous | Current |
| --- | --- |
| ship | submit; status Submitted and submitted: date |
| reset review changes | reopen on the unmerged feature branch |
| reset invalid/missing status | reopen --repair |
| revive Dropped | new feature for human approval |
| release --force | release --discard; ownership/dirty-work guards remain |
| drop id without a doc | drop --reservation for unused-id cleanup |
| shipped-not-pushed / shipped-no-date / done-not-shipped | submitted-not-pushed / submitted-no-date / done-not-submitted |

Unknown push outcome: retain the receipt and retry claim from that clone when online.
Confirmed competing winner: exit 1. Never infer ownership from matching starting commits
or delete refs by hand. Worktree creation warning: the claim is won; preserve obstructing
files and retry. Offline evidence cannot certify remote ownership or integration.

## Parked plans

Upgrade every writer before parking or activating plans. Parked is a scheduling status,
not an invalid Draft. Next adds a `parked` array with reasons and no build tasks; existing
JSON fields keep their meanings. Park and activate extend the lifecycle through reviewed
planning commits; other protocol rules remain in force.

The pre-0077 binary at b8bfe7e refuses a Parked claim with exit 1. Its board treats Parked
as unknown and may suggest repair; older repair commands can erase scheduling intent.
Never repair Parked with an old binary or use mixed writers. Current reopen refuses
Parked even with --repair. A stale clone must update trunk before claiming.

Parking checks visible claims when the command runs. The new HIGH `parked-claim`
finding flags a known claim on a parked plan. Recheck next and audit before
merging the planning PR: a later claim can race a review. Git permissions and human
review enforce merge authority; these commands cannot prevent raw git changes.

## Release boundary

Submitted means awaiting review. After human merge, the release operator validates trunk,
publishes the version and verifies its installation. Required checks and remote merge
permissions are release conditions; local fixtures do not certify live enforcement.
