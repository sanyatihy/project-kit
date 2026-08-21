---
id: 0050
title: A claim is one branch and the remote is the authority
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

The push that takes a feature is an atomic compare-and-swap and it is correct. Everything
that reads the result afterwards is not: five reproduced states end with two agents holding
one feature, a pushed commit destroyed, or a feature nobody can claim or release.

## Non-goals

- **Changing how a claim is won.** `--force-with-lease` plus `pushFlag` is right and six
  concurrent claimants still resolve to exactly one winner.
- **A lock, a lease or a heartbeat.** The branch stays the whole record.
- **Fetching inside `next`.** The board asks `ls-remote` already; this is about which
  answer it believes, not about asking more often.

## Design

**A claim is the branch named after the doc, and nothing else.** `Mine` asks whether HEAD
is *a* branch mentioning the id, so an agent that lost the race takes ownership by creating
an ordinary scratch branch. Reproduced: after losing, `git checkout -b notes-0001-alpha`
makes the board print `yours — this checkout claims it`, and `kit ship` then freezes another
agent's feature. `Mine` becomes `HeadBranch() == f.Branch()`. `Claims` keeps its wider match,
because a stray branch citing an id is still worth reporting as `split-claim`.

**The ref about to be deleted is the ref that is asked.** `Release` resolves the branch
locally first, so a stale local copy answers "carries no work" while the remote carries
commits. Reproduced: A claims, B pushes work to the same branch, A runs `kit release` and
the commit is unreachable — exit 0, no `--force`. `BranchBuildsOn` is asked of the remote
ref when one exists, and `release` refuses when the local and remote copies disagree.

**The claim is won atomically or not at all.** `claim` pushes, then runs `git branch`. A
name git accepts as a refspec and rejects as a branch argument leaves the claim on the
remote and reports failure, after which `claim` says it is taken and `release` says this
clone never created it. Create the local branch first, and roll the ref back if the local
step fails.

**A branch name comes from git, not from a string that looks like one.** `git branch
--format=%(refname:short)` emits `(no branch, rebasing 0002-beta)` and `(HEAD detached at
…)`, and disambiguates a branch shadowed by a tag as `heads/0002-beta`. Each is read as a
second claim: `split-claim` fires with an unfollowable fix, `ship` refuses during a rebase,
and `release` refuses for ever while the tag exists. Read refs with `for-each-ref
refs/heads/` and strip any configured remote prefix rather than only `origin/`.

Reproduced, all against the installed binary:

| state | what the kit did |
| --- | --- |
| loser creates `notes-0001-alpha` | board says `yours`; `ship` froze the winner's doc |
| stale local ref, remote has commits | `release` deleted them, exit 0 |
| doc named `-0007-*.md` | claim on the remote, exit 2, `release` refuses it |
| rebase in progress | cannot ship your own feature |
| remote named `upstream` | no trunk resolves; `stalled` can never fire |

## Tasks

- [ ] `Mine` is branch equality, and a scratch branch citing an id no longer confers ownership
- [ ] `release` asks the remote ref, and refuses when local and remote disagree
- [ ] `claim` cannot leave a ref on the remote it does not own locally
- [ ] Refs are read from `for-each-ref`, and any remote prefix is stripped, not just `origin/`
- [ ] A test drives each of the five states above

## Log
