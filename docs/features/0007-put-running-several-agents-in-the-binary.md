---
id: 0007
title: Put running several agents in the binary
status: Draft
created: 2026-08-12
shipped:
check:
checked:
related: 0006-make-the-kit-safe-for-concurrent-agents.md
---

## Problem
0006 made the kit safe for concurrent agents and told each agent what to do: claim
with `kit claim`, work in the worktree it prints, never start a second feature where
you stand. That is the agent's half, and it is finished.

Nothing tells the operator how to run the fleet. Standing up several agents was done
by hand this week, and everything learned doing it lives in a shell history and two
untracked dotfiles. The kit's stated promise is that an agent never needs this
repository to use the tool; on this subject the tool says nothing at all, and
`README.md` says three sentences written before any of it was tried.

Four of the things learned cost a debugging session each, and none is guessable:

- **A pane title is not a handle.** Claude Code sets its own terminal title, so a
  launcher that labels panes by feature id loses track of every one of them the
  moment its agent starts. Measured: the title we set was gone a second later.
- **Killing the pane does not kill the agent.** It survives the hangup and keeps
  running detached, so the process outlives the thing that was supposed to bound it.
  Three orphans were left behind before this was noticed.
- **Working directory is the join key for everything.** Sessions are stored under
  it, and the editor binds to the window whose folder matches it. That is what makes
  a worktree self-contained — and it is also why a worktree starts with none of the
  parent checkout's agent memory.
- **One installed binary serves every worktree.** In any project that builds the
  tool it is being driven by, an agent that installs its own unmerged build changes
  what every other agent is running.

These are properties of the substrate, not of anyone's preferences, and every one of
them is found the same way: by losing an hour.

## Non-goals
- **Shipping a launcher.** 0006 settled that worktree management is git's job and a
  launcher script's job. That still holds, and a tmux launcher would tie the kit to
  one multiplexer at the moment credible replacements are being built. The topic
  describes what a launcher must do and why; writing one stays the operator's.
- **Scaffolding any of it into projects.** Same reason `deployment` and
  `go-checklist` are not written into repos: `audit` cannot check a word of it, so a
  copy per project is drift nothing could detect.
- **Recommending how many agents to run.** 0006 already refused to encode a count,
  and nothing since has produced the data to change that.
- **`kit next --claim`, cross-feature dependencies.** Still deferred, still waiting
  on a signal that has not arrived.

## Design
**A new registered topic, `kit help agents`.** Same shape as `deployment` and
`go-checklist`: standing guidance embedded in the binary, from a file in
`templates/`, so `kit help` cannot drift from what exists. It is the operator's
counterpart to `kit help protocol`, which is the agent's.

**The topic is the substrate, not the workflow.** What belongs in it is what is true
regardless of taste: a worktree is one feature's checkout, a session belongs to a
working directory, a process outlives the pane it was started in. What does not
belong is a layout, a pane count, or an editor. The test for a line is whether it
would still be true under a different multiplexer.

**The four traps are the payload.** The model is a paragraph and is close to
obvious; the traps are the part nobody derives, and each is stated with what it
looks like when it bites, because that is how it will be recognised.

**A launcher is described by its obligations, not its code.** Address panes by
something the program cannot overwrite; kill the process, not the pane; let the
worktree outlive the pane so removing it stays a decision; assign the session
identifier rather than discovering it. Those four hold for anything that drives
agents, which is why they are worth writing down and a script is not.

**`README.md` defers rather than competes.** One source of truth per fact is already
a convention here; the `## Several agents` section becomes a pointer.

## Tasks
- [ ] `kit help agents` — a registered topic: worktree per feature, session per working directory, and the board as the answer to "who needs me?"
- [ ] The topic carries the four traps, each with what it looks like when it bites
- [ ] The topic states a launcher's four obligations without shipping one
- [ ] `README.md`'s `## Several agents` defers to the topic instead of restating it
- [ ] Conformance: the topic prints, `kit help` lists it, and it contains no `- [ ]`

## Open questions
- Should the kit ship a reference launcher after all? Against: 0006's non-goal, and
  betting on one multiplexer while its replacements are being written. For: every
  project that runs agents needs the same script, and the parts that resolve a
  feature id to a doc, a branch, and a worktree path are kit-shaped rather than
  substrate-shaped. Leaning: no, and revisit if a second project writes the same
  script — that is the signal, not the anticipation of one.

## Log
- 2026-08-12 — the four traps were all found by running the thing rather than
  designing it. Recorded because it is the argument for the topic existing: none of
  them appears in any documentation, each is a property of the substrate rather than
  a preference, and each cost an hour that a paragraph would have saved.
