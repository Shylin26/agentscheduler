# 0002: Priority Scheduling — Mechanism vs. Measured Effect

**Status:** Implemented, correctness-testable, performance benefit unproven
by current benchmarks. Revisit test design before claiming a speedup.

## What exists

Every `ScheduledRequest` carries a `Priority int` (higher = more urgent).
`collectBatch` inspects the *first* request in a new batch and selects a
shorter collection window (2ms) if its priority is at or above a threshold
(5), versus the normal window (10ms) otherwise. Within a batch, requests
are sorted by priority (descending), with prefix hash as a tie-breaker.

## What we tried to measure

A controlled benchmark (`cmd/priority_bench`) fired 5 rounds of 5 concurrent
requests each, alternating low-priority (0) and high-priority (8) batches,
with a discarded warm-up round, specifically to avoid the order-effect
confound found during initial ad-hoc testing (an early, uncontrolled test
showed high priority ~3.7x faster, which further investigation attributed
to MLX/model warm-up state rather than the priority mechanism itself).

## What we actually found
High priority came out *slower* on average, driven substantially by one
outlier round (742ms). Round-to-round variance (50-80ms) was larger than
the expected effect size (an 8ms window difference against a ~400ms
baseline, roughly 2%). **No clear, reproducible speedup was demonstrated.**

## Why the test itself was likely the wrong design

Each `runBatch` call sends N requests that all share the *same* priority
value, so they always land in the same collection window together — this
tests "does one uniform-priority batch process faster with a shorter
window," not the actual scenario priority scheduling is meant to address:
**a high-priority request arriving concurrently alongside unrelated
low-priority traffic, and getting served sooner because of it.** The
current benchmark never creates that mixed-priority contention scenario.

## What a real test would require

- Concurrent, *mixed*-priority load — low and high priority requests
  arriving interleaved in the same time window, not tested as separate,
  uniform batches.
- Measuring latency *per individual request*, not just total batch
  wall-clock time — the real question is whether a single high-priority
  request's response time improves when it's competing with background
  load, not whether a whole batch of same-priority requests is faster.
- A larger, more realistic effect size to test against — the current 2ms
  vs 10ms window gap may simply be too small to matter at these request
  volumes and model sizes; worth testing with a bigger gap (e.g. 1ms vs.
  50ms) to see if the mechanism has *any* measurable ceiling effect before
  concluding it's the test design alone at fault.

## Revisit when

- Building the real mixed-priority contention test described above.
- If that test also shows no effect, reconsider whether window-tiering is
  the right mechanism at all, versus e.g. a genuine priority queue that
  reorders pending requests ahead of others still waiting, not just
  shortening the collection window for the request that happens to arrive
  first.