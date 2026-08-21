# Surface audit — 2026-08-21

A snapshot taken at `1475513`, to decide what the kit cuts before it onboards new projects.
It is a decision input, not a durable record: what survives becomes feature docs and
`BACKLOG.md` lines, and this file goes.

## Method

Four parallel readings — the command surface, the check registry, the onboarding path, the
prose — each required to drive a binary built from source rather than reason about it. The
check registry was additionally replayed against all 382 commits reachable from every ref.

Every defect in the next section was reproduced twice: once by the reading that found it,
once by hand against the installed binary.

## Verdict

**The kit is not too big. It is too quiet in four places, and its check vocabulary has
outgrown what it protects.**

The three commands expected to be ceremony — `block`, `drop`, `release` — each cost under
200 lines and each close a hole nothing else closes. Cutting all three saves 6.9% of the
non-test code and costs three guarantees. The expectation was wrong.

`audit` is 969 lines of Go and 1,449 lines of test — **15.7% of the non-test kit and 21.6%
of its tests** — for 34 checks. Across 382 commits, four codes have ever fired.

## Defects

Four states where the kit is wrong rather than verbose. Each blocks or corrupts the loop a
new project walks on day one.

### A claim taken from a worktree carrying work builds on that work

`kit next` in a worktree that has just shipped `0001` offers `0002`. Taking it exits 0:

```
$ kit claim 0002
  claimed  0002-beta-thing
  pushed   origin/0002-beta-thing — the push is the claim, and you won it
$ git log --oneline origin/main..0002-beta-thing
6deb089 0001: ship
956b992 0001: add it
```

`0002`'s pull request contains `0001`'s work. `ClaimFeature` already refuses this shape when
the doc sits on an unlanded plan branch, and the refusal names the reason — *"only its own
author could merge it"* — which is equally true here. The check is `trackedAt(trunk)`; what
is missing is the symmetric one on HEAD.

### A dropped stub refuses every ship in the project, for ever

`kit drop` on a feature whose plan was never written leaves three HIGH findings:

```
status: Dropped
docs/features/0005-stub-feature.md  (unfilled-template)
  0005 Stub feature: ## Problem was never written
  fix: write it, or set status: Draft until you can
audit --strict exit=1
```

Both offered fixes are refused elsewhere: writing a plan for a retired feature is work
nobody wants, and rule 21 forbids editing a terminal doc. Protocol rule 22 runs
`kit audit --strict` before every ship, and strict is project-wide, so one dropped stub
stops every agent. `unfilledTemplate` tests `s.tracked` and not `terminal(s.f.Status)`.

### Deleting the highest-numbered doc lets `kit new` reuse its id

Deleting a doc in the middle of the range is safe. Deleting the last one is not: `0004` was
removed and the next `kit new` wrote `0004` again, under a different title. Every branch and
commit citing `0004` now names a different feature. `NextID`'s own comment states this must
not happen; only `kit drop` prevents it, and nothing says so.

### `kit next` offers a claim `kit claim` refuses

With the plan round pushed and not yet merged — steps 9 and 10 of the onboarding the
protocol prescribes, so every new project stands here:

```
$ kit next                    → start here / 0001 / kit claim 0001        exit 0
$ kit claim 0001              → is not on main … Land the round first     exit 2
$ kit audit                   → 1 MEDIUM finding (unfilled-vision)
```

`claim` learned the rule and `next` did not. Milder than the three above — the refusal names
the fix — but it is the first thing a new operator meets, and 0049 set the standard that the
kit prints nothing untrue.

## The check registry: 34 → 21

11 HIGH, 18 MEDIUM, 5 LOW today. The cut applies the standard 0049 already set: **a check
that reports a state nobody may fix is deleted, not tuned.**

**Deleted outright — 6.** `unticked-shipped` and `shipped-early` report states `kit ship`
refuses to create, and both prescribe editing a Shipped doc. `needs-cycle` costs 58 lines
for a state reachable only by hand-writing two `needs:` fields at each other, never once
seen. `no-plan` prints the message and the fix string `kit next` already prints. `dead-link`
has fired zero times in 382 commits and its parser once panicked inside the command that
gates every ship. `fyi-stale` reports a note nobody is required to act on.

**Merged away — 7.** `no-tasks` into `unfilled-template`, `bad-status` into `no-status`,
`no-tech` into `tech-drift`, `no-inbox` and `no-vision` into `no-protocol`, `inbox-unknown`
into `inbox-unparsed`, `shipped-bad-date` into `shipped-no-date`. Each pair is one defect
with one fix, split across two vocabulary entries.

**Kept — 21**, of which nine are the core guarantees: `secret-in-doc`, `duplicate-id`,
`unfilled-template`, `stalled`, `shipped-not-pushed`, `unreadable-doc`, `id-mismatch`,
`inbox-stale`, `split-claim`.

The cut removes roughly 420 lines of implementation and conformance test, and takes HIGH
from 11 to 9 — two fewer ways for an unrelated doc to refuse every agent's ship.

Two findings that are not about size:

- **`split-claim` is MEDIUM.** Two agents believing they own one feature is the worst
  failure the kit has, and it does not fail `--strict`. `unticked-shipped` — a state `ship`
  cannot produce — is HIGH and does.
- **`orphan-claim` costs 143 lines and 5 tests for zero firings.** 0019's non-goals forbid
  cutting it. The number is recorded, not acted on.

## The prose: 505 lines that ship, 190 that earn it

**The protocol is 27 rules; 14 earn their place.** Ten are cuttable with no loss, and three
name constraints a refusal should carry. Three are worse than redundant:

- **Rule 10 teaches something the binary contradicts.** It says a branch only you can see is
  not a claim. `kit help` says local branches, remote branches and worktrees all count, and
  an unpushed local branch does halt the board.
- **Rule 22 names `make check`** in projects `kit init` gives no Makefile — which is the
  default, fastest onboarding path.
- **Rule 27 names `make release`**, which in a scaffolded Go project builds a binary rather
  than cutting a tag. `budget_test.go` records that the generated section cannot name the
  command, and the rule names it anyway.

**Rule 5's id clause fails silently**, which is the one that matters. A branch named
`plan/fold-in-0002-fix` makes the board report 0002 as taken and `kit audit` exit 0 saying
the docs and the repo agree. `OrphanClaims` anchors on `^(\d+)`, so a prefixed name cannot
be caught. The kit's own principle — prefer a refusal in the tool to a rule in prose — says
`kit new` should refuse to write a doc while HEAD cites a four-digit id that is not its own.

**`templates/go-checklist.md` is 193 lines, the largest text the binary ships**, larger than
the protocol every agent must obey. Nothing in the loop points at it; the work protocol
never names it. Eight of its opening lines exist to stop the host tool misreading its
checkboxes, and a conformance test pins that it must contain none. It is also single-language
in a kit whose `--stack` flag is built to take more.

## What `kit init` writes: 5 files, 269 lines

| file | lines | verdict |
| --- | --- | --- |
| `AGENTS.md` | 121 | keep, trim to ~63 with the protocol |
| `docs/features/TEMPLATE.md` | 69 | **delete — read by nothing** |
| `docs/VISION.md` | 46 | stop writing the stub |
| `docs/INBOX.md` | 21 | create it in `kit block`, as `ship` creates `TECH.md` |
| `docs/features/BACKLOG.md` | 12 | stop writing it |

`TEMPLATE.md` is dead and the source says so: `scaffold.go` records that *"the conventions
doc is read by nobody"*. Deleted from a project, `kit new` still writes a full doc and
`audit` never mentions it.

**Writing `docs/VISION.md` makes the audit worse.** No file is `no-vision` at LOW; the stub
`init` writes is `unfilled-vision` at MEDIUM. Scaffolding a file downgrades the project's
health report, and it is the only finding a virgin project has.

## Commands: keep all eleven

`init`, `new`, `claim`, `ship` and `next` complete the loop for two agents with no other
command invoked. The other six each cover a hole nothing else covers:

- **`drop`** reserves a retired id. Deleting the doc instead reuses the number.
- **`release`** is three refusals on the only destructive operation — carries work, not
  yours, cannot be read. Raw `git push --delete` has none of them and destroyed a live
  claim carrying a shipped feature.
- **`block`** is the only "wait for a human" state. It owns 75 lines; the 103-line read path
  belongs to `next`, `claim` and `audit` and survives it.
- **`audit`**, **`help`**, **`version`** — the first for `duplicate-id`, which git cannot
  surface because the filenames differ and the merge is clean.

**Cut `block --fyi`** — 35 lines across five files and one check, for a note nobody must
read.

Two flags are load-bearing beyond their documentation: `--offline` guards an **unbounded**
hang rather than the 15 seconds the help claims — measured past 55 seconds against a
black-holed remote, because `CommandContext` cannot reap the `git-remote-https` grandchild
holding the pipe.

## What this becomes

Three rounds, in this order. The first is not optional before onboarding.

1. **The loop is right or it refuses** — the four defects above. Fixes to shipped work, so
   no feature docs: a branch each, no four-digit id.
2. **The vocabulary is what it protects** — 34 checks to 21, and `split-claim` to HIGH.
3. **A new project gets the loop, not a library** — protocol to 14 rules, `go-checklist`
   deleted, `init` writing two files.

## Caveats

The 382-commit replay measures the trunk, which is by construction the clean corpus: checks
that fire mid-work — `unfilled-template`, `stalled`, `split-claim` — are undercounted, and
"four codes ever fired" understates them. It does not understate `dead-link` or
`needs-cycle`, whose states are equally absent from branches.

`--json` is judged on cost rather than observed use; how the agent harness consumes it was
not measured.
