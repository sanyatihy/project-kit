# 0080 review evidence

Baseline: `237ab912c288`. Candidate inputs are identified by SHA-256 in metadata.json.

- prose-map.json: 265 baseline paragraphs/bullets, destinations and omission decisions.
- TASKS.md and rubric.md: predefined tasks, scoring and limits.
- metadata.json: input hashes, sizes, session order and localcode command.

AGENTS.md, templates/AGENTS-section.md and both language checklists match the baseline byte-for-byte. The 0078 protocol changes belong to a separate branch.

Static mapping does not prove behavioral equivalence. Paired session results are pending.

| File | Baseline words | Candidate words |
| --- | ---: | ---: |
| docs/TECH.md | 5398 | 1992 |
| templates/TECH.md | 378 | 142 |
| templates/features-TEMPLATE.md | 659 | 327 |
| templates/compatibility.md | 777 | 553 |
| templates/AGENTS-section.md | 1066 | 1066 |
| cmd/kit/help.go | 4495 | 2477 |
