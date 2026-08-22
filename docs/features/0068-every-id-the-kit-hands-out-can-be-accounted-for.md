---
id: 0068
title: Every id the kit hands out can be accounted for
status: Draft        # Draft | Shipped | Dropped — kit ship and kit drop write it
created: 2026-08-22
shipped:             # written by kit ship, never by hand
needs:               # feature ids that must ship first, e.g. 0002, 0003. Empty = can start now
---

## Problem

`kit new` reserves an id by pushing `refs/kit/ids/<id>` at the empty tree, so the
reservation records nothing — no title, no date, no author — while a doc becomes real to
the board only once it lands on the trunk. Planning legitimately cuts drafts between those
two moments, and four ids have already gone that way: 0065 and 0066 in a round that was
reversed and pivoted into 0061, 0063 and 0064 on a plan branch never merged, whose boxes
0061's task list now carries almost verbatim. Nothing reports any of it, and `kit drop`
cannot reach an id no doc holds.

## Non-goals

- **Reusing a burned id.** 0058 decided a reservation outlives its doc, and reuse is the
  collision the reservation exists to remove; an explained hole costs a reader one line.
- **Reporting a doc with no reservation.** 0058 rejected that for a board where every doc
  predating reservations would fire it, and 0001 to 0061 here are exactly those.
- **Recovering the two lost drafts.** Their work shipped inside 0061, so the retirement
  reason is the record, and a dangling object is not one anybody should read.
- **A new command.** `kit drop` already means retire what will not be built; the case it
  cannot reach is the one to widen it to.

## Design

**A reservation says what it was for.** `kit new` points the ref at a commit naming the
title rather than at the shared empty tree. The tree was chosen so that a re-push answers
"up to date" instead of conflicting, and `pushFlag` already reads `!` for a ref holding
something else — so the compare-and-swap is untouched and only which flag a loss carries
moves.

**`kit drop <id>` works when no doc holds the id.** It rewrites the reservation into a
retirement record carrying the reason, which is the disposal path the kit lacks today.
`drop --help` already says an unwanted doc cannot be disposed of by deleting a branch;
that warning only bites once the doc reached the trunk. The id stays out of circulation.

**`audit` reports a reservation that no trunk doc describes and no retirement explains.**
MEDIUM, beside `orphan-claim`, which is its mirror — a branch naming a doc nobody wrote. It
waits out the `stalled` horizon, because a reservation younger than that is an ordinary
draft in flight, and a check that fires on those is worse than no check.

**Reading the record costs one fetch.** `ls-remote` answers with SHAs alone, so `audit`
fetches `refs/kit/ids/*` into the same namespace locally — one round trip beside the one it
already makes, and every read after it is offline.

## Tasks

- [ ] `kit new` points a reservation at a commit naming the title, and one id still reaches
      exactly one drafter
- [ ] `kit drop <id> "<why>"` retires a reservation no doc holds, and says the id stays out
      of circulation
- [ ] `audit` reports a reservation past the stalled horizon that no trunk doc describes and
      no retirement explains
- [ ] 0063 to 0066 are retired with their reasons, and `plan/what-a-project-still-needs` is
      deleted

## Open questions

- **The horizon, or position in the sequence?** A reservation below the highest id that has
  a landed doc has been skipped over, which needs neither a date nor the fetch — but two
  rounds drafted at once, the higher landing first, would report the lower while it is
  still being written. Leaning: the horizon, which is what `stalled` and `inbox-stale`
  already use.

## Log
