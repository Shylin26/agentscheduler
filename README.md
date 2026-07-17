# AgentScheduler

**AgentScheduler** is a local-first inference scheduler for concurrent LLM
agents on Apple Silicon — batches requests for throughput, and isolates
priority tiers so urgent work never waits behind background traffic.

## Why

Running several LLM agents concurrently on a single machine forces a bad
trade-off: one model instance per agent exhausts memory fast; routing every
call through one shared instance serializes everything and throws away
concurrency. AgentScheduler batches concurrent requests into a single MLX
`batch_generate` call for real throughput gains, and separates high- and
low-priority traffic into isolated queues so one doesn't block the other.

## Architecture

- **Go scheduler** — accepts requests over HTTP, queues them by priority
  tier, collects short-lived batches, dispatches to the batch server.
- **Custom MLX batch server** (`batch_server.py`) — wraps `mlx-lm`'s
  `batch_generate`, loading the model once and serving batched completions
  over HTTP.
- **Priority queues** — `highQueue` / `normalQueue`, each collected and
  batched independently, with a starvation guard so low-priority traffic
  is never indefinitely blocked.

## Performance

**Batching** — controlled benchmark, same model/prompt/token-limit,
isolating batch size as the only variable:

| Batch size | Sequential | Batched | Speedup |
|---|---|---|---|
| 1 | 0.266s | 0.288s | 0.92x (sanity check) |
| 2 | 0.509s | 0.356s | 1.43x |
| 4 | 1.099s | 0.547s | 2.01x |
| 8 | 2.052s | 0.936s | 2.19x |
| 16 | 4.051s | 1.702s | 2.38x |

**Priority isolation** — one high-priority request submitted concurrently
alongside four low-priority ones:

| Tier | Latency |
|---|---|
| High priority (isolated batch of 1) | 104ms |
| Low priority (batch of 4) | 624ms |

See [docs/0001](docs/0001-prefix-hashing-scope.md) and
[docs/0002](docs/0002-priority-scheduling-scope.md) for the full
methodology, including a first priority-scheduling design that measurably
*didn't* work, why, and how it was fixed.

## Usage

```bash
# Terminal 1: start the MLX batch server
python3 batch_server.py

# Terminal 2: start the scheduler
go run .

# Terminal 3: submit a request
curl http://127.0.0.1:9000/submit \
  -X POST -H "Content-Type: application/json" \
  -d '{"prompt": "Say hello.", "priority": 8}'
```

## Status

Working: HTTP-exposed scheduler, real batching, prefix-hash tagging
(near-duplicate detection, see docs/0001), priority queue isolation
(verified working, see docs/0002).

Not yet built: real prefix/prompt-cache reuse, sustained mixed-load
testing beyond a single burst. This is Phase 2 of a larger project — see
[agentmemory](https://github.com/Shylin26/agentmemory) for the versioned
memory layer this will eventually couple with.

## Setup

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```