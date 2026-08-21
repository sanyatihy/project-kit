# Backlog

Ideas that are not features yet. Nothing here is scheduled — the default state is **stays
frozen**. Promote one with `kit new` only on a real signal: someone asked, a metric moved,
something else needs it.

Keep each entry to a few lines: what it is, and what would make it worth doing.

## Reserve feature IDs over the network

`NextID` reads the highest ID on disk and never asks the remote, so two agents drafting off
the same trunk both allocate the same number. The filenames differ, so both pull requests
merge clean and `duplicate-id` fires at HIGH afterwards — a human then renames a doc, its
file, and possibly its branch.

The mechanism exists: push `refs/kit/ids/<id>` as a reservation and read the winner from the
ref update, the compare-and-swap `kit claim` already uses. What stops it being obvious is
that `kit new` touches no network at all, so this decides whether drafting offline stops
working or silently loses the guarantee.

Promote when: a duplicate ID actually costs a merge, or a second agent starts drafting in
parallel for real.

## `needs:` cannot tell "forgotten" from "nothing"

It is the only field that changes what `kit next` offers, and an empty one means "can start
now" — the permissive answer. An agent that never considered the question produces the
identical bytes to one that considered it and decided nothing was required, and the cost of
being wrong is an agent building on a foundation that is not there.

The honest fix is an explicit token: `needs: none`, with an unanswered field reported. That
puts every shipped doc in breach the day it lands, so it needs a migration pass or a rule
that only applies to docs created after it.

Promote when: a feature is claimed whose dependency was real and undeclared.

## The work is only on the remote once, at the end

`kit claim` pushes `HEAD`, so the branch that claims a feature carries the doc and no work,
and the kit's only other push is `release` deleting that same ref. Everything after that is
local until somebody pushes for review. Liveness is read from `git log --all`, which is this
machine's refs.

So the mutual visibility the multi-agent model rests on is the shared local object store,
not the remote. On one machine with worktrees that holds. Across two it does not, and it
fails silently: `stalled` never ages a claim carrying no commits beyond the doc, which is
exactly what a live agent looks like from anywhere else. Progress reads 0/N the same way.

The fix is that the push becomes the heartbeat — the claim branch gets one when a box is
ticked — after which `stalled` can age from the remote branch's commit date and be true from
any machine. It has to degrade for `--offline` and for projects with no remote.

Promote when: a second machine joins, or a claim's work is lost with the machine holding it.

## A merged claim holds its slot until the trunk ref catches up

`AwaitingReview` skips a branch the trunk contains, comparing against `trunkRef`, which
prefers the local `refs/heads/<trunk>`. A checkout that has not pulled since the merge keeps
listing the feature as awaiting review — and the protocol tells every other agent not to
touch it. Deleting the remote branch clears it, which is why the symptom reads as a missing
check rather than a stale ref.

`kit release` is the delete half and it is built: on a merged branch its work guard passes,
because the branch carries nothing the trunk lacks. What is left is that nothing says to run
it, and most forges do not delete a source branch on merge.

Promote when: a merged feature is seen holding a slot, or a second machine reads a board this
one has already cleaned.

## A claim can carry a doc without carrying the latest edits to it

`TrackedAtHEAD` asks whether the feature doc exists in `HEAD`'s tree, which stops a claim
pushing a branch with no plan on it. It does not ask whether the working tree's version
matches, so claiming with uncommitted edits to the doc pushes the committed one — the plan
travels, the last changes to it do not.

Milder than the failure it replaced, and `git diff --quiet HEAD -- <doc>` would close it.
What needs thinking about first is whether that would refuse claims people legitimately make
mid-edit.

Promote when: a worktree is found holding an older doc than the one that was claimed.

## `kit defer <id>` moves an unticked box to the backlog

`unticked-shipped` reports a Shipped doc with open boxes and asks for a hand edit of a frozen
doc plus a hand-written BACKLOG entry — two chances to get it wrong, and the second one
silently skipped. A command that takes the box out, appends it here with its feature id, and
leaves the doc consistent is the prevent-tier version.

Promote when: it happens a third time.

## Steps the operator still runs by hand

- `kit claim` prints a `git worktree add` line to copy. `--worktree` could create it and
  print where to work.
- `make install` after every merge. Nothing checks it, and the kit records what it cost
  once: a project built against a binary two features behind, with nothing reporting
  anything wrong.

Promote when: one of them is forgotten and costs something.

## Frontmatter values are truncated at ` #`

`trailingRe` strips ` # comment` from every frontmatter value so template scaffolding never
reads as data. It cannot tell a comment from content, so `title: Add # tags to links` becomes
`Add`. Nothing has hit it, and the fix — require two spaces, or only strip from keys known to
carry comments — is small.

Promote when: a title or any other field loses text to it.

## One agent that plans and builds in the same session has no path

The protocol has two: a drafting round lands on the trunk, then a builder claims from it.
An agent asked to do both in one session can do neither cleanly — `claim` refuses until the
round lands, and `ship` needs a claim, so the work ends as a plan branch carrying docs whose
boxes are ticked and whose status is still `Draft`. `done-not-shipped` reports it, correctly,
and there is nothing the agent may do about it.

Either the protocol says this is the coordinator's case and names what to do, or `ship`
learns a path that does not require a claim branch. Both are more than one session's
awkwardness justifies.

Promote when: it happens again, or a second operator hits it.

## Closed by the stabilisation round

Seven entries left here when 0047-0049 landed. `kit accept` could refuse an unwritten doc,
and the three checks that disagreed about a frozen doc, went with the command and the checks
themselves. `kit help deployment` and its missing mechanics went with the topic. The two
proposals for checks over prose — a doc that argues more than it decides, and a fact written
twice — were dropped with the writing checks: `audit` reports what is checkable, and no
threshold over prose was ever measured against a corpus written under the rule it enforces.
The fleet-runtime notes documented one operator's tooling rather than the kit's model.
