# Feedback from a 25-session local-model chain

A Qwen3.8-27B chain ran `kit` unattended for 25 sessions against a fresh project
(`slack-stats`), planning and shipping six features with no human in the loop. Every
finding below is something that chain did, with the commit or command that shows it.

The headline: **kit's refusals held, and its audit was actionable enough that the model
repaired its own damage unaided.** What failed was the gap between a command mutating the
working tree and anything making that mutation durable.

## 1. `kit ship` leaves the status stamp uncommitted — HIGH

`kit ship <id>` writes `status: Shipped` into the doc and stops. Nothing commits it, and
nothing warns that it is loose.

What happened: 0002 was shipped in session 4. The stamp was never committed. Three
sessions later a fresh session saw the dirty doc, reasoned that it was "a stray frontmatter
edit, already committed", and discarded it. `git log -p` shows 0002's only ever committed
status was `Draft`. A shipped feature silently reverted, and nothing noticed for six
sessions — until `kit audit` reported `done-not-shipped`.

Reproduced outside the chain: `kit ship 0028` in another repo left
`M docs/features/0028-*.md` in the working tree with no mention that a commit was owed.

The failure needs two agents, or one agent with no memory of the last session — which is
exactly what a chain is.

**Suggested fix**, cheapest first:
- `kit ship` commits the stamp itself, the way `kit claim` already pushes a branch.
- Or it refuses when the tree is dirty, so the stamp is never mixed with other edits.
- Or `kit audit` gains a `shipped-not-committed` check: status is `Shipped` in the file but
  not in `HEAD`.

Any of the three would have prevented the loss. The third is the smallest and catches the
general case.

## 2. An invalid `status:` has no repair that is not a hand-edit — HIGH

`status: In Progress` entered a doc in commit `7cf767a`. `kit` never writes that value —
`grep -rn 'In Progress'` over this repo returns nothing — so an agent wrote it by hand,
against the rule that says only `ship` and `drop` change a status.

`kit audit` catches it: `unknown status "In Progress"`, `fix: use one of: Draft, Dropped,
Shipped`. But that instruction *is* a hand-edit of `status:`, which is the thing the rule
forbids. The chain complied literally and committed `docs: fix 0003 status to valid value
before shipping` — repairing the damage with the same mechanism that caused it.

`kit ship` also refused while the status was unknown, so the doc was wedged: it could not
be shipped, and the only way out was the forbidden edit.

**Suggested fix:** give a bad status a repair path that is a command. Either let
`kit ship` overwrite any unrecognised status (it is authoritative about status anyway), or
add `kit reset <id>` that returns a doc to `Draft`. Then the audit's `fix:` line can name a
command instead of an edit.

## 3. The `needs:` comment argues for empty, and empty was wrong — MEDIUM

The template comment above `needs:` reads: *"Leave it empty if this could be built today
against what already exists — that is the common case, and several empty ones is what lets
agents work in parallel."*

The chain drafted three docs and left `needs:` empty on all three. Two were wrong on the
evidence of their own prose: 0002's Problem says *"without this, 0003's charts have nothing
to aggregate"*, and 0003's says *"The numbers exist after 0001+0002"*. Both name the
dependency in the sentence above the field they left blank.

The cost was real. 0002 was claimed off a `main` that did not yet contain `parse.py`, so
the next session spent its budget hand-merging the 0001 branch into trunk to carry the
parser forward — and in doing so broke the rule about not merging your own work.

Notably, the same chain got `needs:` **right** on its second plan round (`needs: 0004` on
0005) once it had felt the consequence. The nudge misleads a first-time planner
specifically.

**Suggested fix:** two independent changes.
- Soften the comment. State the test ("could this start before that ships?") without
  pre-answering it with "that is the common case".
- Add a LOW audit check: a doc whose `## Problem` or `## Design` names another feature id
  that `needs:` does not list. It is a grep, and it would have caught both docs.

## 4. `kit new` still emits template prose — LOW

After the frontmatter prose was stripped from this repo's docs, `kit new` in a consuming
project still emitted the full comment blocks in both frontmatter and body. The consuming
repo stripped them by hand a second time. The template the binary carries and the docs in
this repo have drifted apart.

## 5. Nothing notices an agent merging its own work — LOW

`kit claim` correctly says *"this project has no remote, so no other agent can see it"*.
But with no remote there is also no pull request, and the chain merged its own feature
branches into `main` four times rather than stopping at the branch. That is a protocol
rule, not a kit rule, so this may be out of scope — but `kit ship` already knows the branch
name and could note when it is already reachable from `main`.

## What worked, and is worth protecting

- **`kit audit` findings were repairable by a 27B model with no human help.** All three
  MEDIUM findings were found and fixed across two sessions. The `fix:` line naming an exact
  command is why; the one finding whose fix was prose (§2) is the one it botched.
- **`kit next` refusing to hand out a claimed feature held every time.** No session ever
  worked a feature it had not claimed.
- **`kit new` reserving ids by pushing a ref worked** across 6 features and 25 sessions with
  no collision.
- **`kit ship` demanding `docs/TECH.md` be touched** produced a real as-built doc rather
  than an empty one. The chain filled it because it could not ship otherwise.
- **`kit init` refusing to do anything before `VISION.md` existed** made the model write a
  vision grounded in the data it had just measured, instead of inventing features first.
