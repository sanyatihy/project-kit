# VISION

What this project is for. Feature docs in [features/](features/) say what is being
built now; this says what it is being built *towards*, and it is what an agent reads
before drafting one.

<!-- This file has no lifecycle and no status. It is expected to change while the
     project is built. Keep it short: everything here is read before every draft. -->

## What this is

A single Go binary and a documented protocol that let coding agents run a project
from markdown in git. The feature docs and the git branches are the whole state —
there is no database, no queue, and nothing to keep in sync. `kit` answers three
questions in every repo that adopts it: what should I work on, what has drifted, and
is it green.

## Who it is for

An operator running several coding agents at once, and the agents themselves. The
operator's scarce resource is attention: they decide what gets built and whether it
is good, and everything else should happen without them. The agents' scarce resource
is certainty about what is true — which feature is taken, what the plan is, whether a
doc still describes reality.

## What done looks like

An operator gives a prompt; an agent drafts a feature; the operator accepts it; agents
build it unattended and stop only when a human must decide something. Concretely: a
new project is set up with one command, an agent's first five minutes are identical in
every repo, and `kit audit` disagreeing with the repo is always a real problem rather
than noise to be ignored.

## What it is not

Not an issue tracker, and not a planning surface with a roadmap, milestones or dates.
Not a build system, a CI system, or deployment scaffolding — those are named in
`kit help deployment` as knowledge and deliberately not generated. Not a judge of
whether work is any good: `audit` reports what is checkable and never guesses, because
being believed is the only thing it has. And not a service — nothing here may require
a daemon, a server, or an account.

## What it is measured by

Speed and transparency **together** — each one alone is a failure mode, and the kit exists
at the balance between them. Grinding code with no record is fast until the week after,
when nobody can say what was done or why and the only source left is the diff. Writing
documents as a phase of their own is slow while it happens and produces files nobody
finishes reading. Neither is the goal.

So the record has to be a **byproduct of the work rather than a stage before it**: written
while doing, not in advance of doing, and only the parts that a person or an agent coming
back later cannot recover from the code. A ticked box in the commit that implemented it
costs nothing and says what happened; a design essay written before the work is expensive
and frequently wrong by the end.

Every field, rule, command and document is weighed both ways: what it costs the person
doing the work now, and what it saves the one who has to understand it later. Something
that fails both does not belong in the kit, whatever else it improves.

## Constraints

One binary, no runtime dependencies, and state that is only markdown and git. An
agent must never need this repository to use the tool, so anything an agent has to
know is embedded in `kit help`. A check that fires on healthy projects is worse than
no check. And the kit stays small: every addition is weighed against the thing that
makes it good, which is that a person can read all of it.
