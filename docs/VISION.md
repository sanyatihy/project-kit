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

Operators give intent in conversation: draft, park, finish or revise. Agents run kit,
manage branches and prepare review. Humans approve plans and merge work; next and audit
are also available for direct inspection by the operator.

This includes smaller local coding models such as Qwen. Changes must keep commands,
conventions and next actions explicit and easy to follow without reconstructing the
workflow from scattered rules or relying on frontier-model reasoning.

## What done looks like

An operator gives a prompt; an agent drafts a feature; the operator accepts it; agents
build it unattended and stop only when a human must decide something. Concretely: a
new project is set up with one command, an agent's first five minutes are identical in
every repo, and `kit audit` disagreeing with the repo is always a real problem rather
than noise to be ignored.

## What it is not

Not an issue tracker, and not a planning surface with a roadmap, milestones or dates.
Not a build system, a CI system, or deployment scaffolding — the last is the
fastest-rotting part and is deliberately not generated. Not a judge of
whether work is any good: `audit` reports what is checkable and never guesses, because
being believed is the only thing it has. And not a service — nothing here may require
a daemon, a server, or an account.

## What it is measured by

Speed and transparency **together** — each one alone is a failure mode, and the kit exists
at the balance between them. Grinding code with no record is fast until the week after,
when nobody can say what was done or why and the only source left is the diff. Writing
documents as a phase of their own is slow while it happens and produces files nobody
finishes reading. Neither is the goal.

For planned work, record the decisions and observable outcomes the human must approve
before implementation. During implementation, maintain only what a returning reader
cannot recover from the code. A ticked box in its implementing commit records the result;
Log records changes to the approved plan. Human-directed exploration records its decisions
retrospectively through harvest, then receives implementation review.

Design prose stays short: facts, decisions and constraints needed for review.

Every field, rule, command and document is weighed both ways: what it costs the person
doing the work now, and what it saves the one who has to understand it later. Something
that fails both does not belong in the kit, whatever else it improves.

## Constraints

One binary, `git` as the only runtime dependency, and state that is only markdown and git. An
agent must never need this repository to use the tool, so anything an agent has to
know is embedded in `kit help`. A check that fires on healthy projects is worse than
no check. And the kit stays small: every addition is weighed against the thing that
makes it good, which is that a person can read all of it.
