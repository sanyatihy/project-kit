---
id: 0063
title: A claim's worktree lives inside the project
status: Shipped
created: 2026-08-22
shipped: 2026-08-23
needs:               # feature ids that must ship first, e.g. 0002, 0003. Empty = can start now
---

## Problem

`kit claim` prints `git worktree add ../<repo>-<id>`, a path relative to the current
directory, so it is right only from the trunk — run it from a worktree and it names a
sibling of that worktree. The same line puts every live claim in the project's parent,
where a directory per claim is indistinguishable from a repository, and the operator's
trees hold one repository per child.

## Non-goals

- **Creating the worktree.** `claim` prints the command, because the push is the claim and
  the worktree is only where you stand; an agent claiming from a clone dedicated to that
  feature already has its checkout, and `ClaimFeature` permits that.
- **A line in the committed `.gitignore`.** Where an operator keeps worktrees is a fact
  about a machine, and a tracked file carries one operator's layout to every clone.
- **A `worktrees-not-ignored` check.** `audit` reports what it cannot prevent, and once
  `claim` writes the exclude there is nothing left to drift.
- **Making the location configurable.** The kit has no config file and keeps no local
  state, so the option costs the mechanism before it costs the two code paths.

## Design

**The worktree goes inside the project, at `.worktrees/<branch>`.** It beats a sibling
directory, which cannot be committed by accident and which no recursive search can reach.
It costs exactly that: `grep -r` from the trunk sees a second copy of the source, where
ripgrep and the Go toolchain both skip the directory on the leading dot alone. What buys
the cost back is the parent staying a list of repositories, and each live claim appearing
in the editor as its own repository with its own branch and diff — the board, in the
window the work is already in. The directory is named for the branch so the listing reads
as one.

**The printed path is absolute, anchored at the main worktree.** `git worktree list
--porcelain` names it in its first record, which `worktreeFor` in `release.go` already
parses. This is the half that is a bug today, and it is worth fixing whatever the layout.

**The exclude goes in `.git/info/exclude`.** It is per-clone and untracked, which is what
the fact is; git honours it for `add`, and ripgrep honours it for search. `git clone` does
not carry it, so the write belongs in `claim`, which runs in every clone, and not in
`init`, which runs in one.

**`claim` writes the exclude and prints the worktree command.** Declining a worktree is
legitimate and declining the exclude is not, so the one that must be true is the one the
kit does rather than asks for. It writes only when `git check-ignore` reports the path
uncovered, and names the file it wrote in its output.

## Tasks

- [x] `kit claim` prints an absolute `<main worktree>/.worktrees/<branch>`, correct from
      any checkout
- [x] `kit claim` writes `.worktrees/` to `.git/info/exclude` when `check-ignore` reports
      it uncovered, and says which file it wrote
- [x] `kit help claim` and `docs/TECH.md` name where a worktree goes, and why the exclude
      is written once per clone

## Log

- 2026-08-22 — renumbered from 0067, into the id freed by releasing the unused
  reservations. `9db63b6` and the merge above it cite the old number.
