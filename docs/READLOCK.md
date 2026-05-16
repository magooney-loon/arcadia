# Frontend read-lock investigation

The frontend (SSE + REST) freezes intermittently while the indexer is sprinting through catch-up. Symptom: API reads stall for hundreds of ms to several seconds during periods of high indexer throughput. WAL, prefetch, adaptive pacing, and `wal_autocheckpoint=256` are already in place, so the easy levers are pulled. This doc traces the root cause and the fix plan.

---

## What was already in place

- `cmd/server/main.go:89-105` — `journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=2000`, `cache_size=-8000`, `mmap_size=268435456`, `wal_autocheckpoint=256`.
- `internal/indexer/runner.go:126` — 50-block batches (target write tx <200ms).
- `internal/indexer/runner.go:110-134` — prefetch goroutine overlaps WAN RTT with SQLite write time.
- `internal/indexer/runner.go:252-269` — adaptive pacing: 0ms sleep when lag≥500, 50ms when ≥100, 200ms near tip.
- `internal/indexer/batch.go:39-42` — bulk dedupe (`loadBatchSeen`) replaces per-row SELECTs.

## Ruled out from the initial writeup

- "200-block batches" — they're 50.
- "Default `busy_timeout=1000`" — already 2000.
- "`PRAGMA wal_autocheckpoint=0` + passive checkpoint ticker" — PASSIVE checkpoints make no progress while a writer is active, so under sustained sprint load the WAL would grow unbounded. Not pursuing without a TRUNCATE/RESTART fallback during idle windows.
- SSE itself holding locks — long-lived HTTP reads don't hold DB locks; only the payload-composition reads do.
- The per-agent read-modify-write loop at `batch.go:248-275` — measured at ~15ms avg, negligible.

---

## Phase 1 — Profiling (done)

### Instrumentation

`internal/indexer/batch.go` was wired with `time.Now()` markers around each phase inside and around `RunInTransaction`. Emits one line per batch:

```
[indexer] batch_profile blocks=N txs=N logs=N traces=N | seen=Nms tx_total=Nms | blocks=Nms txs=Nms logs=Nms traces=Nms backfill=Nms stats=Nms edges=Nms agents=Nms
```

Phases:
- `seen_ms` — `loadBatchSeen` (pre-tx).
- `blocks_ms` — initial block inserts.
- `txs_ms` — transaction inserts + accumulator updates.
- `logs_ms` — log routing + edge/agent delta accumulation.
- `traces_ms` — trace inserts.
- `backfill_ms` — `seen.newBlocks` re-save loop.
- `stats_ms` — `block_stats` insert loop.
- `edges_ms` — `flushEdgeDeltas`.
- `agents_ms` — per-agent read-modify-write loop.
- `tx_total_ms` — wall time inside `RunInTransaction`.

### Summary script

`scripts/profile_batches.sh` — awk pipeline that prints count, avg, p50, p95, max per phase. Handles the `=Nms` vs `=N` ambiguity by tagging timings with a `_ms` suffix.

Usage:
```bash
# capture logs while the app runs
mkdir -p logs
pb-cli --run-only 2>&1 | tee logs/profile.log     # use tee -a if you want to append across restarts

# summarise
./scripts/profile_batches.sh logs/profile.log

# sprint vs near-tip slices
grep batch_profile logs/profile.log | head -200 | ./scripts/profile_batches.sh -
grep batch_profile logs/profile.log | tail -200 | ./scripts/profile_batches.sh -
```

### Results (fresh DB, sprint catch-up, 50-block batches)

Two runs measured:

**Run 1** — fresh DB, 32 batches during sprint catch-up:

| phase | avg | p95 | share of `tx_total` |
|---|---|---|---|
| `logs_ms` | 1538ms | 3660ms | ~64% |
| `edges_ms` | 504ms | 724ms | ~21% |
| `txs_ms` | 242ms | 339ms | ~10% |
| `blocks_ms` | 11ms | 24ms | <1% |
| `agents_ms` | 15ms | 21ms | <1% |
| `backfill_ms` | 4ms | 13ms | <1% |
| **`tx_total_ms`** | **2406** | **4710** | — |

**Run 2** — restart after a few minutes (token_analytics warm but with a fresh sprint gap), 17 batches:

| phase | avg | p95 |
|---|---|---|
| `logs_ms` | 2295ms | 4310ms |
| `edges_ms` | 809ms | 1089ms |
| `txs_ms` | 276ms | 334ms |
| **`tx_total_ms`** | **3528** | **5456** |

### Interpretation

- Write lock is held for ~2-5 seconds per batch under sprint load. That fully explains the frontend freezes — a reader hitting the DB while a batch is processing waits up to ~5s before its query can run.
- `logs_ms` is dominant (60-65% of `tx_total`).
- `edges_ms` grows with the size of the `wallet_edges` table: Run 2 has higher `edges_ms` than Run 1 because the table now has rows from Run 1 — every `app.Save` re-indexes against a larger row set.
- New tokens show up continuously, so the in-memory cache never fully cools the RPC path even after restart.
- 17 MB WAL observed at runtime (vs ~1 MB target with `wal_autocheckpoint=256`): consistent with the writer never releasing the lock long enough for the autocheckpoint to make progress.

---

## Root cause of `logs_ms`

`internal/indexer/save_transfer.go:43` — `saveTransfer` calls `rpc.LookupTokenInfo` for every Transfer log. On cold-cache token addresses, `LookupTokenInfo` (`internal/rpc/erc.go:69`) does:

1. In-memory cache check (fast).
2. **DB read**: `app.FindRecordsByFilter("token_analytics", ...)` — read inside the write transaction.
3. **On still-miss**: `fetchTokenInfoFromRPC` (`erc.go:166`) — up to **4 sequential JSON-RPC HTTP round-trips** per token (`decimals` → `supportsInterface(721)` → `supportsInterface(1155)` → `symbol/name`), retried across `chain.ArcRPCPool`. `rpcHTTPClient` has `Timeout: 8 * time.Second` per call (`erc.go:47`).
4. **DB write**: `app.Save(r)` to persist the new `token_analytics` row — inside the write transaction.

For each batch with ~5-20 cold tokens, the writer holds the SQLite write lock open across all of those HTTP round-trips. That's the freeze.

## Root cause of `edges_ms`

`internal/indexer/aggregator.go:69-100` — `flushEdgeDeltas` does:

1. One bulk read of existing edges via `loadEdgesFor` (good).
2. Per-edge `app.Save` (bad) — each save is a separate UPDATE or INSERT with its own index maintenance. For ~hundreds of edges per batch, this dominates once the table grows.

There's no concurrent-HTTP problem here; it's pure SQLite write volume.

---

## Phase 2 — Fix plan

### Fix 1 — Move token metadata resolution out of the write transaction (priority)

**Goal:** by the time `RunInTransaction` runs, every token address referenced in the batch's logs is already in the in-memory cache.

**Shape of the change** in `internal/indexer/batch.go`, before `RunInTransaction`:

1. Walk `res.Data.Logs`, collect distinct `log.Address` where `log.Topic0 == chain.TopicTransfer` and the address is not already in the cache.
2. Resolve them concurrently via an `errgroup` bounded to a small pool (4-8 workers).
3. Persist new `token_analytics` rows. Two options:
   - (a) write them in a short pre-tx (separate `RunInTransaction` with only inserts — milliseconds).
   - (b) skip the persist on the hot path entirely; let a background job seed `token_analytics` from cache misses. Simpler, but loses per-batch durability of new token metadata.
4. The main `RunInTransaction` now only does reads from the warm in-memory cache when `saveTransfer` calls `LookupTokenInfo` — no network, no DB read, no DB write for token metadata.

**Expected effect:** `logs_ms` drops to per-log time of inserts only (~0.3-0.5ms/log), so a batch with 1000 logs becomes ~300-500ms instead of 1500-3000ms.

**Edge cases:**
- A token that fails every RPC (`LookupFailed: true`) must still be cached so it doesn't get re-tried every batch.
- The existing `cacheTokenInfo` eviction at `erc.go:118` is FIFO past 5000 entries; verify the prefetch doesn't churn the cache when batch token count exceeds the cap.
- Errors from the prefetch should not fail the whole batch — fall through with `LookupFailed: true` and continue.

### Fix 2 — Bulk upsert for `wallet_edges`

**Goal:** replace per-edge `app.Save` with a single raw-SQL upsert.

**Shape:** in `flushEdgeDeltas` (`aggregator.go:52`), construct one `INSERT INTO wallet_edges (...) VALUES (...), (...), ... ON CONFLICT(from_wallet, to_wallet) DO UPDATE SET total_usdc = total_usdc + excluded.total_usdc, tx_count = tx_count + excluded.tx_count, last_seen_block = MAX(last_seen_block, excluded.last_seen_block)`. Bypasses PocketBase's per-record Save overhead and lets SQLite do one index update per row in one statement.

**Expected effect:** `edges_ms` collapses from hundreds of ms to tens.

**Caveat:** need a UNIQUE index on `(from_wallet, to_wallet)` for `ON CONFLICT` to work. Verify the schema before shipping.

### Order of work

1. Implement Fix 1.
2. Re-measure with the same `profile_batches.sh` flow. Expect `logs_ms` to drop ~5×.
3. Implement Fix 2.
4. Re-measure. Expect `tx_total_ms` p95 to drop from ~5s to well under 1s.

### Not pursuing (yet)

- Phase-splitting `RunInTransaction` into multiple smaller txs. Once Fixes 1 and 2 land, total tx time should be short enough that splitting adds complexity without meaningful benefit. Revisit only if data says otherwise.
- Runtime sync-mode toggle / `/api/sync/pause` endpoint. Was on the table as a UX patch; if the root-cause fixes work, it becomes optional polish.
- Reader connection pool isolation. Not needed if the writer releases the lock quickly.
- `BroadcastIndexerUpdate` audit. Worth a glance but unlikely to dominate after the writer-side fixes.
