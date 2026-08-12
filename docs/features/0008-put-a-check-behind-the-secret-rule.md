---
id: 0008
title: Put a check behind the secret rule
status: In progress
created: 2026-08-12
shipped:
check:
checked:
related: 0005-make-claims-and-blocking-mean-what-the-docs-say.md
---

## Problem
The protocol's rule 10 says never put a real secret in a doc. It is the only rule
with irreversible blast radius — a doc is committed, and a pushed commit cannot be
unpublished — and it is the one rule with nothing behind it. Every other rule that
matters has a check.

The exposure grew this week rather than staying flat. Agents now push their own
branches, so the gap between an agent writing a value into a `## Log` line and that
value being on a remote is one command and no human. And the docs are exactly where
a secret gets pasted: an agent debugging a deploy has a token in its context and a
`## Log` entry to write about what went wrong.

0005 named this and deferred it, with the design already settled: *"If it is ever
added it should be a handful of unambiguous patterns and nothing clever."*

## Non-goals
- **Entropy scanning.** Noisy on prose, which is what these files are. It is the
  "clever" that 0005 ruled out, and a check that fires on healthy docs costs more
  than it saves.
- **Scanning source, config, or history.** This checks the documents the kit already
  reads. A repo-wide secret scanner is a different tool, run at a different moment,
  and pretending `audit` is one would be a promise it cannot keep.
- **Blocking anything.** `audit` never writes and never refuses. HIGH plus
  `--strict` is the whole enforcement story, same as every other finding.
- **Catching a secret that does not match a known shape.** A password in prose is
  invisible to this and always will be. The rule still does the work; the check only
  covers the cases where a machine can be certain.

## Design
**A handful of unambiguous, vendor-shaped patterns.** Each one is a token format
whose issuer publishes a fixed prefix or envelope, so a match is a secret rather than
a guess: private key envelopes, the GitHub token prefixes, AWS access key ids, Slack
and Google API key prefixes, and a URL carrying inline credentials. Nothing generic —
no `password =`, no long-base64-blob, nothing that reads prose and infers.

**HIGH, because `--strict` is the only lever.** The severity is not a claim about how
wrong the doc is; it is the exit code. This is the one finding where being noticed
late is not recoverable, so it belongs in the band that fails a build.

**It scans what the repo model already opens** — the feature docs, INBOX, VISION, and
the as-built doc. No new traversal, no new file discovery, and nothing that turns
`audit` into a walker of the whole tree.

**The patterns live in code and nowhere else.** This check has a self-reference trap
the kit has met before: writing the literal token shapes into a template, a help
topic, or a feature doc would make the kit's own documentation trip its own check.
`kit help checks` prints what the check means, never what it matches — the same
reason guidance topics carry no `- [ ]`. This feature doc describes the shapes in
prose for the same reason.

## Tasks
- [x] Audit: `secret-in-doc` (HIGH) over the docs the repo model already reads
- [x] Conformance: it fires on each shape, and stays quiet on the prose that talks about them
- [ ] `kit help checks`, README, and protocol rule 10 point at each other

## Open questions
<!-- none -->

## Log
- 2026-08-12 — built now rather than on the usage signal 0005 asked for, because the
  cost of being wrong is asymmetric: every other deferred check costs a missed
  finding, this one costs a published credential. The design was already settled by
  0005, so waiting bought nothing.
