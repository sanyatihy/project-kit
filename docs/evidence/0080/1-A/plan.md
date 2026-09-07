---
id: 0043
title: Resolve repeated customer lookups from a per-process cache
status: Draft
created: 2026-09-07
submitted:
needs: 0041
---

## Problem

Support operators making the same customer lookup twice within 30 seconds send two identical
upstream requests. The signal is repeated lookups for one customer inside a short window, each
one an upstream call with its latency and cost paid again.

## Non-goals

- Cross-worker cache sharing. Invalidation between workers is unavailable in this deployment,
  so data could silently diverge; correctness over hit rate.
- Caching other endpoints or lookup types. Only repeated customer lookups show the repeated
  upstream traffic.

## Design

The cache is per-process, keyed by the customer lookup's identity as defined by 0041's lookup
API. An entry expires after 30 seconds; a hit within that window returns the cached result
instead of calling upstream. A miss or an expired entry makes the upstream request and stores
the result.

A shared cache was rejected: it needs cross-worker invalidation, which this deployment does not
have. The accepted cost is up to 30 seconds of stale data per worker, and each process pays its
own memory for its own entries.

## Tasks

- [ ] The lookup path consults a per-process cache before calling upstream; an identical
      lookup within 30 seconds returns the cached result without a second upstream request.
- [ ] A lookup older than 30 seconds, or for a different customer, misses and calls upstream,
      storing the fresh result.
- [ ] make check passes with the cache in place, including the new tests for hit, expiry and
      miss behavior.

## Log

<!-- One entry per change to the plan. Empty: nothing has changed since drafting. -->
