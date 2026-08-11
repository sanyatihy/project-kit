## Work protocol

Run `kit next` at the start of every session. It names the feature and the task.
Run `kit audit` when you want to know what has drifted. `kit help` explains the
whole model, and `kit help template` prints the feature doc format.

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
