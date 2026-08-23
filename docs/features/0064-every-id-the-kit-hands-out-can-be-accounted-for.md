---
id: 0064
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
two moments, and four ids went that way before anything noticed: two in a round that was
reversed and pivoted into 0061 (`2c1f807`), and two on a plan branch never merged whose
boxes 0061's task list carries almost verbatim (`33480e9`). Nothing reported it, and
`kit drop` cannot reach an id no doc holds.

## Non-goals

- **Reusing an id a doc on the trunk ever held.** 0058's rule stands where it was aimed: a
  number two commits can cite means two things. A reservation whose doc never landed makes
  nothing ambiguous, and that is the only case released here.
- **Reporting a doc with no reservation.** 0058 rejected that for a board where every doc
  predating reservations would fire it, and 0001 to 0061 here are exactly those.
- **Recovering a cut draft.** Its work either shipped elsewhere or was not wanted, so the
  reason given on release is the record and a dangling object is not one anybody reads.
- **A new command.** `kit drop` already means retire what will not be built; the case it
  cannot reach is the one to widen it to.

## Design

**A reservation says what it was for.** `kit new` points the ref at a commit naming the
title rather than at the shared empty tree. The tree was chosen so that a re-push answers
"up to date" instead of conflicting, and `pushFlag` already reads `!` for a ref holding
something else — so the compare-and-swap is untouched and only which flag a loss carries
moves.

**`kit drop <id>` releases a reservation no doc holds.** Deleting the ref returns the
number to circulation, which is the disposal path the kit lacks today: `drop --help` says
an unwanted doc cannot be disposed of by deleting a branch, and that only bites once the
doc reached the trunk. It refuses while any branch carries a doc with that id, and while
the reservation is younger than the horizon — together those are what keep it off a draft
still being written.

**`audit` reports a reservation that no doc describes.** MEDIUM, beside `orphan-claim`,
which is its mirror — a branch naming a doc nobody wrote. Its fix line is `kit drop <id>`,
so the finding is one somebody can act on. It waits out the `stalled` horizon, because a
reservation younger than that is an ordinary draft in flight, and a check that fires on
those is worse than no check.

**Reading the record costs one fetch.** `ls-remote` answers with SHAs alone, so `audit`
fetches `refs/kit/ids/*` into the same namespace locally — one round trip beside the one it
already makes, and every read after it is offline.

## Tasks

- [x] `kit new` points a reservation at a commit naming the title, and one id still reaches
      exactly one drafter
- [x] `kit drop <id> "<why>"` releases a reservation no doc holds, and refuses while a
      branch carries one or the horizon has not passed
- [ ] `audit` reports a reservation past the horizon that no doc describes, and names
      `kit drop` as the fix

## Open questions

- **The horizon, or position in the sequence?** A reservation below the highest id that has
  a landed doc has been skipped over, which needs neither a date nor the fetch — but two
  rounds drafted at once, the higher landing first, would report the lower while it is
  still being written. Leaning: the horizon, which is what `stalled` and `inbox-stale`
  already use.

## Log

- 2026-08-22 — renumbered from 0068, and 0063's doc from 0067, after the four unused
  reservations were released rather than left as gaps. `370f20d` and the merges above it
  cite the old numbers.
- 2026-08-22 — `drop` releases the reservation where it recorded a retirement, and the
  fourth box went with the change: the four ids this was drafted for are already free, so
  the design is what stops the next four.
