# Arcadia internal/ audit

Read everything under `internal/` (4.7K LOC) plus `cmd/server/main.go`. The architecture is sound — HyperSync indexer → SQLite (PocketBase) → REST handlers + cron-driven snapshot jobs — but it's bottlenecked by N+1 reads on the hot path and a few "load everything to count" anti-patterns.

---

## Progress tracker

Legend: ✅ done · 🔄 in progress · ⬜ pending · ⏭ skipped

| # | Item | Status |
|---|---|---|
| 1 | Bulk dedupe lookups in `processBatch` | ✅ |
| 2 | Dedup block backfill SELECTs | ✅ |
| 3 | Aggregate `wallet_edges` upserts | ✅ |
| 4 | Adaptive inter-batch sleep | ✅ |
| 5 | Prefetch next batch (parallel fetch/process) | ✅ |
| 6 | `indexerHealthJob` SQL COUNT(*) | ✅ |
| 7 | `indexerEventsCleanup` bulk DELETE | ✅ |
| 8 | Drop `batch_start` events / buffer event writes | ✅ |
| 9 | `RunTokenAnalytics` worker pool + single tx | ✅ |
| 10 | Collapse `transfers` scans in snapshot | ✅ |
| 11 | Numeric columns for amount/fees | ✅ |
| 12 | Agent leaderboard SQL sort | ✅ |
| 13 | HTTP cache headers on hot endpoints | ✅ |
| 14 | `walletHandler` errgroup concurrency | ✅ |
| 15 | `searchHandler` broader coverage | ⏭ product, not perf |
| 16 | SQLite tuning audit | ✅ |
| 17 | Indexer graceful shutdown via ctx | ✅ |
| 18 | `MustCollection` → error-returning | ✅ |
| 19 | Token cache eviction | ⏭ defer (small bound today) |
| 20 | DB connection pool tuning | ⏭ (see #16) |
| R1 | Split `handlers.go` into `handlers/*.go` | ⬜ |
| R2 | Split `collections.go` into `collections/*.go` | ⬜ |
| R3 | Move `utils/tokens.go` → `internal/rpc/erc.go` | ⬜ |
| R4 | Move `utils/config.go` → `internal/chain/arc.go` | ⬜ |
| R5 | Introduce `internal/repo/` | ⬜ |
| R6 | Extract indexer aggregator | ⬜ |

### SQLite snapshot (probed 2026-05-15 on `pb_data/data.db`)

| PRAGMA | Value | Comment |
|---|---|---|
| `journal_mode` | `wal` | ✓ |
| `synchronous` | `2` (FULL) | could be `NORMAL` (1) for write throughput; WAL+NORMAL still crash-safe |
| `busy_timeout` | `0` | per-connection; verify pb-ext writer sets it |
| `cache_size` | `-2000` (2 MB) | low for hot indexer writer; per-connection |
| `mmap_size` | `0` | disabled; consider 256 MB on the read side |
| `temp_store` | `0` | default; could be `2` (memory) for transient sorts |
| `wal_autocheckpoint` | `1000` | default; fine |

These read out from a fresh `?mode=ro` connection, so they reflect SQLite defaults rather than what pb-ext applies on the live writer. Re-verify by attaching to the live process if perf becomes an issue.

Row counts at audit: blocks 8.8k, transactions 200.7k, transfers 204.3k, wallet_edges 55.5k, agents 381, agent_jobs 383, indexer_events 1.5k, token_analytics 3.1k, traces 0 (collection populated by query but no rows yet), fx_swaps 0. Aux DB (`auxiliary.db`) holds only `_logs` — routing `indexer_events` there is viable.

---

## 🔥 Hot-path performance (the indexer)

### 1. N+1 SELECT before every INSERT inside `processBatch`
Every save fn does a "find existing then insert" — `saveBlock`, `saveTransaction`, `saveTransfer`, `saveCrosschain`, `saveAgentRegistration`, `saveAgentJobCreated`, `fxUpsert`, `agentJobUpsert`. For a 200-block batch with thousands of txs/logs this is the dominant cost.

**Fix**: at the top of `processBatch` (batch.go:57), do *bulk dedupe lookups* once per collection:

```go
// gather all keys we're about to insert
txHashes := collectTxHashes(res)
blockNums := collectBlockNums(res)
// one query each
seenTx := loadSeenSet(txApp, "transactions", "hash", txHashes)
seenBlock := loadSeenSet(txApp, "blocks", "number", blockNums)
// pass into save* fns; skip if in set
```

Or use raw `INSERT OR IGNORE` via `txApp.DB().NewQuery(...)`. PocketBase's unique indexes (`idx_tx_hash`, `idx_transfers_unique`, `idx_blocks_number`, `idx_crosschain_unique`, `idx_fx_trade_id`) already enforce uniqueness — the per-row SELECT is just paranoia. Net effect: should drop indexer per-batch time by 5–10×.

### 2. Double work on blocks in `batch.go`
After inserting blocks, batch.go:182 re-queries `blocks WHERE number = {:n}` for every block to backfill `tx_count` and `block_time_ms`. Same with `block_stats` lookup at line 196. Both can use a map keyed by block number, populated as you insert.

### 3. `wallet_edges` upsert is per-transfer
`saveTransfer` → `upsertWalletEdge` (save_transfer.go:90) does a SELECT+SAVE for every stablecoin Transfer log. You already have the `agentDeltas` aggregator pattern in batch.go — apply the same to edges:

```go
type edgeKey struct{ from, to string }
edgeDeltas := map[edgeKey]*edgeDelta{}
// inside log loop: accumulate
// after log loop: one SELECT for existing edges, then bulk update/insert
```

### 4. The 400 ms inter-batch sleep is fixed, not adaptive
`runner.go:198`. When lag is high you should sprint; when caught up, sleep proportionally to expected time-to-next-block (~380 ms Arc). Trivial: `if lag > 200 { /* no sleep */ } else { time.Sleep(...) }`.

### 5. Indexer is serial fetch → serial process
Current loop: fetch batch N → process → fetch batch N+1. With HyperSync over WAN you waste ~half your wall-time on RTT. Prefetch the next batch into a buffered channel while processing the current. One worker goroutine prefetching, main loop consuming.

---

## ⚠️ Jobs & background tasks

### 6. `indexerHealthJob` loads entire tables to count
`indexer_health.go:36`: `FindRecordsByFilter(name, "", "", 0, 0)` loads *all* records to call `len()`. For `transactions`/`transfers` after a week of indexing this OOMs.

**Fix**:
```go
var cnt struct{ N int `db:"n"` }
app.DB().NewQuery("SELECT COUNT(*) AS n FROM " + name).One(&cnt)
```

### 7. `indexerEventsCleanup` deletes one-by-one
`indexer_events_cleanup.go:38`: loop calling `app.Delete(r)` per record. With heartbeats every 15s + batch_start/batch_done every 400 ms, that's ~7k events/hr to delete each hour.

**Fix**: `app.DB().NewQuery("DELETE FROM indexer_events WHERE timestamp < {:c}").Bind(...).Execute()`.

### 8. `recordIndexerEvent` is synchronous inside the indexer loop
Every batch_start/batch_done writes a row on the same DB writer that's processing the batch. Consider:
- Drop `batch_start` events entirely (you already have `batch_done` with same info)
- Buffer events in a channel, flushed by a single writer goroutine

### 9. `RunTokenAnalytics` is single-threaded RPC + N saves
`token_analytics.go`: serial RPC calls. The sleep-every-10-calls throttle is fine, but the per-token SELECT+SAVE (`FindRecordsByFilter` then `app.Save`) is again N+1. Aggregate `token_analytics` updates via a single transaction; better, drive RPC enrichment from a worker pool (4–8 concurrent).

### 10. `takeAnalyticsSnapshot` re-scans `transfers` 4×
analytics_snapshot.go runs 4 separate SELECTs over `transfers WHERE block_number >= {:from}`: count/sum, largest, group-by-symbol, distinct senders/receivers. Collapse into one query with `GROUP BY token_symbol` returning all aggregates, plus a separate ORDER BY for largest. Saves three table scans every 5 min × 3 windows = 9 scans → 3.

### 11. `largest_transfer` uses `ORDER BY CAST(amount_human AS REAL)`
analytics_snapshot.go:71. Full sort over the window with no usable index. Store an indexed numeric column (`amount_num REAL`) alongside `amount_human` so `ORDER BY amount_num DESC LIMIT 1` is index-backed. Same applies to `total_fee_usdc`, `avg_fee_usdc` in `block_stats` — currently text-cast in `LoadFeeColumn` and the snapshot queries.

---

## 🌐 Handler perf

### 12. `analyticsAgentLeaderboardHandler` does in-memory sort
handlers.go:946 sorts by parsing `usdc_transferred_human` (a string) with `fmt.Sprintf("%v", ...)`. With 500 agents that's 500 sprintf+ParseFloat calls per request. Either:
- Store an indexed numeric `usdc_transferred_num` column, ORDER BY in SQL
- Or at minimum pull the `*big.Int` directly from the record without sprintf round-tripping

### 13. No HTTP caching on hot endpoints
`/stats`, `/health`, `/analytics/overview` get hammered by the frontend. Add `Cache-Control: public, max-age=2` (live stats) and `max-age=30` (snapshot-backed). Cuts DB load 10–50× depending on poll rate.

### 14. `walletHandler` runs 7 sequential queries
handlers.go:223–235. Fire them concurrently with goroutines + a small `errgroup`. Tail-latency win of ~5×.

### 15. `searchHandler` doesn't search by token symbol or partial address
Only exact tx/addr/block. Probably acceptable — flagging for product, not perf.

---

## 🏗️ Scaling concerns

### 16. SQLite single-writer contention
You have **one writer**: indexer batches, analyticsSnapshot, token_analytics, indexer_events writes, cleanup, health, plus user API writes if any. PRAGMA-level concerns:
- Confirm WAL mode is on (PocketBase default — verify)
- `auxiliary.db` exists in `pb_data/` — figure out what's in it. If it's pb-ext logs, good. If not, route `indexer_events` there to keep the hot writer thread free.

### 17. Indexer has no graceful shutdown
`runner.go` uses `context.Background()`. On SIGTERM, in-flight batch is abandoned mid-transaction. Pass a context from `StartIndexer` derived from `srv.App()` lifecycle; honor `OnTerminate`.

### 18. `MustCollection` panics inside the indexer goroutine
`utils/convert.go:47`. A missing collection at runtime kills the process. Since collections are registered at startup, this is unlikely — but if it ever fires, the supervisor in `StartIndexer` swallows it via the `recover` in pb-ext (if any). Convert to error-returning.

### 19. Token info cache (`tokenInfoCache`) has no eviction
`utils/tokens.go:33`. Bounded only by # of unique tokens on Arc — fine today, watch it later.

### 20. No connection pool tuning visible
PocketBase manages the DB pool. For high write throughput, look at exposing `_journal_mode=WAL`, `_busy_timeout=5000`, `_synchronous=NORMAL` on the DSN. Probably already set by pb-ext defaults — worth confirming.

---

## 🧹 Reorganization (collections / handlers split)

Both `handlers.go` (960 lines) and `collections.go` (555 lines) are doing too much. The proposed layout:

```
internal/server/
├── routes.go                    # unchanged: route registration only
├── handlers/
│   ├── common.go                # qp, limitOffset, recordsToMaps, enrich*
│   ├── chain.go                 # blocks, transactions, traces, tx_detail, block_detail, search
│   ├── tokens.go                # transfers, tokens, token_detail
│   ├── wallets.go               # wallet, edges
│   ├── agents.go                # agents, agent, jobs, leaderboard
│   ├── crosschain.go            # crosschain, fx
│   ├── analytics.go             # overview, fees, volume, bridge_flow, history
│   └── stats.go                 # stats, block_stats, health
└── collections/
    ├── register.go              # RegisterCollections + helpers
    ├── meta.go                  # indexer_meta, indexer_events
    ├── chain.go                 # blocks, transactions, traces
    ├── transfers.go             # transfers, token_analytics
    ├── crosschain.go            # crosschain_events, fx_swaps
    ├── agents.go                # agents, agent_jobs
    ├── stats.go                 # block_stats, analytics_snapshots
    └── graph.go                 # wallet_edges
```

Other moves worth considering:

- **`utils/tokens.go` → `internal/rpc/erc.go`** — it's a full JSON-RPC client (ethCall, ABI decode, ERC-165 detection). Not "utils".
- **`utils/config.go` → `internal/chain/arc.go`** — chain config (addresses, topics, RPC pool) belongs to a chain package, not utils.
- **Introduce `internal/repo/`** — wrap every `FindRecordsByFilter` call behind a typed function (`repo.Transfers.ByBlock(n int) ([]*Transfer, error)`). Handlers stop knowing about PB internals; you can swap collection name without grepping the world; you can add caching at the repo layer for free.
- **Indexer batch flusher**: extract the in-memory accumulators (perBlock, agentDeltas, future edgeDeltas) into `indexer/aggregator.go`. Keep `batch.go` as the orchestrator.

---

## TL;DR — shipping order

1. Bulk dedupe lookups in `processBatch` (#1) — biggest win, ~5–10× indexer speedup
2. Fix `indexerHealthJob` count query (#6) — currently a ticking OOM
3. Fix `indexerEventsCleanup` to use bulk DELETE (#7)
4. Aggregate `wallet_edges` updates (#3)
5. Reorganize `handlers.go` and `collections.go` per the split above
6. Then: numeric columns for amount/fees (#11), HTTP cache headers (#13), prefetch the next batch (#5)
