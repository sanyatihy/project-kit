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
related: docs/features/0028-see-what-is-waiting-for-review.md, docs/features/0035-stop-re-asking-git-the-same-questions.md
---

## Problem

The operator's normal call is one board read over every repo, and it is serial: `kit audit .`
takes 3.0s, `kit audit . .` takes 5.8s, and ten projects would take half a minute. Almost all
of it is waiting — the same audit `--offline` costs 0.09s, and one `git ls-remote` to GitHub
over SSH is 2.3s of the 3.0s. The rest is fan-out the kit already has answers for: 60 git
processes for 38 features, 25 of them `branch --remotes --contains` for three distinct
revisions. That is the regression `make gitcount` exists to catch, and it is invisible to it,
because the harness measures `next --offline` where none of this runs.

The same code is also wrong. `OnlyHere` proves the work reached the remote by asking whether
the commit that last wrote the doc is on a remote ref — a proxy that broke the day editing a
shipped doc became routine, which 0038 made it by having `kit ship` ask for compaction. One
blank line appended to a doc merged weeks ago produces `shipped-not-pushed` at HIGH, failing
`--strict`, with a fix line that recreates a stale claim branch the board then reads as a live
claim on a Shipped feature.

## Non-goals

- **No cache that outlives the process.** A remote branch list held between runs would report
  a claim that was released seconds ago; claims are read from the remote, never from a cache.
- **Not leftover-branch cleanup.** Twelve merged claim branches sit on this remote; the answer
  is delete-on-merge on the forge, not a kit check that fires on a healthy repo. Nothing here
  deletes a branch either — `audit` does not write.

## Design

**Projects are read concurrently, capped.** They share nothing — each is its own `Repo`, and
the fleet board already merges independent results — so the whole serial cost is avoidable.
The cap follows 0034's precedent rather than inventing a second parallelism idiom. Output
order stays the order the paths were given: 0026 made reporting positional, and a board whose
lines move between runs cannot be compared with the last one.

**Within a project, the network call starts before the local walk.** Parse, `show-ref` and the
per-feature reads do not need the remote's answer until the checks run, so the round trip
overlaps the work instead of preceding it. Worth about 0.8s per project here, nothing where
there is no remote.

**`shipped-not-pushed` asks the trunk's copy of the doc what its status is.** The handoff
happened if and only if a copy saying `Shipped` reached the trunk on the remote, and that is
what to test — not the age of a commit, which any later edit resets. Presence of the doc is
not the test and cannot be: a doc reaches the trunk when it is *drafted*, so 0039's own copy
on `origin/main` says `Draft` while this is being written, and a presence test would silence
the check for every feature. The branch-list negative stays as it is, because a Shipped doc on
a pushed branch is the awaiting-review state and must not fire.

**That replaces the per-feature fan-out with a batched read.** `ls-tree` on the trunk plus one
`cat-file --batch` reads every doc in two processes — 20ms for 40 docs here — where the
current pair of questions costs one `log -1` per feature and one `branch --contains` per
revision, 63 processes for the same answer. The memoisation 0035 applied to per-repo answers
is not needed if the question is asked once for all of them.

**`gitcount` measures the online path too.** It stays offline by default and grows a second
line for the online fan-out against a local remote, so the network is one cheap process and
what is measured is still per-feature growth. A guard that cannot see the path where the
duplication lives is why this shipped twice.

## Tasks
- [x] Projects audited and surveyed concurrently under 0034's cap, output in path order, with
      a test that a failing path still reports in place and still exits 2
- [x] The remote branch list requested before the local walk, `--offline` unchanged
- [x] `OnlyHere` reads the trunk's copy of the doc, batched for all features in one pass:
      conformance that an edit to a merged doc is silent, that a Shipped doc never pushed
      still fires, and that awaiting-review and `--offline` stay silent
- [x] `gitcount` reports the online fan-out against a local remote, with the before and after
      numbers in the `## Log`

## Log
- 2026-08-18 — measured before drafting: 3.0s one project, 5.8s two, 2.3s of each in
  `ls-remote`, 60 git processes for 38 features with 25 of them three distinct questions.
- 2026-08-18 — `merged-claim` dropped rather than shipped at LOW: it would fire twelve times
  on a healthy repo, and delete-on-merge stops the branches at the source.
- 2026-08-18 — the `shipped-not-pushed` fix folded in rather than drafted beside this: it is
  the same fan-out, and the batched read is what makes the correct test cheap enough to be the
  obvious one.
- 2026-08-18 — 0034 is not the precedent the design claimed: its cap is `go test -parallel` in
  the Makefile, a CPU-bound resource. The cap here is 8 and bounds connections to one forge.
- 2026-08-18 — asking the remote first bought 0.16s, not the 0.8s predicted: `OnlyHere` called
  it per feature, so the local fan-out sat behind the round trip rather than beside it. The
  batched read is what freed the rest.
- 2026-08-18 — an answered-but-empty branch list is evidence, not ignorance: a remote with no
  trunk at all carries nothing, which is the case the check exists for. Missing that turned it
  silent on its own fixture.
- 2026-08-18 — 62 git processes to 12 on this repo, and 67 to 10 in the harness where the
  offline count is 28 either way. Four projects 11.2s to 2.5s; one project 2.77s to 2.48s,
  against a floor of 2.3s that is `ls-remote` itself.
