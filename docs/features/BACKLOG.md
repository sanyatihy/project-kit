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
