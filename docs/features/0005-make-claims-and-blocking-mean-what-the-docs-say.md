---
id: 0005
title: Make claims and blocking mean what the docs say
status: Accepted
created: 2026-08-11
shipped:
check:
checked:
related: ../../cmd/kit/help.go
---

## Problem
An outside review of the kit — conducted against `kit help` and the subcommands,
without reading this repository — produced twenty-odd findings. Checking them against
the code turned up three real bugs, five places the protocol is silent, and something
more useful than either.

Four findings were factually wrong: that `INBOX.md` has no defined format, that
`kit next` ignores remote branches, that Accepted features have no defined order, and
that nothing exits non-zero on HIGH. All four are already handled. **But every one of
them is a fair reading of the binary.** `templates/INBOX.md` is not a registered help
topic, so the format the kit enforces is invisible to anyone who has only the tool.
`--strict` is absent from the Commands block. Rule 3 says "a branch named after the
doc" without saying that local branches, remote branches, and worktrees all count.
The tie-break among Accepted features is stated nowhere.

The kit's stated promise is that an agent never needs this repository to use the
tool. This review is evidence that the promise is broken in four specific places, and
that is worth more than the bug reports.

The bugs themselves: a question filed against one feature blocks a different one if it
mentions that ID in passing; a doc claiming `In progress` with no branch anywhere
audits clean; and a feature with no task boxes is reported as "every task is ticked".

## Non-goals
- **Auditing task quality.** Flagging boxes as vague or compound needs judgement no
  heuristic has, and would fire on good boxes. A noisy check trains people to ignore
  the tool, which is `audit`'s only asset.
- **Feature-size limits by task count.** The threshold would be arbitrary; `stalled`
  already covers the dimension that matters.
- **Retuning `stalled` to agent cadence.** Plausible, but there is no usage data yet.
  Revisit when a real feature has been missed.
- **Secret detection.** Rule 6 has no check and it is the rule with real blast radius,
  but entropy scanning is noisy on prose. If it is ever added it should be a handful
  of unambiguous patterns and nothing clever. Not here.
- **`kit claim`, `kit next --claim`, cross-feature dependencies, intra-feature
  parallelism.** All correctly deferred by the review itself. The signal to build
  them is real usage, not anticipation.

## Design
**Blocking reads the field, not the line.** `BlockersFor` scans the whole INBOX line
with `Mentions()`, so an ID mentioned in prose blocks a feature the question was never
about. The format already has the feature ID in a fixed position — the parser should
use it. This is a parsing fix, not a format change.

**`kit block` is part of the same change, not a follow-up.** Fixing the parser while
lines are still written by hand just moves the failure into hand-typed lines. The
command that writes the canonical format is what makes the strict parse safe.

**New audit checks go in the existing registry**, so `kit help checks` cannot drift and
`TestEveryRegisteredCheckIsExercised` forces a conformance case for each. The claim
checks are the ones that matter: `In progress` with no branch is a doc asserting
something untrue, which is precisely `audit`'s one job.

**Protocol gaps are prose, not machinery.** Releasing a claim, rewriting a box,
inserting urgent work, and pushing before working are each a line or two in the
regenerated section. They need no new state.

**Help comes last** because several boxes change what it has to say. The measure of
success is not a longer help text — it is that the same review, run again against the
binary alone, cannot reach those four conclusions.

## Tasks
- [x] Blocking reads the feature-ID field of an INBOX line, not the whole line
- [x] `kit block <id> "<question>"` appends a canonical INBOX line
- [x] Audit: `inbox-unparsed` — a line outside the documented format, found while fixing the parser
- [x] Audit: claim consistency — `In progress` with no branch, branch with no doc, two branches for one doc, a branch on a `Draft`
- [x] A feature with no task boxes reports that, in `next` and in `audit`, instead of "every task is ticked"
- [x] Audit: an INBOX question naming a feature ID that does not exist
- [x] Protocol: releasing a claim, rewriting a box, where urgent work is inserted, push before working
- [x] `kit ship <id>` — verify boxes, set status and `shipped:`, prompt the as-built migration
- [x] A `TECH.md` template, and `no-tech` pointing at it
- [ ] Tighten `kit help` so the four things this review got wrong are answerable from the binary

## Open questions
<!-- none -->

## Log
- 2026-08-11 — the four "wrong" findings were reclassified as help defects rather than
  analyst errors. Each is a reasonable inference from what the binary prints, and the
  kit claims an agent never needs this repo to use it. Treating them as reader error
  would have thrown away the only outside measurement of that promise there has been.
- 2026-08-11 — fixing the blocking parser and adding `kit block` are one box each but
  one change in intent. A strict parse over hand-written lines would trade a false
  positive for a silent false negative, which is worse: a question nobody sees.
- 2026-08-11 — rejected auditing task quality. It is the one proposal that would make
  `audit` less trustworthy rather than more, and `audit` has no value other than being
  believed.
