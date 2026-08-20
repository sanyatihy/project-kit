# Backlog

Ideas that are not features yet. Nothing here is scheduled or prioritised — the
default state is **stays frozen**.

Brainstorm output lands here. Promote one to its own `NNNN-*.md` with `kit new`
only on a real signal: someone asked, a metric moved, something else needs it.

Keep each entry to a few lines: what it is, and what would make it worth doing.

The findings from reviewing the kit against `kit help go-checklist` were promoted to
[0004](0004-close-the-go-checklist-findings.md) and closed — three fixed, one dropped
once its premise turned out to be false.

## What running several agents actually takes

Standing up a fleet turned up four properties of the substrate, none guessable and
each costing a debugging session: a pane title is not a handle, because the agent
overwrites it; killing a pane leaves the agent running detached; working directory is
the join key for both session storage and editor binding, which is why a worktree
starts with no agent memory; and one installed binary serves every worktree, which
bites hardest in a repo that builds the tool it is driven by.

Worth doing as a `kit help agents` topic if a second operator needs it, or if the
launcher these describe turns out to be the same script in every project. Until then
it documents one person's tooling rather than the kit's model, and the kit is better
kept light.

## `kit help deployment` stops where the traps start

The topic is 94 lines of principle. Reviewing it against a working pipeline turned up
the mechanics it omits, and they are the parts that cost an outage:

- Path-filtered jobs make *skipped* indistinguishable from *green* to `needs:`, so
  path filtering plus separate gates silently lets a publish run on a red gate. The
  fix is one aggregate gate job that treats skipped as passing and failure as fatal.
- A job whose condition uses a status function loses the implicit success check for
  everything downstream, so a deploy can be skipped after a build that succeeded.
- Gate the deploy on the artifact existing, not on the trigger that produced it.
- Plan should be a real dependency of apply, not a preview a human is trusted to read.
- The handoff between provision, configure, and deploy is remote state, not copy-paste.
- Secrets belong in a generated vars file, never on a command line.
- Validation must run without a credential, and at full strictness or not at all.
- Assert in CI on any third-party signal an alert depends on.

Worth doing when a project using the kit is actually going to production and this
would be read rather than written. It is guidance, not machinery: `audit` cannot
check any of it, which is why it stays in the binary and not in a scaffolded file.

## Reserve feature IDs over the network

`NextID` reads the highest ID on disk and never asks the remote, so two agents drafting
off the same `main` both allocate the same number. The filenames differ, so both pull
requests merge clean and `duplicate-id` fires at HIGH afterwards — a human then renames a
doc, its file, and possibly its branch. Under pull-request drafting this is the common
case rather than the edge one, because every draft is cut from the same trunk.

The mechanism already exists: push `refs/kit/ids/<id>` as a reservation and read the
winner from the ref update, exactly the compare-and-swap `kit claim` uses. What stops it
being obvious is that `kit new` currently touches no network at all, so this decides that
drafting offline either stops working or silently loses the guarantee.

Promote when: a duplicate ID actually costs a merge, or a second agent starts drafting
in parallel for real.

## `needs:` cannot tell "forgotten" from "nothing"

It is the only field that changes what `kit next` offers, and an empty one means "can
start now" — the permissive answer. An agent that never considered the question produces
the identical bytes to one that considered it and decided nothing was required, and the
cost of being wrong is an agent building on a foundation that is not there.

The honest fix is an explicit token: `needs: none`, with an unanswered field reported.
That puts all forty-four shipped docs in breach the day it lands, so it needs a migration
pass or a rule that only applies to docs created after it. Both are more than the
problem currently justifies, and 0016 deliberately left it here rather than shipping a
cosmetic version.

Promote when: a feature is claimed whose dependency was real and undeclared.

## `kit accept` could refuse a doc nobody wrote

It refuses a non-Draft, open questions, and a BLOCKING inbox line — every way the *doc*
says the design is not settled. It does not refuse a doc where `## Problem`, `## Design`
and `## Tasks` are all empty, which is the strongest such signal there is. Replaying the
lifecycle found it: `kit accept` succeeded on a doc straight out of `kit new`.

The consequences are covered everywhere else: `unfilled-template` reports it at HIGH,
`--strict` fails on it, and `next` and `claim` both refuse it through `Feature.Unwritten()`
— including a claim made by ID without consulting the board. So what is left is `accept`
blessing a doc nobody wrote, which misleads a reader rather than sending an agent to build
nothing. Reusing the same emptiness test in `Accept` would close it in about three lines.

Promote when: touching `accept` for any other reason, or if an agent is seen claiming an
unwritten feature.

## A claim can carry a doc without carrying the latest edits to it

`TrackedAtHEAD` asks whether the feature doc exists in `HEAD`'s tree, which is what stops
a claim pushing a branch with no plan on it at all. It does not ask whether the working
tree's version matches, so claiming with uncommitted edits to the doc pushes the committed
one — the plan travels, the last changes to it do not.

Much milder than the failure it replaced (the agent gets a real doc, just a stale one) and
`git diff --quiet HEAD -- <doc>` would close it. What needs thinking about first is whether
that would refuse claims people legitimately make mid-edit.

Promote when: a worktree is found holding an older doc than the one that was claimed.

## Steps the operator still runs by hand

Two, each one a command the kit prints or a rule the protocol states, and each a place
work stops while a person types:

- `kit claim` prints a `git worktree add` line to copy. `--worktree` could create it and
  print where to work.
- `make install` after every merge. Nothing checks it, and the kit already records what
  it cost once: a project built against a binary two features behind, with nothing
  reporting anything wrong.

Promote when: one of them is forgotten and costs something, or when several agents run
often enough that the typing is the slow part.

## Frontmatter values are truncated at ` #`

`trailingRe` strips ` # comment` from every frontmatter value so template scaffolding
never reads as data. It cannot tell a comment from content, so
`title: Add # tags to links` becomes `Add`. Nothing has hit it, and the fix — require
two spaces, or only strip from keys known to carry comments — is small.

Promote when: a title or any other field loses text to it.

## The work is only on the remote once, at the end

`kit claim` pushes `HEAD`, so the branch that claims a feature carries the doc and no work,
and the kit's only other push is `release` deleting that same ref. Everything after that is
local until a person or an agent pushes for review, which `kit ship` asks for in prose — the
tier TECH.md ranks last. Liveness is read from `git log --all`, which is this machine's refs.

So the mutual visibility the multi-agent model rests on is the shared local object store,
not the remote. On one machine with worktrees that holds. Across two it does not, and it
fails silently: `stalled` never ages a claim carrying no commits beyond the doc, which is
exactly what a live agent looks like from anywhere else, so a dead agent and a working one
stay indistinguishable for the whole life of the feature. Progress reads 0/N the same way.
`shipped-not-pushed` is this failure caught at the end; nothing watches the middle.

The fix is that the push becomes the heartbeat — the claim branch gets one when a box is
ticked — after which `stalled` can age from the remote branch's commit date and be true from
any machine. Enforcement belongs in a command rather than another protocol sentence, and it
has to degrade the way 0030 does for `--offline` and for projects with no remote.

Promote when: a second machine joins, or a claim's work is lost with the machine holding it.

## A doc that argues more than it decides

`doc-bloat` measures a whole file against one number, so it cannot see composition: it would
flag a doc that is all durable design and stay quiet on one that is thirty lines of argument
and eight of decision. The second is what this repo actually writes.

Composition is measurable, and one comparison carries most of it — `## Problem` at least as
long as `## Design` means the doc argued more than it decided. Across 54 real docs it
separates the two corpora exactly: 10 of 40 here, 0 of 14 in `localcode`. Per-section
ceilings are the fuller version of the same idea.

0041 fixes what gets written; this checks the shape of what came out. It wants its own
calibration, taken after 0041 changed the corpus: five docs have been written since, which
is a sample rather than a corpus.

Promote when: 0041 has shipped and a round of docs has been written under it.

## Three checks disagree about what a frozen doc is

Rule 11 says a Shipped doc is frozen history, and twice now a human has authorised a
compaction pass over the whole corpus anyway — 0037, and the pass that followed 0044. The
checks were each built against one reading of that and now contradict each other and the
protocol:

- `session-narration` skips Shipped because rule 11 forbade the fix; `doc-bloat` does not.
  So on a real board the length alarm fires only where the fix is forbidden, and the
  narration check goes quiet exactly there. Measured on `localcode`: both its `doc-bloat`
  findings are on Shipped docs, and all 7 narration hits are too.
- `missing-section` requires a non-empty `## Log` on a Shipped doc, which the routing rule
  contradicts — an entry naming no change is not an entry, so a feature that changed no
  plan has an honest empty Log. It is what produced a status-echo entry in `ai-bot`'s 0004.
- `unticked-shipped` tells you to tick the boxes or move them to BACKLOG, both of which
  edit a frozen doc.

A fourth disagreement was settled the other way and narrows this one. `unchecked` asked for
two writes into a Shipped doc, and 0046 gave them to `kit check` rather than loosening rule
11. That is the answer wherever the fix is a *write* a command can own, which covers
`unticked-shipped` too — the entry below. It is no answer for `doc-bloat` and
`session-narration`, whose fix is rewritten prose, so those two are what is actually being
asked about here.

One change, because it is one question: rule 11 says what it already means — frozen to an
agent working a feature, editable in a pass a human asks for — the two LOW checks then fire
on every status and become the worklist for that pass, and `missing-section` goes. The cost
to weigh first: a doc that earns its length reports LOW for ever, since nothing detects the
"say in ## Log why" the fix line offers.

Promote when: the next compaction pass is wanted, or a board carries a finding nobody is
allowed to fix.

## A fact written twice

The routing rule gives every fact one home, and nothing checks it. Token overlap between
each feature doc and the as-built doc finds real duplicates cheaply — 9 in `localcode`, 4
here, at a Jaccard of 0.45 over content words.

What stops it being a check today is that some overlap is correct: a `## Design` sentence
promoted to TECH at ship is the rule working, not breaking. It needs a threshold measured
against a corpus written under the routing rule, and probably a rule about which section
the duplicate sits in.

Two docs exist under the routing rule so far, which is thin for a threshold.

Promote when: a duplicated fact goes stale in one of its two homes, or enough docs are
written under the rule to measure against.

## `kit defer <id>` moves an unticked box to the backlog

`unticked-shipped` reports a Shipped doc with open boxes and asks for a hand edit of a
frozen doc plus a hand-written BACKLOG entry — two chances to get it wrong, and the second
one silently skipped. A command that takes the box out, appends it here with its feature id,
and leaves the doc consistent is the prevent-tier version.

0046 built this shape for `unchecked` — a command owning the two writes instead of an
exception in prose — so it is a pattern to copy rather than a proposal to weigh.

Promote when: it happens a third time, or the frozen-doc entry above is built and this is
the cheap half of it.

## The routing rule is the one convention with no check behind it

Every other rule an agent obeys unprompted fails a command when it is broken — gofmt, a
task over budget, a claim on a plan that has not landed. "One `## Log` entry, naming the
change" is prose only, and it is the one being broken: three breaches in two days by one
agent, two of them after correction.

Two properties make it harder than ordinary prose. It inverted inside a day — 0041's
template called a measurement a valid entry, 0044 says an entry naming no change is not one
— so an agent carrying the older phrasing writes entries that look right and are not. And it
lives in the template and `AGENTS-section.md`, neither of which is opened to append to a doc
that already exists.

Length is not the instrument and that is settled: 0038 capped an entry, 0041 removed the arm
after it fired on 13 of 56, the ones carrying the most. What is left is `NarrationMarkers`'
shape — markers for the verbs an entry naming a change actually uses, at LOW, certain when
it fires and blind to prose. Cheaper still, and how every convention here that stuck was
taught: one rejected entry beside its rewrite, in the template.

Renaming the section to `## Decisions` would outweigh any amount of prose, because
everywhere else a log is a chronological record and the name argues against the guidance. It
costs a migration across every project the kit scaffolded, most of those docs frozen — the
same question the frozen-doc entry above asks, and it should be weighed there.

Promote when: a breach reaches the default branch, or the frozen-doc entry is built and the
rename can ride with it.

## A merged claim holds its slot until the trunk ref catches up

`AwaitingReview` already skips a branch the trunk contains, with the `merge-base
--is-ancestor` test a `merged-claim` check would use. It compares against `trunkRef`, which
prefers local `refs/heads/<trunk>`, so a checkout that has not pulled since the merge keeps
listing the feature as awaiting review — and rule 2 tells every other agent not to touch it.
Deleting the remote branch clears it, which is why the symptom reads as a missing check
rather than a stale ref.

`kit release` is the delete half and it is built: on a merged branch its work guard passes
without `--force`, because the branch carries nothing the trunk lacks. What is left is that
nothing says to run it. After a merge the board either lists the feature as awaiting review,
where the trunk ref is stale, or drops it silently while the branch stays on the remote —
and most forges do not delete a source branch on merge.

The two fixes differ in what they buy: preferring the remote-tracking trunk is a line, and
is silently wrong when that ref is stale too; a finding says the claim is over and names the
command that ends it.

Promote when: a merged feature is seen holding a slot on a board, or a second machine reads
a board this one has already cleaned.
