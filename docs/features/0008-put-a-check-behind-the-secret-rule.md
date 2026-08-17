---
id: 0008
title: Put a check behind the secret rule
status: Shipped
created: 2026-08-12
shipped: 2026-08-12
check:
checked:
related: 0005-make-claims-and-blocking-mean-what-the-docs-say.md
---

## Problem
Rule 10 — never put a real secret in a doc — is the only rule with irreversible blast
radius and the only one with nothing behind it. The exposure grew this week: agents push
their own branches, so the gap between writing a value into a `## Log` line and that value
being on a remote is one command and no human.

## Non-goals
- **Entropy scanning.** Noisy on prose, which is what these files are; 0005 already ruled out the clever version.
- **Scanning source, config or history.** A repo-wide scanner is a different tool at a different moment.
- **Catching a secret with no known shape.** A password in prose is invisible to this and always will be.

## Design
**A handful of vendor-shaped patterns** whose issuer publishes a fixed prefix or envelope,
so a match is a secret rather than a guess. Nothing generic — no `password =`, nothing that
reads prose and infers.

**HIGH, because `--strict` is the only lever.** The severity is the exit code, not a claim
about how wrong the doc is: this is the one finding where being noticed late is
unrecoverable.

**It scans what the repo model already opens** — feature docs, INBOX, VISION, as-built. No
new traversal.

**The patterns live in code and nowhere else.** Writing the literal shapes into a template
or a help topic would make the kit's own docs trip its own check — the same reason guidance
topics carry no `- [ ]`.

## Tasks
- [x] Audit: `secret-in-doc` (HIGH) over the docs the repo model already reads
- [x] Conformance: it fires on each shape, and stays quiet on the prose that talks about them
- [x] `kit help checks`, README, and protocol rule 10 point at each other

## Log
- 2026-08-12 — built without the usage signal 0005 asked for: every other deferred check
  costs a missed finding, this one costs a published credential.
