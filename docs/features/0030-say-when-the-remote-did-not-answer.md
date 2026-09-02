---
id: 0030
title: Say when the remote did not answer
status: Shipped
created: 2026-08-17
shipped: 2026-08-17
needs: 0029
related: 0028-see-what-is-waiting-for-review.md
---

## Problem

The review board and `shipped-not-pushed` both need the remote's branch list, and
`remoteHeads` bounds that call at 5 seconds. When it expires the answer is unknown, and
unknown is printed as nothing — the same page a project with nothing waiting gets.

Measured on this repo, minutes after 0029 shipped:

```
$ time git ls-remote --heads origin      5.58s total
$ kit next                               "nothing is planned yet."
$ kit next   # with the bound at 30s     awaiting review  0029 …
```

GitHub over SSH from this machine does not fit inside the bound, so the board went blank
while a feature sat pushed and unmerged — the failure 0028 exists to report, reintroduced
one layer down and harder to notice, because nothing is wrong on the page. `audit` is
silent the same way, so the HIGH that catches work stranded on one machine never fires and
`--strict` stays green.

## Non-goals

- Removing the bound. `kit next ~/code/*` fans out over every project, and one unreachable
  remote must not hang the board.
- Caching the answer between commands. State on disk the kit does not own, and it would
  report a claim from a mirror without saying so.
- Making it configurable. The kit has no config, deliberately.
- Fetching. The kit still never mutates refs behind an agent's back.

## Design

The bound goes to 15 seconds, a number a working-but-slow remote fits inside, with the
5.58s measurement in the comment beside it. Still bounded, so the fan-out argument holds.

Expiry is reported rather than swallowed. `remoteHeads` already distinguishes "no remote"
from "did not answer"; only the caller throws it away.

| where | what it says |
|---|---|
| `kit next` | a line under the board: the remote did not answer, so what is waiting may be missing |
| `kit next --json` | a field beside `awaiting_review`, so an agent reads the same fact |
| `kit audit` | the checks it could not run, rather than a clean bill |

This is the `RefsAreAuthoritative` precedent applied to the case where there is no answer at
all. Silence is the one output that cannot be told from good news.

`--offline` stays silent, being the reader's own instruction rather than a failure.

## Tasks

- [x] `remoteHeads`' bound is 15s, with the measurement recorded beside it, and callers can
      tell "did not answer" from "no remote at all"
- [x] `kit next` says the board may be incomplete when the remote did not answer, in text
      and `--json`
- [x] `kit audit` names the checks it could not run instead of reporting nothing
- [x] A remote that cannot be reached produces the notice rather than a blank board, and
      `--offline` stays quiet
- [x] README and `kit help next` say what an unanswered remote does to the board

## Log

- 2026-08-17 — the bound was the smaller half. Reporting expiry as an empty page is worse,
  because absence and ignorance render identically.
