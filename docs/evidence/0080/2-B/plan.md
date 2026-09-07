---
id: 0043
title: Serve repeated customer lookups from a per-process cache within 30 seconds
status: Draft
created: 2026-09-07
submitted:
needs: 0041
---

## Problem
Support operators resolving the same customer repeatedly hit the upstream service once per request. Within a short window, identical lookups are repeated and each one pays the full upstream cost. The observable improvement: for the same lookup key, a second request within 30 seconds is served from the in-process cache instead of making the same upstream request twice — upstream receives one call where it previously received two.

## Non-goals
- A shared, cross-worker cache. None: cross-worker invalidation is unavailable in this deployment, so a shared cache could serve stale entries with no way to invalidate them. Per-process caching belongs here; anything more is a separate feature.
- Cache entries older than 30 seconds. Stale data beyond that window is out of scope for the accepted freshness bound.

## Design
The cache is per-process: each worker process keeps its own in-memory map from lookup key to result with a 30-second expiry, and only a miss or an expired entry triggers the upstream request. A shared cache was considered and rejected because this deployment has no cross-worker invalidation, so one worker's write could not invalidate another worker's copy. The accepted cost is up to 30 seconds of stale data per process, plus each process issuing its own first upstream call after expiry. The lookup API from 0041 must be integrated into trunk and this checkout before implementation; this feature builds on that API and changes nothing about it. Feature 0042 edits the same file but makes no decision this feature depends on, so it is not listed in `needs:` — shared files alone are not dependencies.

## Tasks
- [ ] 0041's lookup API is integrated into trunk and present in this checkout (dependency verified before any cache code lands).
- [ ] A repeated customer lookup within 30 seconds returns the cached result without issuing a second upstream request for the same key (verified: one upstream call observed for two identical lookups inside the window).
- [ ] The same lookup after the 30-second expiry issues a fresh upstream request and refreshes the cache entry (verified: second upstream call observed once the window lapses).
- [ ] Two worker processes serving the same key each hold their own entry; no cross-process coordination or invalidation is attempted (verified: entries live only in process memory).

## Log
