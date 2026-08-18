---
id: 0039
title: Cut what a fleet board read costs
status: Draft
created: 2026-08-18
shipped:
check:
checked:
review:
needs:
related: docs/features/0035-stop-re-asking-git-the-same-questions.md
---

## Problem

The operator's normal call is one board read over every repo, and it is serial: `kit audit .`
takes 3.0s, `kit audit . .` takes 5.8s, and ten projects would take half a minute. Almost all
of it is waiting — the same audit `--offline` costs 0.09s, and one `git ls-remote` to GitHub
over SSH is 2.3s of the 3.0s. Inside a single project the online path also repeats an answer
it already has: 60 git processes for 38 features, 25 of them `branch --remotes --contains` for
three distinct revisions. That is the regression `make gitcount` exists to catch, and it is
invisible to it, because the harness measures `next --offline` where none of this runs.

## Non-goals

- **Not a change to what any check decides.** Memoising and parallelising must leave the
  findings byte-identical; the one check whose *answer* is wrong on an edited shipped doc is
  a separate feature, and rewriting `OnlyHere`'s test belongs there, not here.
- **No cache that outlives the process.** A remote branch list held between runs would make
  the board report a claim that was released seconds ago; the kit's rule is that claims are
  read from the remote, not from a cache.
- **Not a fix for leftover branches on the remote** beyond reporting them. Deleting a
  branch is the forge's job or the operator's, and `audit` never writes.

## Design

**Projects are read concurrently, capped.** They share nothing — each is its own `Repo`, and
the fleet board already merges independent results — so the whole serial cost is avoidable.
The cap follows 0034's precedent rather than inventing a second parallelism idiom. Output
order stays the order the paths were given: a board whose lines move between runs is unusable
for comparing two runs, and 0026 fixed reporting to be positional.

**Within a project, the network call starts before the local walk.** Parse, `show-ref` and the
per-feature reads do not need the remote's answer until the checks run, so the round trip
overlaps the work instead of preceding it. Worth about 0.8s per project on this repo and
nothing on a project with no remote.

**`branch --remotes --contains` is memoised by revision.** Many docs share one commit — a trim
or a compaction round writes thirty in one — so the count collapses from 25 to 3 here. This is
the same rule 0035 applied to per-repo answers, extended to per-revision ones: the same
question, asked once.

**`gitcount` measures the online path too.** It stays offline by default and grows a second
line for the online fan-out against a local remote, so the network is one cheap process and
what is measured is still the per-feature growth. A guard that cannot see the path where the
duplication lives is the reason this shipped twice.

**Leftover claim branches are reported at LOW, not deleted.** Twelve merged claim branches sit
on this remote; each costs a `git show` on every `next`, and each one suppresses
`shipped-not-pushed`'s first negative, so the HIGH check is off for the twelve most recent
features. LOW because nothing here is untrue and the repo is healthy — the finding is that a
check has been quietly disarmed, which the operator can fix in one command or by turning on
delete-on-merge.

## Tasks
- [ ] Projects audited and surveyed concurrently under 0034's cap, output in path order,
      with a test that a failing path still reports in place and still exits 2
- [ ] The remote branch list requested before the local walk, and `--offline` unchanged
- [ ] `branch --remotes --contains` memoised by revision, with the findings pinned identical
      on a fixture that exercises the online path
- [ ] `gitcount` reports the online fan-out against a local remote, and the numbers land in
      the `## Log`
- [ ] `merged-claim` at LOW: a Shipped feature whose claim branch the remote still has, with
      the fix line naming the delete and the forge setting

## Open questions
- Does `merged-claim` earn its place at all? It fires twelve times on this repo today, and
  "a check that fires on healthy projects is worse than no check" is a constraint in the
  vision. Leaning: yes at LOW, because what it reports is a disarmed HIGH check rather than
  untidiness — but dropping the box and letting delete-on-merge handle it is a real answer.

## Log
- 2026-08-18 — measured before drafting: 3.0s one project, 5.8s two, 2.3s of each in
  `ls-remote`, 60 git processes for 38 features with 25 of them three distinct questions.
