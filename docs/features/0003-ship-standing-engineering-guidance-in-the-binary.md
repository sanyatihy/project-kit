---
id: 0003
title: Ship standing engineering guidance in the binary
status: Accepted
created: 2026-08-11
shipped:
check:
checked:
related: ../../cmd/kit/help.go
---

## Problem
Two bodies of hard-won guidance live in one project's `docs/` and are invisible to
every other project: a tiered Go engineering checklist, and the deployment patterns
behind ai-bot's terraform/ansible setup. `url-shortener` is bootstrapped and about to
be deployed, and today the only way it gets either is a human remembering to copy
them. The kit already carries its own documentation in the binary — this is the same
mechanism applied to guidance that is not about the kit's process.

Separately, the eight rules in `## Work protocol` govern the *feature-doc process* and
say nothing about how to approach the code once a task is picked. That hole is why an
agent can follow the protocol perfectly and still produce sprawling, speculative work.

## Non-goals
- **Scaffolding deployment files.** Terraform, ansible, images, and dashboards stay
  ungenerated — they rot fastest and are not needed to write code. This ships the
  *patterns*, not the files. See the decision in [TECH.md](../TECH.md).
- **Auditable rules.** `audit` will not check any of this. It cannot, and a checklist
  it could check would be a much smaller and duller checklist.
- **Provider specifics.** DigitalOcean droplets, Spaces, and doctl stay in ai-bot.
  Only the provider-agnostic patterns move.
- **Copying guidance into projects.** One copy, in the binary.

## Design
Both bodies ship as embedded `kit help` topics, registered in the `topics` map in
`cmd/kit/help.go` exactly like `protocol` and `template`. Nothing is written into
scaffolded projects.

Embedding rather than copying is the whole point. `audit` exists because copies drift
and something has to notice; a checklist copied into twenty repos is drift by
construction with no mechanism to detect it. In the binary there is one copy, it
updates on upgrade, and the existing promise — an agent never needs this repository —
keeps holding.

**The checkbox syntax has to go.** In kit, `- [ ]` means "a PR-sized unit of work, do
the topmost one" (protocol rule 3). In the source checklist it means "a property to
verify". Same syntax, opposite meaning, and an agent following the protocol that meets
139 unticked boxes will start working through them. The content converts to prose
bullets; the tiering (Core / CLI / Libraries / Services / Datastore) survives, because
skipping the tiers is what makes a checklist noise.

The conduct rules go into `templates/AGENTS-section.md`, so they reach every project
through the section `kit init` already regenerates. They are about how to approach
work, so they belong beside the process rules rather than in a topic nobody prints.

Dogfooding the checklist against the kit is a task here, not a follow-up: guidance
that its own author's code cannot survive is guidance nobody else should adopt. Its
findings become their own docs, not fixes smuggled into this feature.

## Tasks
- [x] `kit help go-checklist` — the tiered checklist, embedded, checkbox syntax removed
- [ ] Conduct rules in `## Work protocol`: how to approach the code, not just the process
- [ ] `kit help deployment` — the provider-agnostic patterns behind ai-bot's setup
- [ ] Dogfood: review the kit against its own checklist; findings become BACKLOG entries or feature docs

## Open questions
<!-- none -->

## Log
- 2026-08-11 — embedding as `kit help` topics rather than writing files into each
  project: the kit's own argument against copies is that they drift and `audit` is
  what notices. Nothing could notice a stale checklist, so there should only be one.
- 2026-08-11 — the deployment patterns ship as guidance while the no-scaffolding
  decision stands. That decision was about generating files that rot; it never said
  the knowledge should stay in one repo. The patterns are what a future
  "go to production" feature doc puts in its Design section.
