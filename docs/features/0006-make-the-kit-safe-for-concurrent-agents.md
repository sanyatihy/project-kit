---
id: 0006
title: Make the kit safe for concurrent agents
status: Shipped
created: 2026-08-12
shipped: 2026-08-12
related: 0005-make-claims-and-blocking-mean-what-the-docs-say.md
---

## Problem

The rules were written for one operator shipping weekly. Two agents break three of them, and
two break silently.

- Both get the same id. `kit new` reads the next free id off disk, so two branches off one
  base both produce `0002`. Different filenames, so the merge is clean and `audit` reports
  agreement — and the id is the primary key, so one branch claims both features and one
  INBOX line blocks both.
- The claim is not atomic. Creating a branch and pushing it are two steps, and both agents
  can pass "is it taken?" before either is visible.
- `next` routes agents into each other. Standing in 0001's checkout it offered 0002 under
  "start here", which starts a second feature in the first one's branch.

Thresholds are an order of magnitude off in the same direction: `next` surfaces a block
instantly, `audit` waits a week, and an idle agent costs minutes.

## Non-goals

- Hash ids. They buy correctness without coordination, and the shared remote is already a
  point of coordination. The bill is paid daily in branch names and commit subjects.
- A database. Git's ref creation is already atomic, which is the only property missing.
- Worktree management. Git's job and a launcher's.
- Choosing how many agents run at once. A function of agent reliability and review
  bandwidth, both moving.

## Design

Uniqueness is auditable, so `duplicate-id` goes first. Everything else assumes an id names
one feature, and two agents offline can always collide. The residue is a file rename, one
`id:` line and a renamed branch.

The push is the claim. `kit claim <id>` pushes the branch and creating the ref decides, the
remote compare-and-swapping against the value it advertised. The ref status decides and not
the exit code: pushing a branch the remote already has at that commit prints "Everything
up-to-date" and exits 0, which would hand one feature to both agents. `--porcelain` marks
`*` created, `=` already there, `!` rejected, and only `*` is a win.

`next` reports the claim on HEAD as yours, and never headlines another feature in a checkout
already holding one.

Thresholds follow the cadence of what they watch rather than the operator's calendar.

## Tasks

- [x] Audit: `duplicate-id` — two feature docs sharing one `id:`
- [x] `kit claim <id>` — create and push the branch; a rejected push is a lost race
- [x] `kit next` reports the claim on the current HEAD as yours, not as taken
- [x] `kit next` will not headline a different feature in a checkout already claimed
- [x] `StalledDays` and `InboxStaleDays` retuned from weekly-human to agent cadence
- [x] Protocol: the worktree is the isolation, the push is the claim, and where the ID race
      can still bite

## Log

- 2026-08-12 — a rejected push does not mean somebody was first. It is false when both push
  the same commit, which is the normal case. Verified by racing four clones onto one ref 21
  times: exactly one `[new branch]` per round.
- 2026-08-12 — thresholds are 3 days stalled, 1 day inbox. Three because a claim taken on
  Friday and picked up on Monday is not a stall; one because an inbox line carries a date
  and no time.
- 2026-08-12 — `kit new` stays offline. A fetch there is advisory where `claim`'s push is
  atomic: it narrows the window, cannot close it, and leaves `duplicate-id` needed anyway.
- 2026-08-12 — `next` prints `kit claim <id>` where it printed `git worktree add -b`, which
  recommended the one route that skips deciding who gets the work.
