---
id: 0083
title: The board names the live claim a free feature would collide with
status: Submitted
created: 2026-09-10
submitted: 2026-09-10
needs:
---

## Problem

`needs:` orders work that cannot start until something merges. Nothing says anything about
work that can start and will collide. Two agents take two independent features, work in
separate worktrees, and the second pull request conflicts with the first or quietly undoes
it. An operator who ran nine features through the protocol arrived at the rule by hand —
features touching the same core files run one at a time, only disjoint features run in
parallel — and applied it while briefing each agent. That is a fact the repository already
holds, kept in a human's head and spent on every claim.

## Non-goals

- Refusing a claim. `needs-unstated` is LOW because a citation cannot prove a dependency,
  and a path named in prose cannot prove a collision either. Sequencing stays the
  operator's call.
- A `touches:` frontmatter key. `needs:` earned one because a dependency has no other
  evidence; a file has evidence in git from the moment somebody starts.
- An `audit` check. Audit reports where the docs and the repo disagree, and a collision
  that has not happened is not drift.
- Predicting a merge conflict. Two features meeting in one file may never conflict. The
  report is that they meet.

## Design

The two sides are not the same kind of fact, and the design turns on that.

A live claim's files are evidence. `git diff --name-only trunk...<branch>` is what that
agent has changed, and `BranchBuildsOn` already reads it for `stalled`. A free feature's
files are prose its author wrote before starting. So the report compares evidence against a
claim and is worded as one: "0083 names `internal/kit/next.go`, which 0079 has changed."

Paths come from backticked tokens in `## Design` and `## Tasks` that resolve to a file
present in the checkout. Existence is what makes prose usable here: `internal/kit/audit.go`
is a path, "the audit registry" is not, and a token naming no file is dropped rather than
guessed at. A doc that names none is silent — the same shape and the same ceiling as
`needs-unstated` reading Problem and Design for ids.

That ceiling is measurable, and it is the size of this feature. Over the 83 feature docs
here on 2026-09-10, 35 name at least one file that exists, and the median among those is
one path. So the report reaches under half the board and thinly, unless naming files
becomes a habit the kit asks for.

`next` reports it under the free feature it concerns, and `start <id>` on the exit-0 path,
where the claim is already won and this is what its holder will meet. Nothing refuses and
no exit code changes, so a wrong reading costs a line of output.

## Tasks

- [x] A feature's `## Design` and `## Tasks` yield the repo-relative paths they name in
      backticks, dropping any that names no file
- [x] A live claim's changed files come from its branch against the trunk
- [x] `kit next` names the overlapping claim and the files under each free feature
- [x] `kit start <id>` prints the same overlap on a won claim
- [x] `kit help next`, `kit help start` and `docs/TECH.md` carry it, and the feature
      template says a backticked path is read

## Open questions
- [x] 2026-09-10 · Should a free feature that names no backticked path be reported, so docs learn to name the files they touch? Measured here: 35 of 83 docs name at least one existing file in Design or Tasks, median 1, so silence means the warning reaches under half the board. Leaning: stay silent and accept the partial reach — a finding on the other 48 docs is exactly the noise a check must not make, and the habit is better asked for in the template than enforced.

## Log

- 2026-09-10 — boxes 1 and 2 land in one commit: both sides of the comparison are one
  file, and the second is ten lines over `BranchBuildsOn`'s existing diff.

- 2026-09-10 — settled: a free feature naming no path is not reported, so the design
  stands as written. The warning covers 35 of the 83 docs here and no more by choice;
  the habit of naming files is asked for in the feature template, not enforced by a
  finding on the other 48.
