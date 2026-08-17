---
id: 0030
title: Say when the remote did not answer
status: In progress
created: 2026-08-17
shipped:
check:
checked:
needs: 0029
related: 0028-see-what-is-waiting-for-review.md
---

## Problem
The review board and `shipped-not-pushed` both need the remote's branch list, and
`remoteHeads` bounds that call at 5 seconds. When it expires the answer is *unknown*,
and unknown is printed as nothing — the same page a project with nothing waiting gets.

Measured on this repo, minutes after 0029 shipped:

```
$ time git ls-remote --heads origin      5.58s total
$ kit next                               "nothing is planned yet."
$ kit next   # with the bound at 30s     awaiting review  0029 …
```

GitHub over SSH from this machine does not fit inside the bound, so the board went
blank while a feature sat pushed and unmerged. That is the failure 0028 exists to
report, reintroduced one layer down and harder to notice: nothing is wrong on the page.

`audit` is silent the same way. `shipped-not-pushed` needs the same call, so on this
connection the HIGH that catches work stranded on one machine never fires, and
`--strict` stays green.

## Non-goals
- **Removing the bound.** `kit next ~/code/*` fans out over every project, and one
  unreachable remote must not hang the board. That is what the timeout is for.
- **Caching the answer between commands.** State on disk the kit does not own, and it
  would report a claim from a mirror without saying so.
- **Making it configurable.** The kit has no config, deliberately.
- **Fetching.** The kit still never mutates refs behind an agent's back.

## Design
Two changes, and the second is the one that matters.

**The bound goes up**, to a number a working-but-slow remote fits inside — 15 seconds,
with the 5.58s measurement in the comment beside it so the next person tuning it knows
what it was set against. Still bounded, so the fan-out argument holds.

**Expiry is reported, not swallowed.** `remoteHeads` already distinguishes "no remote"
from "did not answer"; only the caller throws it away. Both callers say so instead:

| where | what it says |
|---|---|
| `kit next` | a line under the board: the remote did not answer, so what is waiting may be missing |
| `kit next --json` | a field beside `awaiting_review`, so an agent reads the same fact |
| `kit audit` | the checks it could not run, rather than a clean bill |

This is the `RefsAreAuthoritative` precedent — the kit already says when a claim was read
from the last fetch rather than the remote — applied to the case where there is no answer
at all. Silence is the one output that cannot be told from good news.

`--offline` stays silent: that one is the reader's own instruction, not a failure.

## Tasks
- [x] `remoteHeads`' bound is 15s, with the measurement recorded beside it, and callers
      can tell "did not answer" from "no remote at all"
- [x] `kit next` says the board may be incomplete when the remote did not answer, in
      text and `--json`
- [x] `kit audit` names the checks it could not run instead of reporting nothing
- [x] Conformance: a remote that cannot be reached produces the notice rather than a
      blank board, and `--offline` stays quiet
- [ ] README and `kit help next` say what an unanswered remote does to the board

## Log
- 2026-08-17 — found on the first real run of 0028's board: `kit next` said "nothing is
  planned yet" while 0029 sat pushed and unmerged, because `ls-remote` took 5.58s against
  a 5s bound.
