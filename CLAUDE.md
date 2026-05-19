# Arcadia — Backend Guide

Real-time blockchain indexer and analytics API for Arc L1. Go backend only; frontend (SvelteKit in `frontend/`) needs no guidance here.

## Build & test

```bash
pb-cli              # build frontend + start dev server
pb-cli --run-only   # start server without rebuilding frontend
pb-cli --test-only  # run Go test suite with coverage
go build ./...      # compile check without running
go test ./...       # run all Go tests
```

## Package layout

```
cmd/server/main.go          entrypoint: PocketBase bootstrap, PRAGMA tuning, wiring

internal/
  chain/arc/                Arc Testnet chain package
    arc.go                  chain ID, RPC pool, contract addresses, event topics
    erc.go                  ERC-20/721/1155 token metadata (RPC detection + FIFO cache)

  indexer/                  hot path — optimise for batch throughput
    config.go               ALL tunable constants live here (batch size, pacing, timeouts)
    indexer.go              StartIndexer: retry loop, graceful shutdown via OnTerminate
    runner.go               prefetch/produce/consume loop, adaptive pacing
    batch.go                processBatch: orchestrates the full save pipeline per batch
    seen.go                 loadBatchSeen: bulk dedupe pre-fetch (one query per collection)
    query.go                HyperSync query builder (topic + address filters)
    aggregator.go           blockAcc, agentDelta, edgeDelta in-memory accumulators
    save_block.go           saveBlock, saveTransaction
    save_transfer.go        saveTransfer (calls arc.LookupTokenInfo for token classification)
    save_agent.go           agent registration + ERC-8183 job lifecycle
    save_crosschain.go      CCTP + Gateway events
    save_fx.go              StableFX trade lifecycle
    save_trace.go           routeLog dispatcher + saveTrace

  repo/                     all DB reads go through here — never call FindRecordsByFilter directly
    repo.go                 FindRecords, LatestRecord, RecordMaps, RowCount, CountWithFilter
    *.go                    one file per domain (blocks, transactions, transfers, agents, …)

  server/
    server.go               thin shim wiring handlers and collections
    cache/cache.go          in-memory TTL cache populated by the broadcaster
    realtime/
      broadcaster.go        BroadcastIndexerUpdate / HealthUpdate / AnalyticsUpdate
      notify.go             PocketBase subscription topic helpers
    handlers/               REST API handlers — read-only, always go through repo/
    collections/            PocketBase collection schema definitions

  jobs/                     background cron jobs (analytics snapshot, token enrichment, cleanup)
  utils/                    shared helpers (WeiToUSDC, StablecoinHuman, meta read/write)
```

## Key conventions

**Chain config lives in `internal/chain/arc/`.**
All Arc-specific constants (addresses, topics, RPC endpoints) are in `arc.go`. Token metadata resolution (ERC classification + cache) is in `erc.go`. Both files are in `package arc`. To add a second chain, create `internal/chain/<name>/` as its own package.

**Indexer tuning: edit `indexer/config.go` only.**
Every hardcoded number in the indexer (batch size, pacing thresholds, retry settings, heartbeat interval, etc.) is a named constant in `config.go` with a comment explaining the tradeoff. Don't scatter magic numbers across other files.

**All DB reads go through `internal/repo/`.**
Handlers and jobs call typed repo functions (`repo.ListTransfers`, `repo.AgentByAddress`, …). They never call `app.FindRecordsByFilter` directly. The repo layer owns collection names, filter syntax, and query construction.

**The indexer write path bypasses repo.**
`save_*.go` functions use PocketBase's record API directly inside `app.RunInTransaction`. They check `seen.go` in-memory sets for deduplication instead of per-row SELECTs. This is intentional for throughput — don't route indexer writes through repo.

**Prefetch pattern in `runner.go`.**
Batch N+1 is fetched concurrently while batch N is being written to SQLite. The prefetch goroutine is started before `processBatch` returns. Don't break this overlap.

**Token metadata flow.**
`arc.LookupTokenInfo(app, addr, firstSeenBlock)` — resolution order: in-memory FIFO cache → `token_analytics` collection → live `eth_call`. Result is always persisted to `token_analytics`. Known stablecoins are seeded at startup by `arc.SeedKnownTokens()` and never evicted from cache.

**High-precision amounts.**
`uint256` values (transfer amounts, fees) are stored as `TEXT` in SQLite. A numeric mirror column (e.g. `amount_num`, `usdc_transferred_num`) exists alongside for SQL `ORDER BY` and range filters. Always set both when writing.

**No comments on obvious code.** Only add a comment when the WHY is non-obvious: a hidden constraint, a workaround, a subtle invariant. Don't describe what the code does.

## SQLite notes

PRAGMAs applied at startup in `cmd/server/main.go`: WAL mode, `synchronous=NORMAL`, `busy_timeout=5000`, `cache_size=-8000`, `temp_store=2`, `mmap_size=268435456`. Don't change these without understanding the WAL checkpoint behaviour in `indexer.go` (`maybeTruncateWAL`).

Write transactions come from the indexer loop. API reads and job queries run concurrently under WAL. Keep write transactions short — that's why `batchSize` defaults to 50 blocks.

## Adding a new chain

1. Create `internal/chain/<name>/` with `package <name>`
2. Define chain constants, contract addresses, and event topics (mirror `arc/arc.go`)
3. Add ERC token detection if needed (mirror `arc/erc.go`)
4. Wire a new indexer entry point that references the new chain package
