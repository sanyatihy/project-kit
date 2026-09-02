---
id: 0064
title: Every id the kit hands out can be accounted for
status: Shipped
created: 2026-08-22
shipped: 2026-08-23
needs:
---

## Problem

`kit new` reserves an id by pushing `refs/kit/ids/<id>` at the empty tree, so the
reservation records nothing — no title, no date, no author — while a doc becomes real to the
board only once it lands on the trunk. Planning legitimately cuts drafts between those two
moments, and four ids went that way before anything noticed: two in a round reversed and
pivoted into 0061 (`2c1f807`), two on a plan branch never merged (`33480e9`). Nothing
reported it, and `kit drop` cannot reach an id no doc holds.

## Non-goals

- Reusing an id a doc on the trunk ever held. A number two commits can cite means two
  things; a reservation whose doc never landed makes nothing ambiguous.
- Reporting a doc with no reservation. Every doc predating reservations would fire it.
- Recovering a cut draft. The reason given on release is the record.
- A new command. `kit drop` already means retire what will not be built.

## Design

A reservation says what it was for. `kit new` points the ref at a commit naming the title
rather than at the shared empty tree. The tree was chosen so a re-push answers "up to date"
instead of conflicting, and `pushFlag` already reads `!` for a ref holding something else,
so the compare-and-swap is untouched and only which flag a loss carries moves.

`kit drop <id>` releases a reservation no doc holds, returning the number to circulation.
It refuses while any branch carries a doc with that id, and while the reservation is younger
than the horizon — together those keep it off a draft still being written.

`audit` reports a reservation that no doc describes, at MEDIUM beside `orphan-claim`, which
is its mirror. Its fix is `kit drop <id>`, so the finding is one somebody can act on. It
waits out the `stalled` horizon, because a younger reservation is an ordinary draft in
flight.

Reading the record costs one fetch. `ls-remote` answers with SHAs alone, so `audit` fetches
`refs/kit/ids/*` into the same namespace locally — one round trip beside the one it already
makes, and every read after it is offline.

## Tasks

- [x] `kit new` points a reservation at a commit naming the title, and one id still reaches
      exactly one drafter
- [x] `kit drop <id> "<why>"` releases a reservation no doc holds, and refuses while a branch
      carries one or the horizon has not passed
- [x] `audit` reports a reservation past the horizon that no doc describes, naming `kit drop`

## Open questions

- The horizon, or position in the sequence? A reservation below the highest landed id has
  been skipped over, which needs neither a date nor the fetch — but two rounds drafted at
  once, the higher landing first, would report the lower while it is still being written.
  Leaning: the horizon, which `stalled` and `inbox-stale` already use.

## Log

- 2026-08-22 — renumbered from 0068, and 0063's doc from 0067, after the four unused
  reservations were released rather than left as gaps. `370f20d` cites the old numbers.
- 2026-08-23 — the horizon, not position in the sequence. A reservation carrying no date
  predates the record and so counts as past it.
- 2026-08-22 — `drop` releases the reservation where it recorded a retirement, and the fourth
  box went with the change: the four ids this was drafted for are already free.
