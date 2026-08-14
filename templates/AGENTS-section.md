## Work protocol

Run `kit next` at the start of every session. It names the feature and the task.
Run `kit audit` when you want to know what has drifted. `kit help` explains the
whole model, and `kit help template` prints the feature doc format.

### Picking and finishing work

1. **Asked to plan something? Read `docs/VISION.md`, then draft.** It says what the
   project is for and what it is not; a feature that cannot be traced to it is the
   wrong feature, and saying so is cheaper than building it. Size the doc to one
   agent's reach — if a box cannot start until someone else's work lands, it belongs
   in a different feature, and `needs:` is where you say so. Fill it in for every
   draft: it is what lets `kit next` refuse work whose foundation is missing, and
   several features with an empty `needs:` is what lets agents run at once. Judge it
   by whether the work could *start* now — a shared composition root, a shared
   router, or numbered files in one sequence such as migrations all make a feature
   dependent however separate it looks. Leave the doc `Draft`: accepting is a
   human's call.
2. **One agent per feature. Take it with `kit claim <id>`.** That pushes the branch
   named after the doc, and creating that ref on the remote is what decides who
   gets the work: exit 0 is yours, exit 1 means another agent got there first — run
   `kit next` and take something else. Losing is normal and costs nothing. Never
   create the claim branch by hand: a branch only you can see is not a claim, and a
   push that reports "up to date" found somebody else's.
3. **One feature per checkout. Work in the worktree `kit claim` prints.** The
   worktree is the isolation and git provides it; the kit only refuses to hand out
   work in a checkout that is occupied. Run in one, `kit next` reports that feature
   as *yours* and will not offer you another to start — starting a second one where
   you stand puts two features in one branch and they ship as one.
4. **Read the feature doc for the design.** Never restate or re-derive design
   anywhere else.
5. **Do the topmost unticked `## Tasks` box, and only that one.** List order is the
   order of work. Tick it in the same PR that implements it. Among features of
   equal status, `kit next` goes by ID, lowest first.
6. **Discovered work** becomes a new `## Tasks` box, a new feature doc
   (`kit new "<title>"`), or a line in `docs/features/BACKLOG.md` if it is real but
   not now. Never a `TODO` in the code. Append it unless it genuinely blocks the
   boxes below it — list order is priority, so appending is what says "later".
7. **A box that turns out wrong may be rewritten, split, or deleted** — in its own
   commit, with the reason in `## Log`. The plan is a working document; only
   `status: Shipped` freezes it. What you may not do is silently drop a box you
   could not finish.
8. **Stopping mid-feature? Release the claim.** Delete the branch locally *and* on
   the remote (`git push origin --delete <branch>`), set the status back to
   `Accepted`, and say why in `## Log`. The pushed branch is the claim, so deleting
   only your local copy leaves the feature taken for good and `kit next` will route
   every future agent around it.
9. **Blocked on a decision only a human can make?** Run
   `kit block <id> "<question, with your leaning>"`, then move to another feature.
   Do not guess, and do not idle. Use `--fyi` for anything work can continue
   without.
10. **Never put a real secret in a doc.** Feature docs, `## Log`, and `docs/INBOX.md`
   are committed, and a pushed commit cannot be unpublished. Name the variable and
   where the value lives — never the value. `audit` reports `secret-in-doc` at HIGH
   for the shapes it can be certain about, which is a backstop and not the rule: a
   password in prose matches nothing and is still a leak.
11. **Never edit a doc with `status: Shipped`.** It is frozen history.
12. **When the last box is ticked**, set `status: Shipped` and the `shipped:` date,
    and move durable facts into the as-built doc (`docs/TECH.md`).
13. **Shipping is a handoff, not a finish.** Push the branch and stop there.
    Reviewing and merging are not yours. If the project has no remote, the work
    has nowhere to go — say so and leave it on the branch rather than merging it
    yourself, which is how unreviewed work reaches `main` without anyone deciding
    it should.

### Where two agents can still collide

The claim is airtight and the ID is not. `kit new` takes the next free ID from what
it can see on disk and never goes to the network, so two agents drafting at once on
two branches can both produce `0009`. Nothing in git objects: the filenames differ,
so the merge is clean.

`kit audit` reports it as `duplicate-id`, and it is HIGH because the ID is the
primary key — one branch would claim both features and one INBOX line would block
both. `kit claim` refuses an ID that two docs share for the same reason.

Fix it before claiming either: give one doc a free ID, rename its file to match, and
rename its branch if it already has one. Run `kit audit` after any merge that brought
in feature docs written elsewhere.

### Doing the work

The rules above decide *what* to work on. These govern *how*, and bias toward caution
over speed on anything non-trivial. Use judgement on trivial tasks.

- **Think before coding.** State assumptions out loud. Where a request is ambiguous,
  name the readings rather than silently picking one. Push back when a simpler
  approach exists, and stop when confused — say what is unclear instead of guessing
  past it. Rule 8 is where a question goes when only a human can settle it.
- **Simplicity first.** The minimum code that solves the problem. Nothing
  speculative, no features beyond the box you are on, no abstraction for a single
  use. If a senior engineer would call it overcomplicated, it is.
- **Surgical changes.** Touch only what the task requires. Do not improve adjacent
  code, comments, or formatting, and do not refactor what is not broken — match the
  style already there. Clean up your own mess and nobody else's. Unrelated problems
  you spot are rule 4's business, not this PR's.
- **Define success, then loop.** Say what "done" looks like before starting, and
  verify it yourself rather than reporting completion and hoping. A box you cannot
  check the truth of was too big; split it.
