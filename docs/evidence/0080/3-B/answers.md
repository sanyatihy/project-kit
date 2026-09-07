# Answers

1. **No — do not delete its branch.** Exit 1 is a completed negative result: another agent holds the claim, and a claimed branch or worktree belonging to another agent is not yours to clean up. Recovery never tells a losing claimant to delete the winner's branch; only a confirmed competing winner permits cleanup of an unchanged, unoccupied losing attempt — your own. Run `kit next` and choose different work.

2. **Retry `kit claim` from this same clone once the remote answers.** A lost push acknowledgement preserves a unique receipt, and retry reconciles it with the remote; confirmed ownership survives rebases. Do not hand-delete refs or infer ownership from matching starting commits, and do not treat the unconfirmed push as an acquisition — an unknown push outcome is exit 2 until the remote answers.

3. **No, the claim is not lost.** A worktree failure cannot undo a won claim; preserve the obstructing files (other files are preserved, never cleared) and retry `kit claim <id>`. Existing matching worktrees are reused, so the retry completes the acquisition in `.worktrees/<branch>`.

4. **Commit the plan change in its own commit with the reason in Log, then re-run the gate, `kit audit --strict`, and `kit submit <id>`.** Submit refuses unfinished Tasks boxes; a plan-only unmerged commit is work (doc-only commits count), so it must land before the stamp. If the feature was already submitted for review, use `kit reopen <id> "<why>"` first to return it to Draft, then repeat gate → audit → submit.

5. **No — it is not complete.** An explicit offline audit covers local checks only and labels its reduced coverage; neither an offline pass nor an unreadable remote establishes remote absence. Complete the online audit before handing off: strict online audit fails on an unanswered remote, so fetch/restore connectivity until the remote answers.

6. **Do not reopen it — draft a new feature citing the frozen doc.** A Submitted doc on trunk is integrated and frozen history; further changes need a new doc that cites it by id instead of copying its reasoning. `kit reopen` revises only an unmerged submission, and it refuses docs already Draft or integrated.

7. **Take the independent approved feature from another trunk checkout** (each agent works in its own checkout). The blocked feature's BLOCKING inbox line stops `next` offering it; meanwhile take independent work, or report the blocker and stop. Never invent a product decision to unblock yourself.

8. **Run `kit init .` on the populated project.** It creates only missing files — never clears the target first — and re-running refreshes only AGENTS.md's Work protocol section. Preserve the custom Makefile (init does not update existing stack files; compare it against current scaffolds) and keep house rules outside the Work protocol section so they survive the refresh.

9. **The identifier is defined in the audit registry in `audit.go`** — `featureChecks` and `repoChecks`, from which help is generated. Every registered check must have one implementation and a conformance case, so adding the check obliges both. A prose rule alone is not sufficient to enforce human-only merge: agents can bypass kit with raw git, and shared credentials defeat identity separation — remote permissions and required checks enforce that boundary; prefer enforcement in the tool over another prose rule.

10. **Set `exec.Cmd.WaitDelay` within the 15-second budget.** Killing git alone can leave a subprocess helper holding its output pipe; WaitDelay bounds the whole call (a black-holed remote once took 75s; WaitDelay reduced it to 15.05s). No — race tests must not inherit the stripped build's `CGO_ENABLED=0`: build with CGO_ENABLED=0, -trimpath and stripped ldflags, but do not export CGO_ENABLED=0 into race tests.
