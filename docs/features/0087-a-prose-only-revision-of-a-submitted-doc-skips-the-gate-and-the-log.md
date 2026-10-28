---
id: 0087
title: A prose-only revision of a submitted doc skips the gate and the Log
status: Draft
created: 2026-10-28
submitted:
needs:
---

## Problem

A reviewer asked for 0032's prose to be trimmed. The change touched no code and no task,
but it still cost a `kit reopen`, a full gate run and a fresh submit. `kit reopen` also
writes a Log entry, even though a trim changes no plan. That breaks the Log rule, and it
does so on the kit's own initiative.

## Non-goals

- Skipping the audit. A doc edit can still introduce a secret or a malformed frontmatter.
- Editing a doc that is Submitted on the trunk. It stays frozen (rule 18).
- Detecting prose-only edits in other docs or code. Only the feature's own doc qualifies.

## Design

- **Prose-only finish.** `kit finish` skips the gate when every commit since the last
  `Submit <id>` changes only that feature's doc and leaves its `## Tasks` byte-identical.
  It prints the skip and the Submit commit it relies on, then runs the strict audit,
  submits and pushes as usual. The gate passed on identical code, so a re-run proves
  nothing. Rejected: a `--prose` flag, which leaves the decision with the agent.
- **No Log from reopen.** `kit reopen` stops appending to `## Log`. The reason goes in
  the commit subject, `Reopen <id>: <why>`. A real plan change already gets its Log
  entry under rule 12, when the task is revised. Rejected: keeping the entry for
  task-changing reopens, because reopen cannot tell which kind it is.
- Rule 18 and `kit help reopen` say a prose-only revision needs no Log entry and no gate.
- Files: `internal/kit/finish.go`, `internal/kit/reopen.go`, `cmd/kit/help.go`,
  `templates/AGENTS-section.md`, `AGENTS.md`, `docs/TECH.md`.

## Tasks

- [x] `kit reopen` writes no Log entry and its commit subject carries the reason
- [x] `kit finish` skips the gate, and says so, when only the feature doc outside
      `## Tasks` changed since its last Submit; a code change or a Tasks change still runs it
- [x] Rule 18 and `kit help reopen` describe the prose-only route, with `AGENTS.md`
      regenerated in the same commit
- [x] `docs/TECH.md` records what the skip relies on

## Log
