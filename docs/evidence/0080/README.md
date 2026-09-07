# 0080 review evidence

Original: `237ab912c288`. Tested candidate: `456a8da`. metadata.json verifies every input against its pinned commit.

The candidate completed both bounded sessions with less reported context, but made factual and recovery errors. adoption.json records the original-wording fallbacks. Passing cases support only these reference tasks; they do not certify the full workflow.

| Run | Version | Exit | Minutes | Tool calls | Reported context tokens | Rubric |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| 1-A | original | 0 | 17.8 | 22 | 30,168 | 16/19 |
| 2-B | candidate | 0 | 10.2 | 13 | 20,424 | 17/19 |
| 3-B | candidate | 0 | 9.0 | 13 | 19,309 | 16/19 |
| 4-A | original | 1 | 20.0 | 19 | 33,255 | Incomplete: no artifacts |

No human intervened in the model sessions. The first baseline recovered from refused handoff writes. The final baseline repeatedly used the wrong handoff path after exceeding the 27,648-token session ceiling, then reached its 20-minute limit. Its missing artifacts are not observed factual errors. The server context was 49,152; runner-reported context and its handoff ceiling are distinct measurements.

The final combined references were not rerun through localcode. Source/accounting checks and the project gate validate the final files; the original wording is a conservative fallback, not proof of the combination’s behavior.

| File | Original words | Tested candidate | Final words |
| --- | ---: | ---: | ---: |
| docs/TECH.md | 5398 | 1992 | 2157 |
| templates/TECH.md | 378 | 142 | 142 |
| templates/features-TEMPLATE.md | 659 | 327 | 659 |
| templates/compatibility.md | 777 | 553 | 553 |
| templates/AGENTS-section.md | 1066 | 1066 | 1066 |
| cmd/kit/help.go | 4495 | 2477 | 3049 |

AGENTS.md, its embedded protocol and both language checklists remain byte-for-byte unchanged from the original. The separate 0078 protocol changes are outside this comparison.

- TASKS.md and rubric.md were fixed before the runs; the model did not receive the rubric.
- fixture-AGENTS.md contains the shared fixture instructions.
- metadata.json gives pinned inputs, exact argv, order and reproduction instructions.
- environment.json records model, runner, tool versions, settings and timing limitations.
- prose-map.json accounts for 265 baseline text units against the tested candidate; adoption.json records subsequent restorations.
- Each run directory contains its raw trace, account, available artifacts and per-criterion scores. Model outputs are evidence and may contain errors.

Restored: overview/release/block help, the feature template, and TECH registry/conformance/WaitDelay/CGO explanations. Other candidate passages retain their mapped contracts and the successful scoped checks.
