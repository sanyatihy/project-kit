---
id: 0029
title: Mark the features whose implementation a human must see
status: Shipped
created: 2026-08-16
shipped: 2026-08-17
needs: 0028
related: 0018-give-the-human-an-accept-command.md
---

## Problem

Branch protection is the only enforceable review policy and its granularity is the whole
project. Protect the trunk and a log-field rename waits on a human; leave it open and the
billing change merges unseen. There is nowhere to say "this one", and 0028's board gives a
coordinator no way to know which entries are not its to merge.

## Non-goals

- Replacing branch protection. It stays the floor and the only thing that stops a
  badly-behaved agent.
- Letting an agent set it. Then it is a control the controlled party writes.
- Inferring risk from the diff. Size, paths and dependency changes all correlate with
  wanting a look and none of them decide it.

## Design

An optional frontmatter field on a feature doc, `review:`, in the shape of `check:` — the
existing precedent for an optional per-feature decision only a human makes:

```
review: human        # a person merges this one, whatever the trunk allows
```

Empty means the coordinator may merge it. Absent behaves as empty, so every existing doc
keeps working and no migration is needed.

Set when the round is accepted rather than when the feature is built: the moment the
operator has the whole set in front of them, before anyone has sunk work into the outcome.
`kit accept <id> --review` writes it, through the command that already belongs to the human.

The board carries it, so a coordinator agent has both facts — what is not mine to merge, and
what is nobody's but a person's.

`audit` reports nothing about it. There is no drift to detect: a coordinator that ignores
the mark produces no state the kit can see. What makes it hold is that the field arrives on
the trunk through a pull request a human merged.

## Tasks

- [x] `review:` parsed on `Feature`, empty and absent both meaning the coordinator may merge
- [x] `kit accept <id> --review` writes the field
- [x] The awaiting-review board marks entries needing a human, in text and `--json`
- [x] `templates/features-TEMPLATE.md` documents the field beside `check:`
- [x] The protocol says who sets it, that a coordinator honours it, and that branch
      protection is what enforces it
- [x] README describes the two-level policy: protection as the floor, `review:` as the
      per-feature exception

## Log

- 2026-08-17 — `SetField` claimed in its comment to add a missing key and did not, which
  matters here because no doc written before today carries `review:`. `AddField` does the
  insertion, and `SetField` keeps refusing: a doc with no `status:` is malformed, and
  `accept` and `ship` should say so rather than quietly writing one in.
- 2026-08-17 — the mark is read from the checkout, not from the branch, which is the opposite
  of the status 0028 reads. It is set when the round is accepted, so it reaches the trunk
  with the doc, and a builder that strips it on its own branch changes nothing.
- 2026-08-17 — rule 13 got shorter, not longer. Adding the mark took the protocol over its
  line ceiling, and the reasoning behind separation of duties moved to `docs/TECH.md`, which
  is read once rather than every session.
