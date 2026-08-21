# Backlog

Ideas that are not features yet. Nothing here is scheduled — the default state is **stays
frozen**. Promote one with `kit new` only when its trigger has actually fired.

An entry earns its place by naming a value somebody would notice and a trigger that can be
observed. An entry whose trigger has never fired, or whose fix costs more than the problem,
is deleted rather than kept as an option.

## The work is only on the remote once, at the end

`kit claim` pushes `HEAD`, so the branch that claims a feature carries the doc and no work,
and the kit's only other push is `release` deleting that same ref. Everything after that is
local until somebody pushes for review. Liveness is read from `git log --all`, which is this
machine's refs.

So the mutual visibility the multi-agent model rests on is the shared local object store,
not the remote. On one machine with worktrees that holds. Across two it does not, and it
fails silently: `stalled` never ages a claim carrying no commits beyond the doc, which is
exactly what a live agent looks like from anywhere else. Two checkouts of one feature have
been observed reporting different progress for it in the same instant, because each read
whichever copy of the branch it happened to have.

The fix is that the push becomes the heartbeat — the claim branch gets one when a box is
ticked — after which `stalled` can age from the remote branch's commit date and be true from
any machine. It has to degrade for `--offline` and for projects with no remote.

**Value:** a board that means the same thing on every machine.
**Promote when:** a second machine joins, or a claim's work is lost with the machine holding it.

## Reserve feature IDs over the network

`NextID` reads the highest ID on disk and never asks the remote, so two agents drafting off
the same trunk both allocate the same number. Reproduced under concurrency: six `kit new`
runs at once produced four docs claiming `0001`. `kit claim 0001` is then exit 2 until a
human renumbers.

The mechanism exists: push `refs/kit/ids/<id>` as a reservation and read the winner from the
ref update, the compare-and-swap `kit claim` already uses. What stops it being obvious is
that `kit new` touches no network at all, so this decides whether drafting offline stops
working or silently loses the guarantee.

**Value:** several agents can draft a round at once without a human renumbering afterwards.
**Promote when:** a duplicate ID costs a real merge, or a second agent starts drafting in
parallel for real.

## A merged claim holds its slot until the trunk ref catches up

`AwaitingReview` skips a branch the trunk contains, comparing against `trunkRef`, which
prefers the local `refs/heads/<trunk>`. A checkout that has not pulled since the merge keeps
listing the feature as awaiting review — and the protocol tells every other agent not to
touch it. Most forges do not delete a source branch on merge, and nothing says to run
`kit release` afterwards, whose work guard passes on a merged branch.

**Value:** a finished feature stops occupying a slot on everyone's board.
**Promote when:** a merged feature is seen holding a slot, or a second machine reads a board
this one has already cleaned.

## One agent that plans and builds in the same session has no path

The protocol has two: a drafting round lands on the trunk, then a builder claims from it. An
agent asked to do both in one session can do neither cleanly — `claim` refuses until the
round lands, and `ship` needs a claim, so the work ends as a plan branch carrying docs whose
boxes are ticked and whose status is still `Draft`. It has happened twice.

Either the protocol names this as the coordinator's case and says what to do, or `ship`
learns a path that does not require a claim branch.

**Value:** the common single-operator session stops ending in a state the tool reports as wrong.
**Promote when:** it happens a third time, or a second operator hits it.

## Cut from this backlog

Five entries were deleted rather than carried. `needs: none` would have put every shipped
doc in breach to catch a dependency nobody has yet failed to declare. `kit defer` proposed a
command for a case seen twice. "Steps the operator still runs by hand" was a list of two
conveniences, not a proposal. "A claim can carry a doc without its latest edits" described a
failure milder than the one it replaced and has never been observed. Frontmatter truncation
at ` #` stopped being a proposal when it was reproduced — it is a task box in 0051.
