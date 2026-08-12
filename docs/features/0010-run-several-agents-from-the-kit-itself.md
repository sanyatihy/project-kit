---
id: 0010
title: Run several agents from the kit itself
status: Draft
created: 2026-08-12
shipped:
check:
checked:
related: 0006-make-the-kit-safe-for-concurrent-agents.md
---

## Problem
One agent, one prompt, a whole project: that already works and was measured — a
working URL shortener in twenty minutes with the protocol followed unprompted.
Running *several* agents does not, and the gap is not knowledge. It is that the tool
which starts one lives in an untracked file on a single machine.

0006 said worktree management was git's job and a launcher's. 0007 repeated it and
put the launcher in BACKLOG, with a stated trigger: build it when the same script
turns out to be needed in every project. It is. Every project wants the identical
sequence — resolve a feature id to its doc, claim it, put it in a worktree, start an
agent there — and only the last step is not already kit-shaped.

The reason it must live in `kit` rather than beside it is discovery. The whole flow
rests on an agent finding what it can do by running `kit`, with no repository and no
briefing. A separate script is invisible to that: an agent that cannot see a launcher
does not go looking for one, it runs `git worktree add` by hand, which is exactly what
protocol rule 2 forbids and what makes a claim nobody else can see.

## Non-goals
- **Becoming a process supervisor.** No restarting, no health checks, no daemon.
  Panes hold processes; that is the multiplexer's job and it already does it.
- **Choosing what to run.** `kit next` says what is ready and this starts it. There is
  no scheduler and no queue.
- **Removing a worktree when a pane dies.** Removal is what says a feature is done,
  and it stays a decision rather than a side effect of closing a window.
- **Supporting every multiplexer.** One, with its calls confined so a second is a
  swap. Guessing at an abstraction over multiplexers nobody has asked for is how this
  gets heavy.
- **Deciding how many agents run at once.** Still a dial with a default, still not the
  kit's opinion, same as 0006.

## Design
**`kit agent start|list|attach|kill`, a subcommand rather than a sibling binary.**
Discovery is the whole argument: `kit` with no arguments is what an agent reads, so
capabilities that are not listed there do not exist. A second binary would have to be
found, and nothing would point at it.

**Everything durable is already in git and tmux.** The worktree directory is the
feature, a pane-scoped option carries the id, and the assigned session id lives in the
worktree. No state file the kit has to keep in sync, which is the same reason the
board is markdown and not a database.

**Panes are addressed by an option the agent cannot overwrite, never by pane title.**
Measured: a coding agent sets its own terminal title within a second of starting, so
titles silently detach panes from their features.

**Killing a pane is not enough to stop an agent.** It survives the hangup and keeps
running detached. Stopping means the process, then the pane — also measured, after
three orphans.

**`start` claims through `kit claim`.** It must not create the branch itself: the push
is what decides, and a hand-made branch is the invisible claim rule 2 exists to
prevent. Losing the race is a normal outcome and reported as one.

**The tmux calls live in one file** so replacing the multiplexer is a swap. This is
worth doing now precisely because the credible replacements are being written now.

**The dependency gets stated, not hidden.** `kit` claims git as its only runtime
dependency. That becomes git, plus tmux for this one subcommand, and everything else
keeps working without it. `kit agent` on a machine without tmux says so and exits 2.

**A spawned agent needs its instruction, not just a shell.** Measured: starting one
with no prompt leaves an idle prompt nobody is typing into. `start` passes an opening
instruction so the new agent runs `kit next` and works what it was given.

**What gets run is configuration, not a hard-coded name.** Measured, and it is the
reason this matters: a freshly created worktree is a new directory, so the harness
stops the new agent on a trust prompt and it waits there forever. Getting past that is
harness-specific — a flag, a pre-seeded setting, a sandbox — and none of it belongs in
`kit`, which would otherwise be pinned to one vendor's config file at the exact moment
that vendor is being replaced. `kit agent` runs a configured command in a prepared
directory; the workaround lives in the operator's configuration where it can change.

**Where an unaccepted `Draft` lives.** The protocol has agents draft and leave the doc
for a human, and never says where it sits meanwhile. It sits on a branch: nothing is
committed to `main`, ever. `kit claim` still refuses Drafts, so the branch is not a
claim, and `draft-claimed` has to tell the two apart — a branch holding a doc awaiting
review is the documented flow working, not somebody building on an unsettled design.

## Tasks
- [ ] A branch may hold a `Draft` awaiting review without `draft-claimed` calling it a claim
- [ ] `kit agent start <id>` — claim, worktree, pane, assigned session id, opening instruction
- [ ] The spawned command is configurable, defaulting to the obvious one
- [ ] `kit agent list` — what is running, its feature and its worktree
- [ ] `kit agent attach <id>` and `kit agent kill <id>`, where kill stops the process before the pane
- [ ] A concurrency dial with a default, and a refusal that names it when reached
- [ ] `kit help agents`, and the runtime dependency stated wherever the kit claims it has none
- [ ] Retire `~/.local/bin/agent` in favour of this

## Open questions
<!-- none -->

## Verified before building
Run from a session with no multiplexer, against a throwaway project:

- Creating the tmux session, the board pane and an agent pane all work from a process
  that is not itself inside tmux. The board rendered `0001 Add a greeting — taken`
  live, which settles the board question: it is created with the session.
- The agent pane started in the right worktree with the assigned session id.
- It then stopped on the harness's folder-trust prompt and stayed there. That gate is
  per directory, so it fires for every new worktree — the one thing standing between
  this and an unattended fan-out, and not something the kit can fix.

## Log
- 2026-08-12 — tested the fan-out before designing it, from a session with no tmux.
  Two things it changed: a spawned agent needs an opening instruction or it idles,
  and the harness stops every new worktree on a folder-trust prompt. The second is
  why the command being launched is configuration — the workaround is vendor-shaped
  and the kit must not carry it.
- 2026-08-12 — reversed 0006 and 0007 on where the launcher belongs. Their argument
  was that a tmux launcher ties the kit to one multiplexer, which is true and is
  handled by confining the calls. What they missed is that an agent only uses what
  `kit` advertises, so a launcher outside the binary does not get used — it gets
  reinvented, badly, as the hand-made branch rule 2 forbids.
