---
id: 0029
title: Mark the features whose implementation a human must see
status: In progress
created: 2026-08-16
shipped:
check:
checked:
needs: 0028
related: 0018-give-the-human-an-accept-command.md
---

## Problem
Branch protection is the only enforceable review policy, and its granularity is the
whole project. Real projects are not uniform: in one repo the operator wants to read
the change that touches billing and not the one that renames a log field. Protect the
trunk and every trivial change waits on a human; leave it open and the billing change
merges unseen.

There is nowhere to say *this one*. The awaiting-review board 0028 adds shows a
coordinator everything that is waiting and gives it no way to know which entries are
not its to merge.

## Non-goals
- **Replacing branch protection.** It stays the floor and the only thing that stops a
  badly-behaved agent. This decides what a well-behaved coordinator does *within*
  whatever the trunk allows, and the protocol says which is which.
- **Letting an agent set it.** Then it is a control the controlled party writes. It is
  set by whoever accepts the plan, in the round they are already reading.
- **Inferring risk from the diff.** Size, paths and dependency changes all correlate
  with wanting a look and none of them decide it. That is the "judge whether work is
  any good" line, and it moves here for the same reason it held there.

## Design
An optional frontmatter field on a feature doc, `review:`, in the shape of `check:` —
which is the existing precedent for an optional per-feature decision that only a human
makes:

```
review: human        # a person merges this one, whatever the trunk allows
```

Empty means the coordinator may merge it. Absent behaves as empty, so every existing
doc keeps working and no migration is needed.

**Set when the plan round is accepted, not when the feature is built.** That is the
moment the operator has the whole set in front of them and is already deciding, and it
is a decision made before anyone has sunk work into the outcome. `kit accept <id>
--review` writes the field, so the marking happens through the command that already
belongs to the human rather than by hand.

**The board carries it.** 0028's awaiting-review entries show `needs a human` against
the marked ones. Together with 0028's "you built this" marking, a coordinator agent has
the two facts it needs: what is not mine to merge, and what is nobody's but a person's.

**`audit` reports nothing about it.** There is no drift to detect — the field says what
somebody decided, and a coordinator that ignores it is not producing a state the kit
can see. That is the forge's job, and the protocol says so plainly instead of implying
the kit is a gate.

The honest limit, stated in the protocol rather than left to be discovered: this is
legibility. An agent can edit the field, exactly as it can skip any other check. What
makes it hold is that the field arrives on the trunk through a pull request a human
merged, so changing it afterwards is another pull request — visible in the same place
the work is.

## Tasks
- [x] `review:` parsed on `Feature`, empty and absent both meaning the coordinator may
      merge; conformance that existing docs are unaffected
- [x] `kit accept <id> --review` writes the field; conformance for with and without
- [x] The awaiting-review board marks entries needing a human, in text and `--json`
- [x] `templates/features-TEMPLATE.md` documents the field beside `check:`
- [x] The protocol says who sets it, that a coordinator honours it, and that branch
      protection is what enforces it
- [ ] README describes the two-level policy: protection as the floor, `review:` as the
      per-feature exception

## Log
- 2026-08-16 — scoped after per-project protection turned out too coarse: within one
  project the operator wants to see some implementations and not others.
- 2026-08-17 — `SetField` claimed in its comment to add a missing key and did not, which
  matters here: no doc written before today carries `review:`. `AddField` does the
  insertion, and `SetField` keeps refusing — a doc with no `status:` is malformed, and
  `accept` and `ship` should say so rather than quietly writing one in.
- 2026-08-17 — the mark is read from the checkout, not from the branch, which is the
  opposite of the status 0028 reads. It is set when the round is accepted, so it reaches
  the trunk with the doc — and a builder that strips it on its own branch changes
  nothing, because that is not the copy anybody reads.
- 2026-08-17 — rule 13 got shorter, not longer. Adding the mark took it over the
  protocol's line ceiling, and the fix was to move the reasoning behind separation of
  duties — distinct identities, protection as the enforcer — to `docs/TECH.md`, which is
  read once rather than every session. The protocol keeps the rule and points at the why.
