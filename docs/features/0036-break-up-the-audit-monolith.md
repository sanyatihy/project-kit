---
id: 0036
title: Break up the audit monolith
status: Shipped
created: 2026-08-17
shipped: 2026-08-18
check:
checked:
needs:
related: 0016-make-an-unfinished-doc-fail-the-audit.md
---

## Problem
`Audit()` is one 362-line function holding 34 inline `report(...)` calls — every check the
kit has, in one body, sharing one set of locals and one loop over the features.

The registry beside it is the opposite: `Check{code, severity, meaning}` values that
`kit help checks` prints from, so a severity can never drift from what is reported. The
data is well factored and the logic is not.

What it costs, in order of how often it bites:

- **No check can be exercised on its own.** Every conformance case runs the whole audit
  against a throwaway project, so a test for one check is really a test for all of them
  passing quietly. That is why the fixtures are as elaborate as they are.
- **Adding a check means editing the middle of a long function**, with its conditions
  interleaved with unrelated ones — three of the nested `if`s already guard four checks
  between them.
- **Reading it does not answer "what does `stalled` do".** The answer is spread across a
  status test, a claim test, a date fold, and a threshold.

## Non-goals
- **Changing what any check reports.** Codes, severities, messages and fixes stay
  identical; the conformance suite is the proof and it does not change either.
- **A plugin system, or checks in configuration.** The set is fixed and enumerable, and
  `kit help checks` prints it from the registry. Nothing here becomes extensible.
- **Splitting `repo.go` or `main.go`.** Both are long, neither is in the way. The kit's
  own rule is not to refactor what is not broken; this one is broken in a way the other
  two are not, because adding a check is a routine act and adding a command is not.

## Design
Each check becomes a function beside its registry entry:

```go
type check struct {
    Check
    run func(c *ctx) []Finding   // ctx carries r, today, protocol, and the feature loop's state
}
```

`Audit()` becomes the loop that walks the features and calls them, which is what the
function is actually for. Findings keep their current order, because `--strict` and the
printed report both depend on HIGH coming first and on stability between runs.

Two things the split has to preserve, and they are the reason it is worth doing at all:

- **`TestEveryRegisteredCheckIsExercised`** compares the conformance cases against the
  registry in both directions. With the logic beside the registry entry it can start
  asserting that every entry *has* a `run`, which is a stronger version of the same rule.
- **The shared work stays shared.** `Claims`, `UnmetNeeds` and the date folding are
  computed once per feature and handed to the checks that need them, or the split trades
  readability for the subprocess fan-out 0035 is removing.

Whether the per-feature checks and the whole-repo ones (`duplicate-id`, `no-plan`,
`protocol-stale`) want one shape or two is the open question the builder settles first;
the honest answer may be two small slices rather than one clever interface.

## Tasks
- [x] The per-feature checks move to functions beside their registry entries, with the
      shared per-feature work computed once and passed in
- [x] The whole-repo checks follow, in whatever shape the first box settled
- [x] `TestEveryRegisteredCheckIsExercised` also asserts every registry entry has an
      implementation, so a registered check can never be inert
- [x] `kit audit` output is byte-identical on this repo before and after, recorded in
      `## Log`

## Log
- 2026-08-17 — `golangci-lint` is clean and there is no dead API; this is the one
  structural finding worth a feature.
- 2026-08-18 — open question settled: two slices, not one signature. They take different
  arguments and share nothing but the registry entry, so unifying them would be an
  abstraction with two implementations and no caller that wants either.
- 2026-08-18 — an entry names every code it can report, rather than one code per function.
  Some checks are one decision with two outcomes — a status missing or unknown, a task list
  empty or finished — and splitting those would evaluate the same condition twice.
- 2026-08-18 — output is byte-identical on a fixture firing 10 checks across both formats,
  not on this repo, whose audit is clean and would have proved nothing.
- 2026-08-18 — the split removed a `continue` in `stalled` that skipped every later check
  for that feature, not just the ageing. Unreachable in practice — it needed both liveness
  signals zero, and `LastTouched` falls back to mtime — which is why nothing caught it.
