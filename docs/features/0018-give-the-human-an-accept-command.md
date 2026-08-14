---
id: 0018
title: Give the human an accept command
status: Shipped
created: 2026-08-14
shipped: 2026-08-14
check:               # optional — date to check whether this worked. Only for bets.
checked:             # date you checked; write the outcome in ## Log
needs: 0016, 0017    # all three edit AGENTS-section.md and README.md
related: docs/features/0013-never-leave-an-agent-without-an-instruction.md
---

## Problem

`Draft → Accepted` is the only transition in the whole model that belongs to a human. It
is the approval gate the design is built around — agents draft the whole set, nothing
starts until a person clears it — and it is the one transition with no command.

So the human hand-edits frontmatter. That is precisely what `kit ship` exists to stop
doing: 0005 added it because hand-stamping produced `unticked-shipped` and
`shipped-no-date`, two findings that are only ever self-inflicted. Accepting by hand has
the same shape and the same failure modes available to it — accepting a doc whose
`## Open questions` are still open, or one a BLOCKING inbox line is waiting on, both of
which put an agent to work on a design nobody settled.

It is also undefined *where* the edit happens. A draft waits on a branch of its own and
nothing is committed to `main` directly, so with `main` protected the flip has to reach
`main` through a pull request — either inside the draft's own PR, or as a second PR after
that one merges. The protocol never says which. One is a round trip per feature and the
other is two, and an agent reading the rules cannot tell the human what to do.

Separately, and for the same reason — the commands agents run should carry what agents
need: **`kit next` never mentions that the protocol is stale.** `protocol-stale` only
fires under `audit`, and the protocol tells agents to run `kit next` at the start of
every session, not `audit`. One binary on `$PATH` serves every project, so after a change
to the kit every other project is running superseded rules with nothing saying so. That
is the exact failure 0009 recorded — a real run against a binary two features behind,
where "nothing reported anything wrong, because from inside that project nothing was".

## Non-goals

- **Approving on the human's behalf, or any `--yes` that lets an agent accept.** The
  command exists to make one human action correct and cheap. An agent that could run it
  would remove the only gate in the model.
- **Reading review state from GitHub.** The kit knows about docs and refs. A PR approval
  is not a fact it can see, and inventing an integration for it would tie the tool to one
  host.
- **A `kit reject`.** The states already exist: leave it `Draft` with the question, or set
  `Dropped`. Nothing is ambiguous enough to need a command.
- **Warning about staleness on every command.** `next` is where a session starts. `claim`,
  `ship` and `block` stay quiet — a line repeated on every invocation is a line nobody
  reads, and 0009 already ruled out version-checking machinery that phones home. This
  compares two strings the binary already has.

## Design

### `kit accept <id>`

```
kit accept 0003
```

Sets `status: Accepted` on a `Draft`, and refuses in three cases, each because the doc
says the design is not settled:

| refusal | why |
|---|---|
| status is not `Draft` | nothing to accept; naming the current status is the whole answer |
| `## Open questions` is non-empty | rule 3 of bootstrapping: those are what `Draft` is for |
| a BLOCKING inbox line names the id | a human already owes an answer; `next` would refuse it anyway |

The first two are ordinary errors. The third reuses `BlockersFor`, so `accept` and `next`
agree by construction rather than by two implementations happening to match.

Written the same way `Ship` writes: replace the `status:` line, no other edit. Unlike
`Ship` there is no date to stamp — `created:` already records when it was proposed, and a
second date nobody reads is a field that goes wrong.

The closing instruction is where the round-trip question gets settled:

```
  accepted  docs/features/0003-x.md  (was Draft)

next:
  commit this on the draft's own branch so it lands in the same pull request
  main is protected, so the PR is how it gets there
  then: kit next
```

Accepting **inside the draft PR** is the answer, and the protocol says so too. It is one
review and one merge per feature instead of two, and the human is already reading the doc
at the moment they decide — which is when the branch is still open.

### `kit next` reports a stale protocol

`Audit` already computes it and `runNext` already has the embedded protocol available
through the same `currentProtocol()` helper `runAudit` uses. `next` runs the same
comparison and prints one line above the board when it differs:

```
  ! AGENTS.md is older than this kit — run kit init . before following it
```

Above the board, not below: an agent that reads the plan and starts working has already
missed a warning printed after it. It never changes the exit code — the plan is still
correct, the rules around it are what drifted — and it is suppressed under `--json`,
where the same fact belongs in a field rather than in prose.

### The protocol says who accepts, and how

Rule 1 of `## Picking and finishing work` currently ends "Leave every doc `Draft`:
accepting is a human's call." It gains the mechanism: the human runs `kit accept <id>` on
the draft's branch, that lands with the draft's own PR, and the branch is deleted on merge
because its name is the claim name.

## Tasks

- [x] `internal/kit/accept.go`: `Accept` flips a `Draft` and refuses on a non-Draft, on
      non-empty `## Open questions`, and on a BLOCKING inbox line, reusing `BlockersFor`;
      unit tests for all four paths.
- [x] `kit accept` wired into `main.go` with `commandHelp`, the overview, and exit codes
      matching the rest (2 for a refusal, since accepting is not a race); conformance for
      the happy path and each refusal.
- [x] `kit next` prints the stale-protocol line above the board and carries it in
      `--json`; conformance that it appears with an old AGENTS.md, is absent with a
      current one, and never changes the exit code.
- [x] Confine the status rewrite to the frontmatter. `statusLineRe` is `(?m)^status:.*$`
      applied to the whole document, so a `status:` line inside a fenced example is
      rewritten too. Found while designing `Accept`, which would otherwise be a second
      writer inheriting the same defect; shared with `Ship` rather than duplicated.
- [x] Protocol and README: the human accepts with `kit accept` on the draft branch so it
      lands in that PR, and the new command appears in README's command list.

## Log

- 2026-08-14 — the status rewrite became a fifth box rather than a silent fix. `Accept`
  needed a writer and the existing one was `(?m)^status:.*$` over the whole document, which
  also rewrites a `status:` line inside a fenced example. Duplicating a known defect into a
  second command was the only alternative, so `SetField` is shared by both and the
  conformance test drives `accept` and `ship` over the same doc.
- 2026-08-14 — `accept` reuses `BlockersFor` rather than reading the inbox again, so it and
  `kit next` cannot come to different conclusions about what is blocked. That also means
  `--fyi` is honoured for free, which has its own test.
- 2026-08-14 — no `accepted:` date. `created:` already records when the feature was
  proposed, and the second date would have had no reader.
