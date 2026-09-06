## Work protocol

Act on the human's requested scope. Agents run kit and manage the workflow; humans
approve plans and merge work. Run `kit next` at session start and after finishing or
releasing work. `kit audit` reports drift; `kit help <command>` explains refusals.
Fix the cause; do not bypass it with raw git or hand edits.

### Planning

Plan when asked. Unplanned exploration uses the spike route below.

1. Read `docs/VISION.md`. A feature is a verifiable improvement: capability, removal,
   security, reliability, performance, cost or maintainability. Name who benefits and
   what observable result proves it.
2. Prefer one coherent, safely reviewable PR. Implementation layers, migration, tests
   and docs are task boxes. Split for independent value, separate approval or review
   size, not agent count. Small changes without design decisions may be chores.
   Removing integrated work needs a new feature citing its frozen doc.
3. With no commits, commit setup on trunk first. Draft the round with `kit new "<title>"`
   on one `plan/<round>` branch with no four-digit id. Keep status Draft; `needs:` names
   prerequisite features and stays empty for independent work.
4. Push the round and open one PR. Human approval through merge is required before
   implementation. After merge, update a trunk checkout with `git pull --ff-only`,
   then run `kit audit` and `kit next`.
5. Put uncertainty in Open questions. Use `kit block <id> "<question>"` for human
   decisions; never invent the answer. Record the answer in the plan, remove its inbox
   line and commit both. Integrate the answer into paused checkouts before resuming.

### Taking work

6. From updated trunk, run `kit claim <id>`. Exit 0: yours; use the named worktree.
   Exit 1: not yours; run `kit next` and choose other work. Exit 2: fix the reported
   cause before retrying. Never clean up another agent's claim or worktree.
7. Each agent has its own checkout, index and claim. A branch citing a four-digit id
   is a claim; chores and spikes cite none.
8. Human-directed exploration uses its own spike branch. Never use a spike to bypass
   a pending plan's approval. Finish with `kit harvest "<title>" --box "<outcome>"`;
   it names the branch after the feature. Fill Problem and Design before submitting.
   With no commits, harvest creates an empty trunk base; publish it before the first PR.
   Use `kit harvest --nothing "<lesson>"` for discarded exploration.

### Doing the work

9. Read the feature design, `docs/TECH.md` and the applicable `kit help go-checklist`
   or `kit help python-checklist` before coding.
10. Implement the first unticked Tasks box and tick it in that commit. One box, one
    commit; the subject names the outcome. Continue until ready for review or blocked.
    Explain only what the diff and plan cannot.
11. State assumptions and observable completion criteria; verify them. Match project
    style. Change only what is required; avoid speculative abstractions and cleanup.
12. Record discoveries as task boxes, features or BACKLOG entries. Append boxes unless
    they block earlier work. Revise incorrect tasks in a separate commit with the
    reason in Log; never silently delete unfinished work.
13. `kit drop <id> "<why>"` retires unbuilt plans. `kit release <id>` surrenders untouched
    work. To pause changed work, commit and push, open an unfinished PR and state what remains.
14. Never put secrets in docs; name the variable and storage location. Only `kit submit`,
    `kit drop` and `kit reopen` write status and submitted fields.

### Finishing and review

15. Move durable facts to TECH; Log holds only plan changes and may be empty. Put
    validation and review context in the PR.
16. Run the project gate, `kit audit --strict`, then `kit submit <id>`, in that order.
    Failed or incomplete checks are not passes. Offline audit covers local checks;
    finish online audit before handoff.
17. Push, open the PR and hand it to the human. Submit means ready for review;
    integration requires merge. Never merge your own work. With no remote, state that
    limitation and leave the branch for local review.
18. For review changes, `kit reopen <id> "<why>"`, revise tasks, implement, then repeat
    gate, audit and submit. Submitted docs on trunk are frozen; further work needs a new doc.
19. After merge, update trunk with `git pull --ff-only`, then run `kit audit` and `kit next`.
    Dependencies unlock only when merged work is here. Resolve duplicate offline ids
    before claiming either document.

### Writing

- Keep facts, decisions, constraints and measurements concise; omit git-recoverable narration.
- One home per fact: TECH for current contracts, feature for plan, Log for plan changes,
  PR for validation and handoff. Cite feature ids instead of repeating their reasoning.
- Comments explain constraints code cannot express. Prefer tool enforcement to prose;
  remote permissions and required checks enforce human merge authority.

Formats: `kit help template`, `kit help backlog`, `kit help checks`.
