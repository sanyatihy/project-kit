---
id: 0050
title: Only the agent that won a feature can act on it
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

The push that takes a feature is an atomic compare-and-swap and it holds: six concurrent
claimants still resolve to exactly one winner. What the kit does with the result does not.
An agent that lost the race can put itself back in the winner's seat with an ordinary
branch name, and the board will tell it the feature is its own.

## Non-goals

- **Changing how a claim is won.** `--force-with-lease` plus `pushFlag` is right.
- **A lock, a lease or a heartbeat.** The branch stays the whole record.
- **Reporting stray branches less.** A branch citing an id is still worth `split-claim`.

## Design

**Ownership is branch equality, not a name that mentions the id.** `Mine` asks whether HEAD
is *a* branch mentioning the feature's id. Reproduced: after losing the race,
`git checkout -b notes-0001-alpha` makes the board print `yours — this checkout claims it`,
and `kit ship` then freezes the winner's doc. `review-0001`, `wip-0001-x` and `fix-0001` all
qualify, so this needs no malice. `Mine` becomes `HeadBranch() == f.Branch()`.

**The claim is won atomically or not at all.** `claim` pushes, then runs `git branch`. A
name git accepts as a refspec and rejects as a branch argument leaves the claim on the
remote and reports failure — after which `claim` says the feature is taken and `release`
says this clone never created it, so only `--force` escapes. Create the local branch first,
and roll the ref back if the local step fails.

**A branch name comes from git's ref list, not from a string that looks like one.**
`git branch --format=%(refname:short)` emits `(no branch, rebasing 0002-beta)` and
`(HEAD detached at …)`, and disambiguates a branch shadowed by a tag as `heads/0002-beta`.
Each is read as a second claim: `split-claim` fires with an unfollowable fix, you cannot
ship your own feature during a rebase, and `release` refuses for ever while the tag exists.
Read `for-each-ref refs/heads/`.

Reproduced against the installed binary:

| state | what the kit did |
| --- | --- |
| loser creates `notes-0001-alpha` | board says `yours`; `ship` froze the winner's doc |
| doc named `-0007-*.md` | claim landed on the remote, exit 2, `release` refuses it |
| rebase in progress | cannot ship your own feature |
| tag named like a claim | `release` refuses for ever |

## Tasks

- [x] `Mine` is branch equality, and a scratch branch citing an id confers nothing
- [x] `claim` cannot leave a ref on the remote it does not own locally
- [ ] Refs are read from `for-each-ref`, so git's pseudo-branch names are never claims
- [ ] A test drives each of the four states above

## Log
