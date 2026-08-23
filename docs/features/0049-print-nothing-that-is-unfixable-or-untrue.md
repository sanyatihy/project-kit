---
id: 0049
title: Print nothing that is unfixable or untrue
status: Shipped
created: 2026-08-21
shipped: 2026-08-21
needs: 0048
---

## Problem

Five checks fire where nobody is allowed to act, and eleven statements in the shipped
prose are refuted by the code they describe. Both cost the same thing: an agent that
learns the tool can be ignored stops reading it, and being believed is the only thing
`audit` has.

## Non-goals

- **Judging whether a doc is any good.** `audit` reports what is checkable and never
  guesses. A check that cannot separate a healthy doc from a broken one is removed rather
  than tuned.
- **A prose linter.** The three writing checks are being removed, not replaced.
- **Editing the shipped corpus to silence findings.** A check that cannot separate a
  healthy doc from a broken one is deleted; the docs it fired on stay as written.

## Design

**A check that reports a state nobody may fix is deleted, not tuned.** Rule 11 freezes a
Shipped doc, so a finding whose only remedy edits one is a standing instruction to break
the protocol. Measured on `localcode`, whose 18 docs are the only corpus written under it:

| check | what it does | call |
| --- | --- | --- |
| `missing-section` | demands a non-empty `## Log` on Shipped, which the routing rule says is optional | delete |
| `draft-claimed` | reads any `## Open questions` text as open, including answered ones | delete |
| `doc-bloat` | offers a `## Log` escape hatch that exists in no line of code | delete |
| `session-narration` | skips Shipped, which is 13 of 18 docs, so it never reported its own corpus | delete |
| `long-sentence` | a sentence parser, 130 lines, for one LOW finding on frozen docs | delete |
| `needs-unknown` | walks a Dropped doc's `needs:`, reporting for ever | guard on terminal status |

**Deleting five checks deletes their machinery.** `NarrationMarkers` and the sentence
splitter are half of `parse.go` and serve nothing else.

**Every statement about behaviour is checked against the behaviour.** The exit-code
contract is wrong in two documents — `claim` exits 1 on a lost race, which nine tests pin.
The README enumerates checks by hand under a sentence claiming the list cannot drift, and
it has drifted twice: it invents `no-claim` and omits `duplicate-id`. The list is deleted
and `kit help checks` is cited, because the generated list is the one that cannot lie.
Rule numbers leave user-facing output: `ship` prints "rule 8" and `kit check --help` says
"rule 11", numbers valid in neither list a reader holds.

**`kit init` writes a file an agent can act on.** Today it writes an `AGENTS.md` that
opens on an `## Work protocol` heading with no title and no note that the section is
regenerated, so the promise that house rules live above the marker reaches nobody. The
file gains a project title and that note.

**Every state of `next` ends with the action for that state.** A doc straight out of
`kit new` lands in `blocked` and the closing line reads "stopping here is the right
outcome", when the work is to write the plan that was just created. `guidanceFor` gains
the case.

## Tasks

- [x] The five checks and their machinery are gone; `needs-unknown` skips terminal docs
- [x] The exit-code contract matches `claim`, and the README cites `kit help checks`
- [x] No user-facing string cites a rule by number, and a test covers the binary
- [x] `kit init` writes `AGENTS.md` with a title and the regenerated-section note
- [x] `next` names writing the plan when the only open feature is an unwritten draft
- [x] The audit summary line counts what it found and drops `thing(s)`
- [x] A freshly scaffolded project, filled in as documented, reports nothing at all

## Log
