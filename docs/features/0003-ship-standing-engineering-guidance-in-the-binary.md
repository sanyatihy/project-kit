---
id: 0003
title: Ship standing engineering guidance in the binary
status: Shipped
created: 2026-08-11
shipped: 2026-08-11
related: ../../cmd/kit/help.go
---

## Problem
A tiered Go engineering checklist and the deployment patterns behind ai-bot's setup live in
one project's `docs/` and reach no other project except by somebody remembering to copy
them. Separately, the protocol governs the feature-doc process and says nothing about how to
approach the code, which is how an agent can follow it perfectly and still sprawl.

## Non-goals
- **Scaffolding deployment files.** Terraform, ansible and images rot fastest; this ships the patterns, not the files.
- **Auditable rules.** `audit` cannot check a word of either, and a checklist it could check would be duller.
- **Provider specifics.** DigitalOcean droplets, Spaces and doctl stay in ai-bot.

## Design
Both ship as embedded `kit help` topics. Copies drift and `audit` is what notices drift —
nothing could notice a stale checklist in twenty repos, so there is one copy, updated on
upgrade.

**The checkbox syntax has to go.** In a kit project `- [ ]` means "a PR-sized unit of work,
do the topmost one"; in the source checklist it means "a property to verify". An agent
following the protocol would work through 139 of them. Converted to prose bullets, tiers
kept — skipping the tiers is what makes a checklist noise.

The conduct rules go in `templates/AGENTS-section.md`, beside the process rules, because
that section is what `kit init` regenerates into every project.

Dogfooding against the kit is a task here, not a follow-up: guidance its own author's code
cannot survive is guidance nobody should adopt.

## Tasks
- [x] `kit help go-checklist` — the tiered checklist, embedded, checkbox syntax removed
- [x] Conduct rules in `## Work protocol`: how to approach the code, not just the process
- [x] `kit help deployment` — the provider-agnostic patterns behind ai-bot's setup
- [x] Dogfood: review the kit against its own checklist; findings become BACKLOG entries or feature docs

## Log
- 2026-08-11 — dogfooding found six real things, four filed in BACKLOG.md. The strongest:
  `NewFeature` reads the clock inline while `audit` takes an injected `today`, so the kit
  already knows the rule and breaks it in one place. Three items were rejected as thumping.
