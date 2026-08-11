## Work protocol

Run `kit next` at the start of every session. It names the feature and the task.
Run `kit audit` when you want to know what has drifted. `kit help` explains the
whole model, and `kit help template` prints the feature doc format.

### Picking and finishing work

1. **One agent per feature.** Claim it by working on a branch named after its doc
   (`0009-ton-payment-rail`). If `kit next` lists a feature as taken, someone else
   is on it — pick another.
2. **Read the feature doc for the design.** Never restate or re-derive design
   anywhere else.
3. **Do the topmost unticked `## Tasks` box, and only that one.** List order is the
   order of work. Tick it in the same PR that implements it.
4. **Discovered work** becomes a new `## Tasks` box, a new feature doc
   (`kit new "<title>"`), or a line in `docs/features/BACKLOG.md` if it is real but
   not now. Never a `TODO` in the code.
5. **Blocked on a decision only a human can make?** Append it to `docs/INBOX.md`
   with your leaning, then move to another feature. Do not guess, and do not idle.
6. **Never put a real secret in a doc.** Feature docs, `## Log`, and `docs/INBOX.md`
   are committed. Name the variable and where the value lives — never the value.
7. **Never edit a doc with `status: Shipped`.** It is frozen history.
8. **When the last box is ticked**, set `status: Shipped` and the `shipped:` date,
   and move durable facts into the as-built doc.

### Doing the work

The rules above decide *what* to work on. These govern *how*, and bias toward caution
over speed on anything non-trivial. Use judgement on trivial tasks.

- **Think before coding.** State assumptions out loud. Where a request is ambiguous,
  name the readings rather than silently picking one. Push back when a simpler
  approach exists, and stop when confused — say what is unclear instead of guessing
  past it. Rule 5 is where a question goes when only a human can settle it.
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
