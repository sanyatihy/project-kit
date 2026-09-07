# Answers

1. **No.** Exit 1 means another agent won; a claimed branch or worktree belonging to another
   agent is not yours to clean up, and deleting refs by hand never decides ownership (a
   printed `git push --delete` once let a loser take the winner's branch). Run `kit next`
   and choose different work.

2. **Retry `kit claim <id>` from that clone.** An unknown push outcome preserves the attempt:
   the local receipt is the acquisition evidence, retry reconciles it with the remote, and a
   confirmed competing winner would return exit 1. Do not decide ownership yourself or delete
   refs by hand; wait until the remote is reachable.

3. **No.** A worktree creation warning means the claim is already held: making the worktree
   cannot fail a claim the push has won. Preserve any obstructing files, then retry `claim` —
   a path already holding the claim branch counts as success.

4. **Do not discard it.** A commit changing only the plan is still work, and `release` refuses
   unmerged commits without `--discard`, which would delete them. Push the branch and open the
   pull request unfinished, saying what remains — the doc travels with the branch, so the
   record lands when it merges.

5. **No.** An explicit offline audit covers local checks only and names the checks it dropped;
   unreadable remote evidence cannot certify current ownership, publication or integration.
   Complete the online audit before handing off to review — a failed or incomplete check is
   not a pass.

6. **No.** A Submitted doc on the trunk is frozen history; `reopen` refuses an integrated
   feature and its work is already in the repo. The new behavior change is new work: draft a
   new feature citing the frozen one by id, for human approval through the plan round.

7. **Work the ready feature.** One agent per feature, one feature per checkout; `kit next`
   skips the blocked doc and offers what it can accept. The blocked feature stays parked —
   its decision goes in INBOX via `kit block`, never invented to unblock yourself. If nothing
   free remained, report the blocker and stop.

8. **Run `kit init`** — it refreshes only the real `## Work protocol` section of AGENTS.md.
   Preserve: the custom Makefile (init is not a stack-file updater; compare it with the
   current scaffold and apply relevant gate fixes explicitly), and the house rules, which must
   live outside the regenerated section — above the marker heading, where they survive refresh.
   Commit the protocol refresh through normal review.

9. **The identifier lives in the check registry** (`kit help checks` is generated from it);
   add checks there, never as bare strings. The test obligation: a registered check with no
   conformance case fails the build — `TestEveryRegisteredCheckIsExercised` compares cases
   against `kit.Checks` both ways, and every audit case starts from the same clean project
   that must report nothing. **No**, a prose rule cannot enforce human-only merge: prose is
   the lowest tier, get past it by reading carelessly; who may merge is enforced by remote
   permissions and required checks.

10. **Set `cmd.WaitDelay`** on every git command: `exec.CommandContext` kills git and nothing
    git started, so a subprocess helper holding the stdout pipe makes `cmd.Output()` block for
    ever — the delay comes out of the bound, not on top of it. **No**, the race tests must not
    inherit the stripped build's `CGO_ENABLED=0`: the race detector needs cgo on linux/amd64,
    so a file-level export passes locally (darwin/arm64) and fails CI.
