---
id: 0082
title: A question lives in the feature doc it blocks, and the inbox is retired
status: Draft
created: 2026-09-10
submitted: 
needs:
---

## Problem

A question an agent cannot answer is split across two files. The feature doc holds
`## Open questions` as prose nothing parses, and `docs/INBOX.md` holds the line that
actually refuses the claim — carrying a feature id that must agree with a doc somewhere
else. Two homes for one fact is what the protocol's own writing rule forbids, and the
id field is a foreign key maintained by hand: `inbox-unparsed` exists only to report the
two ways it goes wrong, and one of them blocks work by prose match against whatever the
line happens to mention. A reviewer reading a submitted doc cannot see what was asked.

## Non-goals

- Questions detached from a feature. `kit block` already refuses an id with no doc, so
  none exist to migrate.
- A question during a spike. Recorded direct work has no doc and could never block; the
  outcome it records is where an unknown belongs until `kit harvest` names a feature.
- Changing what `BlockersFor` is consulted for. Its five callers — claim, submit, finish,
  next and action — keep their current refusals.

## Design

`## Open questions` becomes the only place a question lives, and the only thing that
blocks. An unticked box there blocks the feature the doc describes; ticking it means
answered. The id field disappears because the enclosing doc is the id.

```
- [ ] 2026-09-10 · Should codes expire? Leaning: no, keep forever.
```

`BLOCKING`/`FYI` goes with it. Every open question blocks — the marker distinguished a
question from a note, and a note belongs in the pull request or `docs/features/BACKLOG.md`.
`kit block --fyi` is removed rather than kept as a no-op, so a flag that once meant "do not
block" cannot read as still meaning it.

The date stays, and is the only parsed field. It feeds `question-stale`, which is what
makes a human answer at all; `InboxStaleDays` keeps its value and its name changes.

Only a dated box blocks. The prose bullet these sections carry today does not, which keeps
71 frozen docs of settled questions inert and needs no migration, but leaves an agent able
to write a question that reaches nobody. `question-informal` reports a bullet under the
heading that is not a dated box — the dateless box that can never age and the plain bullet
that never blocks, one cause and one line to correct. It is scoped to docs still open to
change: a Submitted or Dropped doc is frozen history, so a finding on one names nothing
anybody may fix. Together with `question-stale` it replaces `inbox-stale` and
`inbox-unparsed`, whose id-agreement failures are gone with the field.

`BlockersFor` reads the target doc's section instead of the inbox. The seam is unchanged,
so the five callers are untouched.

`kit block` commits the doc it edits, for the reason `kit submit` commits its stamp: the
write now lands in a feature doc, and left in the tree it dirties a trunk checkout that
`kit start` then refuses.

Retirement is a hard cutover. Nothing reads `docs/INBOX.md`, `kit init` stops creating it,
and `inbox-retired` reports the file where one survives — MEDIUM because the file instructs
an agent to append questions to it, and a question parked there now blocks nothing and
reaches nobody. `kit init` overwrites nothing it did not write, so the finding is the
migration: read the file once, then delete it. This breaks a
parsed document's meaning and lands before v1.0 freezes it, so `templates/compatibility.md`
records it in the upgrade table rather than the stable list.

## Tasks

- [x] `## Open questions` parses into blocking questions, and `BlockersFor` reads the
      target doc instead of the inbox
- [x] `kit block <id> "<question>"` appends to that doc's section, creating it when absent,
      and commits the edit; `--fyi` is removed
- [x] `question-stale` and `question-informal` replace `inbox-stale` and `inbox-unparsed`
      in the registry, with conformance cases
- [x] `inbox-retired` reports a surviving `docs/INBOX.md`, and `kit init` stops creating it
- [x] `internal/kit/inbox.go`, `templates/INBOX.md` and the kit's own `docs/INBOX.md` are
      deleted, and `ResolveID` moves to the file that keeps it
- [x] The protocol, `kit help block`, `kit help checks`, `kit new`'s next actions, the
      feature template and README describe the one home
- [x] `templates/compatibility.md` records the cutover and the migration step
- [x] `AppendToSection` finds a heading where the parser does, so a fenced example or a
      template comment never swallows a question
- [x] `kit block` refuses a terminal doc
- [x] `inbox-retired` and `templates/compatibility.md` describe a migration that can
      actually be followed

## Log

- 2026-09-10 — the migration is to read the file once and delete it, not to move its
  lines. Every stranded line in the eight projects carrying an inbox was an `FYI`, and
  nine of the ten named a feature already Shipped or Submitted. Moving those in would
  block work a human had cleared to continue, and `block` refuses a terminal doc anyway.
  The design assumed a per-line move that the only real corpus cannot take.
- 2026-09-10 — review found two defects and added the last two boxes. `AppendToSection`
  scanned raw lines, so a doc quoting the format in a fenced example took the question
  into the fence: `block` reported the feature blocked and `next` went on offering it,
  which is the one direction this mechanism must not fail in. And `block` had no status
  guard, because it never needed one while it wrote a file of its own — moving the write
  into the doc let it edit and commit frozen history.
- 2026-09-10 — boxes 1 and 2 land in one commit. `block` writing where nothing reads is
  not a state to leave the tool in, so the parse and its writer are one change.
- 2026-09-10 — boxes 5 and 6 land in one commit. Deleting `templates/INBOX.md` deletes
  the `kit help inbox` topic, and the overview offering a topic the binary cannot print
  fails the build, so the deletion and the prose that points at it are one change.
- 2026-09-10 — added a lock and an atomic write the design did not call for. Appending to
  a section is read-modify-write where appending to the inbox was one `O_APPEND`: twelve
  concurrent `block` runs lost four questions and handed a reader a doc truncated
  mid-write, which `kit block` then reported as the project having no feature docs.
- 2026-09-10 — reopen to Draft: review found kit block writing into a fenced example and into frozen docs
- 2026-09-10 — reopen to Draft: the inbox-retired advice would block cleared work and cannot be followed for frozen docs
