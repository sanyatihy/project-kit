---
id: 0009
title: Fix what the first real run found
status: In progress
created: 2026-08-12
shipped:
check:
checked:
related: 0008-put-a-check-behind-the-secret-rule.md
---

## Problem
An agent built a working URL shortener from one prompt in twenty minutes, following
the protocol without being reminded of it: one commit per box, divergences from the
design doc logged rather than absorbed, `kit audit` clean at the end. The loop works.

It also found three defects, and one of them is mine from yesterday.

**`secret-in-doc` fires on a doc doing the right thing.** The feature doc explains
that URLs carrying userinfo are rejected as a phishing shape, and quotes the form to
say so. That is HIGH, so `--strict` fails a build over documentation that is correct.
0008's own design says every pattern is a format whose *issuer* publishes a fixed
prefix or envelope; seven are, and the URL one is a structural shape that anything
discussing URL credentials will match. The rule was right and the eighth entry broke
it. 0005 named this exact failure: a noisy check trains people to ignore the tool.

**Nothing says who merges.** The protocol covers claiming, working, and shipping, and
then stops. The agent reached the end of `kit ship`, found no instruction, and merged
its own work to `main` unreviewed. It filled a vacuum sensibly; the vacuum is the bug.

**The binary was two features behind.** `kit` on `PATH` was built from the 0006 merge,
so `kit init` wrote no `docs/VISION.md`, the generated protocol had no rule about
drafting from it, and the secret check never ran. Everything 0007 and 0008 shipped
went untested by the run that was meant to test it. `make install` is manual and
nothing re-runs it, so a merged change reaches other projects only when somebody
remembers.

## Non-goals
- **Narrowing the URL pattern instead of removing it.** Distinguishing a real
  password from `pw` in an example needs a placeholder heuristic, which is the
  "clever" 0005 ruled out and would still be wrong at the edges.
- **Making the kit enforce who merges.** `audit` never writes and never refuses. The
  protocol states the boundary and `kit ship` says it at the moment it matters;
  neither can stop anyone, and neither should try.
- **Version-checking machinery.** A binary that phones home, compares itself to a
  remote, or warns on every run buys less than it costs. This is an operational habit
  with a documentation fix.
- **Re-litigating one feature for one project.** Six layered boxes held for one agent
  for twenty minutes, so rule 1 did its job. Nothing to change.

## Design
**Delete the eighth pattern.** `secret-in-doc` keeps exactly the shapes anchored to an
issuer's published prefix or envelope. The cost is stated plainly rather than hidden:
a real password inside a connection string in a doc now goes unflagged. Rule 10
already says the check is a backstop and not the rule, and a password written in prose
was always invisible to it — this makes the boundary honest instead of ragged.

The quiet test missed it by using a connection string with *no* password, so the
regression case is the doc that actually broke it: prose quoting a userinfo URL to
explain that it is rejected.

**Shipping is a handoff, and the protocol should say so.** A new rule: `kit ship`
ends the agent's part — push the branch and stop. Merging is not the agent's to do.
`kit ship` prints it too, because prose in `AGENTS.md` is read at the start of a
session and this decision is made at the end of one. Where there is no remote to push
to, that is worth saying out loud rather than letting the work quietly land unreviewed.

**A house rule for the kit's own repo.** Merging a change to `kit` does nothing for
any other project until the binary is reinstalled, and the one experiment that would
have caught two features' worth of behaviour ran against a stale build. This belongs
in the kit's own `AGENTS.md` house rules and in `README.md`'s section on changing the
kit — the two places someone working on the kit actually reads.

## Tasks
- [x] `secret-in-doc` drops the URL shape, with the doc that broke it as the regression case
- [ ] Protocol and `kit ship`: shipping hands off, merging is not the agent's, and say when there is no remote
- [ ] House rule: reinstall after merging, or every other project keeps running the old kit

## Open questions
<!-- none -->

## Log
- 2026-08-12 — removed the URL shape rather than tuning it. The deciding argument was
  that 0008 already stated the admission rule — an issuer-published prefix or
  envelope — and the entry that failed in practice is the one entry that never met it.
  Tuning would have kept a pattern the design does not justify.
- 2026-08-12 — the run that found all three was worth more than the feature it built.
  Two of the three are invisible from inside the repo: nothing in the kit could have
  told us the protocol had no answer for merging, or that the binary was stale.
