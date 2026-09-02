---
id: 0009
title: Fix what the first real run found
status: Shipped
created: 2026-08-12
shipped: 2026-08-12
related: 0008-put-a-check-behind-the-secret-rule.md
---

## Problem
An agent built a working URL shortener from one prompt in twenty minutes, following the
protocol unprompted. The run found three defects:

- `secret-in-doc` fired on a doc doing the right thing — one explaining that URLs
  carrying userinfo are rejected, and quoting the form to say so. HIGH, so `--strict` failed
  a build over correct documentation. Seven patterns are anchored to an issuer's published
  prefix; the URL one is structural, and anything discussing URL credentials matches it.
- Nothing said who merges. The agent reached the end of `kit ship`, found no
  instruction, and merged its own work to `main` unreviewed.
- The binary was two features behind, so `init` wrote no VISION, the protocol had no
  drafting rule and the secret check never ran. Everything 0007 and 0008 shipped went
  untested by the run meant to test it.

## Non-goals
- Narrowing the URL pattern instead of removing it. Telling a real password from `pw` in an example needs the placeholder heuristic 0005 ruled out.
- Making the kit enforce who merges. `audit` never writes and never refuses; the protocol states the boundary and `ship` says it at the moment it matters.
- Version-checking machinery. A binary that phones home buys less than it costs.

## Design
Delete the eighth pattern. `secret-in-doc` keeps only issuer-anchored shapes. The cost
stated plainly: a real password inside a connection string now goes unflagged — rule 10
already says the check is a backstop, and this makes the boundary honest rather than ragged.
The quiet test missed it by using a connection string with no password, so the regression
case is the doc that actually broke it.

Shipping is a handoff: push the branch and stop, merging is not the agent's. `kit ship`
prints it too, because the protocol is read at the start of a session and this decision is
made at the end of one. With no remote, say so rather than letting work land unreviewed.

A house rule for this repo: reinstall after merging, or every other project keeps
running the old kit.

## Tasks
- [x] `secret-in-doc` drops the URL shape, with the doc that broke it as the regression case
- [x] Protocol and `kit ship`: shipping hands off, merging is not the agent's, and say when there is no remote
- [x] House rule: reinstall after merging, or every other project keeps running the old kit

## Log
- 2026-08-12 — removed the URL shape rather than tuning it: 0008 stated the admission rule
  (an issuer-published prefix or envelope) and this is the one entry that never met it.
- 2026-08-12 — two of the three defects are invisible from inside the repo. Nothing in the
  kit could report that the protocol had no answer for merging, or that the binary was stale.
