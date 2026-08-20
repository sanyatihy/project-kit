---
id: 0046
title: Replace the protocol's hand-edits with commands
status: Draft
created: 2026-08-20
shipped:
check:
checked:
review:
needs: 0045
related: 0018-give-the-human-an-accept-command.md, 0044-route-each-fact-to-one-home.md
---

## Problem
The kit owns the writes that finish a feature — `ship` sets `status:` and the date — and
none of the writes that end one any other way. Rule 8 answers stopping mid-feature with
`git push origin --delete`, which discards reviewed commits when the reason to stop was
that the work should wait. `unchecked` fires at HIGH asking for two writes into a `Shipped`
doc, which rule 11 forbids. `ship` accepts a worktree of scratch files without a word — all
three hit in one feature built under the protocol elsewhere.

## Non-goals
- **Deciding what a frozen doc is.** Rule 11 against the checks that contradict it is one
  question, and it is in BACKLOG. `kit check` needs no answer to it: no agent hand-edits the
  doc, so rule 11 needs no exception in prose.
- **Setting `check:` after a doc ships.** A doc that never named a date never claimed to be
  a bet, and a revisit somebody wants anyway is a BACKLOG line. Nor does `ship` prompt for
  the field: a non-interactive session has nobody to answer.
- **A `merged-claim` finding.** `AwaitingReview` already skips a branch the trunk contains
  and compares against a trunk ref that prefers the local one — a freshness bug, in BACKLOG.
- **A field for "paused".** The branch and its pull request say it, and a second copy in the
  doc could only disagree — 0032's reason for there being no in-progress status. A paused
  branch reads as a claim and then as `stalled`, which is what `stalled` is for.

## Design
**`kit release <id>` deletes the claim on both sides, and refuses while the branch carries
work.** The refusal is the feature: rule 8's recipe is right for abandoning a claim and
destroys reviewed commits when the reason to stop was that the work should wait.
`BranchBuildsOn` already separates the two — a branch carrying only the feature doc is a
proposal, one carrying anything else is work — so the test is one the board already trusts.
`--force` is the way past it, because an agent that cannot release through the kit reaches
for `git` and the kit stops being the record.

**Pausing is rule 13's handoff without the status change**: push, open the pull request
unfinished, and say why there. Neither path writes `## Log` — under the routing rule an
entry naming no change to the plan is not an entry, and a plan that did change has its
command already: rule 7 for a box, `kit drop` for the whole feature.

**`kit check <id> "<outcome>"` stamps `checked:` and appends one `## Log` line**, the way
`ship` stamps `shipped:`. That is the only write into a frozen doc the protocol asks for, so
owning it is what lets rule 11 stay absolute.

**`kit ship` refuses a dirty worktree before it writes anything.** The last box is ticked in
the commit that implements it, so a clean tree is the expected state at ship.
`git status --porcelain` is the test and it already excludes ignored files.

**The protocol does not grow.** Rule 8 loses the recipe it no longer needs, which pays for
the two lines it never had: `kit next --json` is the machine-readable board, and rule 9
covers the human who is present rather than filing a question at somebody sitting there.
`protocolBudget` holds at 123 or falls — 0045 is renegotiating that ceiling now, which is
what `needs:` records.

## Tasks
- [x] `kit release <id>` deletes the claim branch locally and on the remote, refuses while
      `BranchBuildsOn` says it carries work, and takes `--force`; rule 8 splits into it and
      the pause path, and stops asking for a `## Log` line
- [x] `kit check <id> "<outcome>"` stamps `checked:` and appends one `## Log` line on a
      `Shipped` doc; `unchecked`'s fix line names the command instead of the two edits
- [ ] `kit ship` refuses a worktree with uncommitted or untracked files, naming them, before
      it writes the status
- [ ] The protocol names `kit next --json` as the machine-readable board, and rule 9
      distinguishes the human in the session from the absent one; `protocolBudget` holds
- [ ] README carries `release` and `check`

## Log
- 2026-08-20 — pausing says why in the pull request rather than in `## Log`: a pause that
  moved no box changed no plan, so the routing rule sends it to the pull request, and rule
  7 already owns the case where a box does move.
- 2026-08-20 — the last box drops `kit help`, keeping README: `TestEveryCommandHasHelp` and
  `TestOverviewListsEveryCommand` make help text part of adding a command, so it lands with
  the command rather than after it.
