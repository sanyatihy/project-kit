---
id: 0023
title: Stop the commands doing quietly what the board forbids
status: Draft
created: 2026-08-16
shipped:
check:
checked:
needs:
related: 0021-let-agents-work-from-drafts-and-make-accepting-a-signal.md
---

## Problem
`next` and the commands disagree about what may be worked, and the commands win.
Measured on a fresh project with `0002` declaring `needs: 0001`:

```
$ kit next          0002 blocked — waiting on: needs 0001 to ship first
$ kit claim 0002    claimed                                    (exit 0)
$ kit ship  0002    shipped  (was Accepted)                    (exit 0)
$ kit audit         MEDIUM  shipped-early
```

`Survey` consults `UnmetNeeds`; `ClaimFeature` and `Ship` never do. So an agent that
claims by ID without reading the board builds on a foundation that is not there, and
`ship` mints a `shipped-early` finding on a doc the protocol then forbids anyone to
edit. TECH.md's own rule is *prevent before detecting* — `ship` exists because
hand-stamping frontmatter produced findings, and it is now producing one itself.

This got worse when 0021 landed. `status:` no longer gates anything, so `needs:` is
the only ordering constraint left, and it is unenforced at the two moments that
matter.

`ship` has a second, worse version of it. From a checkout whose own board says the
feature is **taken**, with two agents and a real remote:

```
$ kit next          taken   0001  — 0001-contested
$ kit ship 0001     shipped  (was Accepted)                    (exit 0)
```

One agent froze another's feature mid-flight. `status: Shipped` is history the
protocol forbids anyone to edit, so the agent actually holding the branch can no
longer touch its own doc, and `audit` reports nothing at all.

`kit drop` has two silent failures of its own. Dropping a feature others declare in
`needs:` strands them without a word — `audit` reports `needs-unknown` afterwards, and
`next` goes on saying *"waiting on: needs 0001 to ship first"* about a feature that is
Dropped and can never ship. And on a doc with no `## Log` section, the reason is
discarded:

```
$ kit drop 0001 "superseded by a better idea"
  dropped  docs/features/0001-nolog.md  (was Draft)
$ grep -c "superseded" docs/features/0001-nolog.md
0
```

`AppendToSection` reports that the heading was absent and `Drop` ignores the answer.
The command exists to record the reason — its help says so — and the one warning it
prints fires only when *no* reason was given, so the operator who supplied one is
never told it was thrown away.

## Non-goals
- **Making `needs:` mandatory.** An unanswered field is a real problem and a separate
  one; `BACKLOG.md` scopes it.
- **Blocking `accept`.** Accepting is a statement about the design, not a start.
  Blessing something that waits on another feature is normal.
- **A `--force`.** An escape hatch on the only remaining ordering constraint is the
  constraint becoming advisory, which is what 0021 did to `status:` on purpose and
  should not do here by accident.

## Design
`claim` and `ship` reuse `UnmetNeeds` — the same call `Survey` makes, so the board
and the commands cannot disagree.

`claim` refuses as `NotYours`, exit 1: the agent is meant to take something else,
which is the same instruction a lost race carries.

`ship` refuses as an ordinary error, exit 2. It is not "take something else" — the
work is done and the plan is wrong, so the operator has to look. The message names
which ids are unshipped rather than saying the feature is blocked.

**`ship` also refuses a feature claimed by a branch that is not this checkout's.**
`Survey` already computes exactly this as `Mine`: shipping from the worktree holding
the claim is the documented path and stays untouched, while shipping from anywhere
else is one agent freezing another's work. The message points at rule 8 — release the
claim first if the branch is a leftover — because refusing without naming the way out
is how an agent gets stuck.

`drop` does not refuse; it reports. A feature that will not be built is exactly when
you want to know who was waiting on it, and refusing would leave the dropped doc
stuck open. It lists the dependants and says they now need editing.

`next` stops promising a wait that will never end. Where a `needs:` id names a doc
that is Dropped or Superseded, the line says so instead of *"to ship first"* —
`audit` already distinguishes these two cases, and the board is what agents read.

## Tasks
- [x] `ClaimFeature` refuses unmet `needs:` as `NotYours`, naming the ids, with a
      conformance case for the refusal and one for a met dependency claiming fine
- [x] `Ship` refuses unmet `needs:`, exit 2, naming the ids; conformance both ways
- [x] `Ship` refuses a feature claimed by another checkout's branch, naming rule 8;
      conformance for the refusal and for shipping from the holding worktree
- [x] `Drop` reports features whose `needs:` name the dropped id; conformance case
- [x] `Drop` creates `## Log` when it is absent rather than discarding the reason;
      conformance case on a doc with no Log section
- [x] `Survey` distinguishes a dependency that can never ship from one not yet
      shipped, and says which; conformance case
- [ ] README and `kit help` for claim, ship and drop describe the new refusals

## Log
- 2026-08-16 — found bootstrapping url-shortener from an empty directory: the board
  refused 0003 and `kit claim 0003` took it anyway.
