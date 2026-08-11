# Features

One file per feature. This directory is the board — the files, plus each one's
`status:`, are what is in flight. Start a feature by copying [TEMPLATE.md](TEMPLATE.md).

Run `kit next` to be told which feature and task to pick up.

## Conventions

- **`Draft → Accepted → In progress → Shipped`** (exits: `Dropped` / `Superseded`).
  `status:` in the frontmatter is canonical.
- **List order is the order of work.** Do the topmost unticked `## Tasks` box. If two
  boxes can genuinely happen in either order, it does not matter which is first.
- **A branch named after the doc is the claim on it** (`0009-ton-payment-rail`).
  One agent per feature; that keeps two agents out of the same code.
- **One source of truth per fact.** While in flight the feature doc owns the spec.
  On ship, durable facts move to the as-built doc and this file freezes as history.
- **`## Log` holds dated conclusions**, not debate. The argument lives in the PR.
- **IDs are stable** (`0007`). Cite them in commits and branch names; files never move.
- **`check:` is optional.** Set it only when the feature is a bet worth revisiting —
  a pricing change, a growth experiment. Most features do not need one.
