---
id: 0066
title: A status the kit writes lands in a commit
status: Shipped
created: 2026-08-24
shipped: 2026-08-24
needs:
---

## Problem

`kit ship` and `kit drop` write `status:` into a doc and stop, so the working tree holds
the only copy of it. `audit` reads the tree, so it reports that the docs and the repo agree
while `HEAD` still says `Draft`. A shipped stamp was lost that way in an unattended chain:
one session shipped, a later one read the dirty doc as a stray edit and discarded it, and
`done-not-shipped` was what noticed six sessions on. Losing it takes only an agent with no
memory of the previous session, which is what a chain is.

## Non-goals

- Pushing the commit. `claim` pushes because a claim is a race and only the remote can
  settle it; a status is not contested, and the branch still goes for review the way rule 20
  says.
- Committing anything else. `ship` already refuses a dirty tree, so what it writes
  afterwards is the whole diff — see below.
- Repairing a committed status that is wrong. That is a status the kit never wrote, and
  0067 gives it a command.

## Design

The two commands that write a status commit it. Prevention over detection, and the
refusal above `ship`'s write is what makes it safe: the tree is clean before the stamp, so
the commit is exactly `status:` and `shipped:`, plus the as-built doc in the one case where
shipping created it. `drop` writes a status and a `## Log` line and is the same defect, so
it commits too. Message form is the one this repo's own history already uses: `Ship 0065`.

A commit that cannot be made is reported, not swallowed. No identity configured, a
rebase in progress, a detached HEAD: the stamp is already on disk, so the command says what
is uncommitted and exits 2 rather than claiming a freeze that only this checkout can see.

`status-not-committed` reports a terminal status absent from `HEAD`, at HIGH. It reads
the doc at `HEAD` and compares the status the kit acts on, which is the first check to look
past the working tree for anything but a claim branch. HIGH for `shipped-not-pushed`'s
reason: the record says a feature was never built, and the fix — commit it — is not
something a reader would guess from silence. It covers the docs an older binary shipped, a
stamp dropped in a rebase, and the write above whose commit failed. A doc absent from `HEAD`
counts as absent: `ship` cannot reach one, so a terminal status on an uncommitted file is
the same loss.

## Tasks

- [x] `kit ship` commits the stamp it writes, names the commit in its output, and reports
      rather than swallows a commit it could not make
- [x] `kit drop` commits the status and log line it writes, the same way
- [x] `audit` reports a doc whose terminal status is not the one in `HEAD`, with a
      conformance case

## Log
