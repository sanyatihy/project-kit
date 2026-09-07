---
id: 0043
title: Serve repeated customer lookups from a per-process cache
status: Draft
created: 2026-09-07
submitted:
needs: 0041
---

## Problem
Support operators repeatedly run the same customer lookup within seconds, and each one repeats the identical upstream request. With a per-process cache, the second and later lookups inside a 30-second window come back from memory instead of upstream, cutting duplicated upstream load and operator wait time. Observable result: two identical lookups issued less than 30 seconds apart in one process produce exactly one upstream request.

## Non-goals
- A shared cache across workers. Cross-worker invalidation is unavailable in this deployment, so a shared cache would serve stale entries with no way to expire them; each worker keeps its own per-process cache.
- Long-lived or persistent caching. Entries live at most 30 seconds and vanish with the process; nothing is written to disk.

## Design
The lookup path checks an in-memory map keyed by the lookup input before calling 0041's lookup API; a miss calls upstream once and stores the result with a 30-second expiry, and later identical lookups within that window are served from the cache. The cache is per-process: one map per running process, no shared storage. Accepting up to 30 seconds of stale data is the cost of this design — an operator may see a customer record changed upstream for at most 30 seconds. Editing the same file as 0042 is not a dependency; only the lookup API itself must be integrated first, hence `needs: 0041`.

## Tasks
- [ ] Add the per-process cache to the lookup path: repeated identical lookups within 30 seconds produce one upstream call; after expiry, the next lookup calls upstream again.
- [ ] Verify with a test that two identical lookups issued less than 30 seconds apart issue exactly one upstream request and return equal results.

## Log
