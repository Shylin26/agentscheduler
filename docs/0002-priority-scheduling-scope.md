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
Average low priority:  382.04ms
Average high priority: 466.57ms

High priority came out *slower* on average, driven substantially by one
outlier round (742ms). Round-to-round variance (50-80ms) was larger than
the expected effect size (an 8ms window difference against a ~400ms
baseline, roughly 2%). **No clear, reproducible speedup was demonstrated.**

## Follow-up: mixed-priority contention test (conclusive)

A second test fired 4 low-priority and 1 high-priority request concurrently
in a single genuinely mixed batch, measuring individual per-request
latency (`cmd/priority_bench`'s `runMixed`). Result:
low-3 finished in 519.968292ms
low-2 finished in 520.027ms
low-1 finished in 520.042125ms
HIGH-0 finished in 520.010875ms
low-0 finished in 520.033917ms
All five requests, including the high-priority one, finished within ~75
microseconds of each other. **The priority mechanism has no effect in the
actual scenario it's meant to address.**

## Root cause (now confirmed, not speculated)

`collectBatch` selects a collection window based only on the *first*
request to arrive in a new batch — it does not inspect or treat other
requests in the same batch differently. Once multiple requests (regardless
of priority) land in the same batch, they are sent to `batch_generate`
together and necessarily finish at the same time, since batched generation
computes all prompts in one call. **Priority currently can only affect
which of two window lengths an entire batch uses — it cannot make one
request within a shared batch finish before another.** This is a structural
limitation of window-based collection, not a tuning problem (e.g. not
fixable by adjusting the 2ms/10ms constants).

## What would actually be required

Real per-request priority differentiation would require **not** batching
high- and low-priority requests together at all — e.g., maintaining
separate batch queues per priority tier, with the scheduler always
draining/sending the high-priority queue first, even if that means smaller,
less efficient batches for urgent requests. This is a materially different
architecture from the current single-queue, single-collectBatch design.

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

  ## Resolution: separate per-priority queues (confirmed working)

Replaced the single shared queue with `highQueue` and `normalQueue`.
`Submit` routes requests by priority at submission time. `collectBatch`
locks a batch to a single source queue based on the first request's tier
— a high-priority batch only ever pulls further high-priority requests,
never low-priority ones, for its entire collection window (and vice
versa). A starvation guard (`forceNormal` every 4th batch) prevents
high-priority traffic from indefinitely blocking normal-priority work.

Confirmed via direct batch-composition logging that batches are now
strictly homogeneous by priority tier — never mixed. Confirmed via the
mixed-load test that a lone high-priority request, when isolated into its
own batch, finished in ~104ms versus ~624ms for the low-priority batch it
was previously trapped alongside. This is the actual, empirically verified
fix for the limitation identified above — not the window-tuning approach
first attempted, but true queue-level isolation.

**Caveat worth naming:** this test isolated one high-priority request
against four low-priority ones. Behavior under sustained, heavy mixed
load (rather than a single burst) — particularly whether the starvation
guard's fixed 1-in-4 ratio is well-tuned — is not yet tested.