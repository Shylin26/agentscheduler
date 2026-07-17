# 0001: Prefix Hashing — Current Scope and Limitations

**Status:** Implemented in v1, narrower in practice than the name implies. Revisit for real prefix-sharing detection.

## What exists

Every incoming request is tagged with a SHA-256 hash of its first 100
characters (`computePrefixHash`). Within a collected batch, requests are
sorted by this hash (`sortByPrefixHash`) so that requests sharing a hash
land adjacent to each other before being sent to the batch server.

## What this actually detects

Because most real-world prompts used in testing are well under 100
characters, this hashes the **entire prompt** in practice, not a true
leading substring. Two prompts that share an opening phrase but diverge
later — e.g. `"Tell me about dogs"` vs `"Tell me about cats"` — hash
completely differently, due to the avalanche effect of SHA-256. Verified
directly: two identical prompts produced matching hashes; two
prefix-sharing-but-different prompts did not.

**In its current form, this mechanism detects near-duplicate requests, not
requests sharing a meaningful, reusable prefix.**

## Why this still has value

Exact-duplicate detection isn't nothing — multiple agents asking the
identical question (a real scenario in coalition-style multi-agent systems)
now get grouped adjacently in a batch. It also establishes the plumbing
(`PrefixHash` field, sorting logic) that real prefix-sharing detection can
build on later without restructuring the request pipeline.

## What real prefix-sharing detection would require

- Tokenizing prompts consistently with the model's actual tokenizer (which
  lives in Python/`mlx-lm`, not Go) — hashing by token count, not raw
  character count, since token boundaries are what actually matter for
  KV-cache reuse.
- Hashing a genuinely short, fixed leading window (e.g. the first N tokens)
  independently of overall prompt length, rather than the whole prompt
  when it happens to be short.
- Likely computing this hash the same way both in Go (for scheduling
  decisions) and Python (for actual cache-key lookups in Phase 3's
  CacheRouter) — meaning the hashing scheme may need to move to whichever
  side actually owns the KV-cache, or be kept consistent across both.

## Revisit when

- Phase 3 (CacheRouter) needs real prefix-cache-hit prediction, not just
  batch grouping.
- Testing moves to longer, more realistic agent prompts (shared system
  instructions, longer context) where character-count-based hashing starts
  behaving like true prefix hashing rather than near-duplicate detection.