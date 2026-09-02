---
id: 0046
title: Replace the protocol's hand-edits with commands
status: Shipped
created: 2026-08-20
shipped: 2026-08-20
needs: 0045
related: 0018-give-the-human-an-accept-command.md, 0044-route-each-fact-to-one-home.md
---

## Problem

The kit owns the writes that finish a feature and none of the writes that end one any other
way. Rule 8 answers stopping mid-feature with `git push origin --delete`, which discards
reviewed commits when the reason to stop was that the work should wait. `unchecked` fires at
HIGH asking for two writes into a `Shipped` doc, which rule 11 forbids. `ship` accepts a
worktree of scratch files without a word. All three hit in one feature built under the
protocol elsewhere.

## Non-goals

- Deciding what a frozen doc is. `kit check` needs no answer: no agent hand-edits the doc,
  so rule 11 needs no exception in prose.
- Setting `check:` after a doc ships. A doc that never named a date never claimed to be a
  bet, and `ship` cannot prompt for it in a non-interactive session.
- A `merged-claim` finding. `AwaitingReview` already skips a branch the trunk contains.
- A field for "paused". The branch and its pull request say it, and a second copy could only
  disagree.

## Design

`kit release <id>` deletes the claim on both sides and refuses while the branch carries work.
The refusal is the feature: rule 8's recipe is right for abandoning a claim and destroys
reviewed commits when the reason to stop was that the work should wait. `BranchBuildsOn`
already separates the two — a branch carrying only the feature doc is a proposal, one
carrying anything else is work. `--force` is the way past it, because an agent that cannot
release through the kit reaches for `git` and the kit stops being the record.

Pausing is rule 13's handoff without the status change: push, open the pull request
unfinished, and say why there. Neither path writes `## Log`, since an entry naming no change
to the plan is not an entry.

`kit check <id> "<outcome>"` stamps `checked:` and appends one `## Log` line, the way `ship`
stamps `shipped:`. That is the only write into a frozen doc the protocol asks for, so owning
it is what lets rule 11 stay absolute.

`kit ship` refuses a dirty worktree before it writes anything. The last box is ticked in the
commit that implements it, so a clean tree is the expected state at ship.

The protocol does not grow. Rule 8 loses the recipe it no longer needs, which pays for two
lines it never had: `kit next --json` is the machine-readable board, and rule 9 covers the
human who is present rather than filing a question at somebody sitting there.

## Tasks

- [x] `kit release <id>` deletes the claim branch on both sides, refuses while
      `BranchBuildsOn` says it carries work, and takes `--force`; rule 8 splits into it and
      the pause path
- [x] `kit check <id> "<outcome>"` stamps `checked:` and appends one `## Log` line on a
      `Shipped` doc; `unchecked`'s fix line names the command
- [x] `kit ship` refuses a worktree with uncommitted or untracked files, naming them
- [x] The protocol names `kit next --json` as the machine-readable board, and rule 9
      distinguishes the human in the session from the absent one
- [x] README carries `release` and `check`

## Log

- 2026-08-20 — pausing says why in the pull request rather than in `## Log`. A pause that
  moved no box changed no plan, and rule 7 already owns the case where a box does move.
- 2026-08-20 — the last box drops `kit help`, keeping README. `TestEveryCommandHasHelp` and
  `TestOverviewListsEveryCommand` make help text part of adding a command, so it lands with
  the command rather than after it.
