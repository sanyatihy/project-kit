# Documentation rehearsal

Use the supplied reference files to complete these three tasks. They are documentation tasks in this disposable folder, not permission to implement or publish anything. Read only what you need. Write the requested artifacts, then finish. Do not ask for clarifications: all product decisions needed below are supplied.

1. Write answers.md, numbered 1–10, giving the next action and reason for each scenario:
1. claim 7 returned exit 1 after another agent won. Can you delete its branch and retry?
2. A claim push timed out without confirming ownership. A local receipt exists. What now?
3. The claim won, but creation of its worktree reported obstructing files. Is the claim lost?
4. Your branch has one unmerged commit changing only the plan. release refused. The work is valuable. What now?
5. An offline audit is clean; the remote cannot be read. Is online validation complete?
6. Submitted work is already on trunk. The operator wants another behavior change. Reopen it?
7. One feature needs a human decision; another approved feature is independent and ready. What should the agent do?
8. A populated project needs its protocol refreshed. It has custom Makefile and AGENTS house rules. How should init be used and what must be preserved?
9. A new audit check is needed. Where is its identifier defined and what test obligation applies? Is adding a prose rule sufficient to enforce human-only merge?
10. git CommandContext times out, but a subprocess helper retains stdout. What additional bound is needed? Can Go race tests inherit the stripped build's CGO_ENABLED=0?

2. Write plan.md using feature-template.md. The operator requests one reviewable feature: support operators can resolve repeated customer lookups from an in-process cache instead of making the same upstream request twice within 30 seconds. The agreed design uses a per-process cache, accepts up to 30 seconds stale data, and rejects a shared cache because cross-worker invalidation is unavailable. It depends on prerequisite 0041's lookup API, which is not integrated yet. Feature 0042 merely edits the same file; there is no dependency on its decisions. Use id 0043. Do not implement. Record the benefit, dependency, accepted cost, material non-goal, and ordered verifiable task outcomes. Product decisions are complete.

3. Write TECH-result.md using as-built-template.md from these verified facts after implementation: a Go CLI stores bookmarks in one SQLite file; SQLite was chosen over JSONL for atomic updates, accepting a native dependency; a trailing space in imported titles previously broke exact lookup, so the import boundary trims it. make check passes; bm list is the local read command. Cloud sync is absent. Yesterday's investigation involved three experiments and a failed benchmark, but produced no other system fact. Include current facts and accepted costs; do not invent constraints.
