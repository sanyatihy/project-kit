---
id: 0051
title: A status the kit writes always means what it says
status: Draft
created: 2026-08-21
shipped:
needs:
---

## Problem

`ship` and `drop` are the only things allowed to write a status, which is why hand-editing
was removed. Both write with tools that can hit the wrong line, and neither refuses the
states where the write cannot mean anything, so a doc can end up carrying two contradictory
statuses while the command reports success.

## Non-goals

- **A YAML parser.** The frontmatter is a fixed set of `key: value` lines and stays that.
- **Repairing docs already written.** A doc in one of these states is reported, not rewritten.
- **Reopening frozen history.** Shipped stays frozen; this is about the write that freezes it.

## Design

**A terminal status is terminal in both directions.** `drop` refuses a Shipped doc as
frozen history; `ship` refuses only Shipped, so a Dropped feature ships. Reproduced: `kit
ship 0001` on a retired doc exits 0 and prints `(was Dropped)`. `Ship` refuses any terminal
status, with the message `drop` already uses.

**A frontmatter write never touches the body.** `ship`'s fallback for a doc with no
`shipped:` key is a document-wide `strings.Replace` — the shape `SetField` exists to
prevent, and `parse.go` still carries the comment saying why: docs for a tool whose subject
is frontmatter quote frontmatter constantly. `AddField` does exactly this job inside the
frontmatter bounds and has had no callers since `accept --review` was deleted. Use it.

**A key that appears twice is refused, not guessed.** `ParseFrontmatter` keeps the last
occurrence and `SetField` rewrites the first, so `ship` reports success, writes a line
nothing reads, and `audit` repeats the same finding with the same fix for ever. A duplicate
key is a HIGH finding and a refusal in every command that writes.

**An id that names two docs is refused everywhere or nowhere.** `claim` and `release` count
the matches and refuse; `ship`, `drop` and `block` take the first silently. `FeatureByID`
returns the first match, so the ambiguity is invisible to three commands that write. All
five refuse.

**The frontmatter block ends at a line, not at a substring.** The closing fence is found
with `strings.Index(text[4:], "\n---")`, which matches inside a longer run of dashes, so a
setext underline truncates the block and the fields after it are lost — surfacing only as
`no-status`, which names the wrong problem.

Reproduced:

| doc | what the kit did |
| --- | --- |
| `status: Dropped`, boxes ticked | `ship` exits 0, `(was Dropped)` |
| two `status:` keys | `ship` exits 0; `audit` repeats the same fix for ever |
| two docs, one id | `drop` retires the first, no warning |

## Tasks

- [x] `Ship` refuses any terminal status, and a test drives both directions
- [x] The `shipped:` fallback uses `AddField`; a fenced `status:` in the body survives a ship
- [x] A duplicate frontmatter key is a HIGH finding and a refusal in `ship`, `drop` and `block`
- [ ] `ship`, `drop` and `block` refuse an id that names two docs, as `claim` already does
- [ ] The frontmatter close is line-anchored
- [ ] A value containing `#` keeps its text — `title: Fix #123 crash` is not `Fix`

## Log
