# arcadia

A real-time blockchain indexer and analytics platform for Arc L1. Indexes every layer of Arc's onchain activity — blocks, transactions, stablecoin transfers, internal traces, AI agent lifecycle events, cross-chain flows, FX swaps, and derived economic metrics — then serves it through a versioned REST API with auto-generated OpenAPI docs and a live SvelteKit dashboard.

> **Demo**: The live instance runs on the Envio HyperSync free tier. For production throughput, self-host with your own API key (see Get Started below).

---

## Use Cases

### Trading agents
Feed live inflow/outflow and bridge flow direction into agent decision loops. Cross-chain directional USDC volume (CCTP + Gateway) is available in real time — useful for detecting capital entering or leaving Arc ahead of price movements. Combine with the agent leaderboard to track which AI agents are accumulating fees or volume.

### Quant analytics
Pre-aggregated snapshots store a full time-series of transfer volume, fee percentiles (p25/p50/p75/p95), block time, whale transfer count, and active address metrics at 5-minute resolution across 1h / 24h / 7d windows. Pull rolling windows for volatility modelling, regime detection, or autocorrelation analysis on stablecoin flows.

### Whale tracking · copy trading
Transfers above $10K are flagged as whale events and counted in every snapshot window. The wallet endpoint returns complete send/receive history and graph edges per address. Combine with the wallet graph (`wallet_edges`) to map capital flows between large wallets, identify lead actors, and build copy-trading signal pipelines.

### Agent economy monitoring
Arc has native onchain AI agent identity (ERC-8004) and a job escrow standard (ERC-8183). Arcadia indexes every agent registration, job lifecycle event (created → funded → completed/rejected → paid), and agent-to-agent capital flow. Track agent growth rate, settlement ratio, and top earners.

### FX and stablecoin research
StableFX settles USDC↔EURC swaps onchain. Every trade is indexed with implied rate, maker/taker, and settlement status. Cross-chain USDC mint/burn events via CCTP and Gateway give a full picture of stablecoin supply dynamics. Useful for FX basis research, arbitrage signal generation, and stablecoin peg health monitoring.

---

## Get Started

### Prerequisites

- **Go** 1.25+
- **Node.js** 16+ (for frontend builds)
- **npm** 8+
- **Envio API token** (get one at https://envio.dev)

### Clone and configure

```bash
git clone https://github.com/magooney-loon/arcadia.git
cd arcadia
```

Copy the env file and set your token:

```bash
cp .env.example .env
# Edit .env and set ENVIO_API_TOKEN=your_token_here
```

### pb-cli

Install the build toolchain:

```bash
go install github.com/magooney-loon/pb-ext/cmd/pb-cli@latest
```

> **Note**: Ensure `$HOME/go/bin` is in your `$PATH`.

| Command | What it does |
|---|---|
| `pb-cli` | Build frontend + start dev server |
| `pb-cli --install` | Install all dependencies (Go modules + npm) |
| `pb-cli --build-only` | Build frontend only (into `pb_public/`) |
| `pb-cli --run-only` | Start server without rebuilding frontend |
| `pb-cli --production` | Optimized production build into `dist/` |
| `pb-cli --production --dist release` | Production build with custom output dir |
| `pb-cli --test-only` | Run test suite with coverage |

Dev mode builds the SvelteKit frontend, copies it to `pb_public/`, and starts the server:

```bash
pb-cli
```

### Profiling the indexer

Capture server output to a log file while running, then summarize batch phase timings:

```bash
mkdir -p logs
pb-cli 2>&1 | tee logs/profile.log
```

Once you have a log (or while it's running and you've captured enough), run the profiler:

```bash
./scripts/profile_batches.sh logs/profile.log
```

You can also pipe a live log directly:

```bash
journalctl -u arcadia | ./scripts/profile_batches.sh -
```

The script parses `batch_profile` lines and prints count, avg, p50, p95, and max per phase (e.g. `seen_ms`, `tx_total_ms`, `blocks_ms`, `txs_ms`, `traces_ms`, `backfill_ms`, etc.).

### Verify it's running

- **App**: http://127.0.0.1:8090
- **pb-ext dashboard**: http://127.0.0.1:8090/_/_
- **PocketBase admin**: http://127.0.0.1:8090/_/
- **OpenAPI docs**: http://127.0.0.1:8090/api/docs/v1/swagger

---

## Architecture

```
                    ┌─────────────┐
                    │  HyperSync  │  Arc L1 node (columnar streaming)
                    └──────┬──────┘
                           │ Arrow batches
                           ▼
┌──────────────────────────────────────────────────────────────┐
│                       Indexer Pipeline                        │
│  runner.go ──► prefetch goroutine ──► processBatch()          │
│       │              │                     │                   │
│       │         (parallel fetch)     bulk dedupe + save       │
│       │              │              aggregator + flush        │
│       ▼              ▼                     ▼                   │
│  adaptive pacing   channel           seen.go (dedupe maps)   │
│  ctx-based shutdown                 save_block/tx/transfer…  │
└──────────────────────────┬───────────────────────────────────┘
                           │ writes
                           ▼
                ┌─────────────────────┐
                │   SQLite (WAL mode)  │  PocketBase v0.38
                │   14 collections     │  PRAGMA-tuned at startup
                └──────────┬──────────┘
                           │ reads
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │  Repo     │ │  Jobs     │ │ Handlers  │
        │  layer    │ │  layer    │ │  (REST)   │
        └──────────┘ └──────────┘ └──────────┘
              ▲            │            │
              └────────────┴────────────┘
                   all reads go
                   through repo/
                           │
                           ▼
                    ┌─────────────┐
                    │  SvelteKit   │  Frontend dashboard
                    │  (pb_public) │  3D graph + charts
                    └─────────────┘
```

**Data flow**: HyperSync streams Arrow batches → indexer pipeline deduplicates, classifies, and persists to SQLite → repo layer wraps all reads → handlers and jobs consume through repo → REST API serves the frontend.

---

## Project Structure

```
arcadia/
├── cmd/server/main.go          # Entrypoint: PocketBase app bootstrap, PRAGMA tuning, wiring
│
├── internal/
│   ├── chain/arc/              # Arc Testnet chain package (future chains live as chain/<name>/)
│   │   ├── arc.go              # Network config: chain ID, RPC pool, addresses, event topics
│   │   └── erc.go              # ERC token metadata: RPC detection (ERC-20/721/1155), FIFO cache
│   │
│   ├── indexer/                # Blockchain data indexing engine
│   │   ├── config.go           # Tunable constants: batch size, pacing, timeouts, retry settings
│   │   ├── indexer.go          # StartIndexer: retry loop, ctx + OnTerminate shutdown
│   │   ├── runner.go           # runIndexer: prefetch/produce/consume loop, adaptive pacing
│   │   ├── query.go            # HyperSync query builder: topic/address selection per log type
│   │   ├── batch.go            # processBatch: orchestrates the full save pipeline per batch
│   │   ├── aggregator.go       # blockAcc, agentDelta accumulators + flushEdgeDeltas
│   │   ├── seen.go             # loadBatchSeen: bulk dedupe pre-fetch, loadEdgesFor, edgeKey types
│   │   ├── save_block.go       # saveBlock, saveTransaction
│   │   ├── save_transfer.go    # saveTransfer (ERC-20/721/1155 with token metadata lookup)
│   │   ├── save_agent.go       # saveAgentRegistration, agentJobUpsert, job lifecycle handlers
│   │   ├── save_crosschain.go  # CCTP burn/mint + Gateway deposit/withdraw events
│   │   ├── save_fx.go          # StableFX trade lifecycle (recorded → funded → settled)
│   │   └── save_trace.go       # routeLog: dispatches log to the correct save_* by topic
│   │
│   ├── repo/                   # Database read layer (typed query functions)
│   │   ├── repo.go             # Core helpers: FindRecords, LatestRecord, RecordMaps, RowCount
│   │   ├── blocks.go           # ListBlocks, BlockByNumber
│   │   ├── transactions.go     # ListTransactions (dynamic filter), ByHash, ByBlock, BySender/Receiver
│   │   ├── transfers.go        # ListTransfers (dynamic filter), ByToken, ByTxHash, BySender/Receiver
│   │   ├── traces.go           # ListTraces (dynamic filter), ByTxHash
│   │   ├── agents.go           # ListAgents, AgentByAddress, AgentLeaderboard, AgentJobStats (raw SQL)
│   │   ├── jobs.go             # ListJobs (dynamic filter), JobsByAddress
│   │   ├── tokens.go           # ListTokens (search), TokenByAddress, AllTokenAnalytics
│   │   ├── wallet_edges.go     # EdgesByFrom/ToWallet, EdgesByWallet (bidirectional)
│   │   ├── crosschain.go       # ListCrosschainEvents (protocol, direction filters)
│   │   ├── fx_swaps.go         # ListFxSwaps (status, maker, taker filters)
│   │   ├── stats.go            # LatestBlockStats, RecentBlockStats, BlockStatsByNumber
│   │   ├── analytics.go        # LatestSnapshot, SnapshotHistory
│   │   ├── meta.go             # MetaValue, AllMeta (indexer_meta key/value reads)
│   │   └── events.go           # ErrorEventsSince, RecentBatchDones, DeleteEventsBefore
│   │
│   ├── server/                 # HTTP layer
│   │   ├── server.go           # Thin shim: delegates to handlers/ and collections/
│   │   ├── cache/              # In-memory response cache (TTL-based, populated by broadcaster)
│   │   │   └── cache.go
│   │   ├── realtime/           # SSE broadcaster
│   │   │   ├── broadcaster.go  # BroadcastIndexerUpdate/HealthUpdate/AnalyticsUpdate + throttle
│   │   │   └── notify.go       # PocketBase subscription topic helpers
│   │   ├── handlers/           # API route handlers (read-only, all go through repo/)
│   │   │   ├── routes.go       # RegisterRoutes, versioned v1 API registration, OpenAPI config
│   │   │   ├── common.go       # Shared helpers: qp, limitOffset, cacheHeaders, enrichRecord fns
│   │   │   ├── chain.go        # blocks, transactions, traces, search, tx/block detail
│   │   │   ├── tokens.go       # transfers, token list, token detail
│   │   │   ├── wallets.go      # wallet profile (7 concurrent queries), edges
│   │   │   ├── agents.go       # agents, agent detail, jobs, leaderboard
│   │   │   ├── crosschain.go   # crosschain events, FX swaps
│   │   │   ├── analytics.go    # overview, fees, volume, bridge flow, history
│   │   │   └── stats.go        # stats, block stats, health
│   │   └── collections/        # PocketBase collection schema definitions
│   │       ├── register.go     # RegisterCollections + collectionExists helper
│   │       ├── meta.go         # indexer_meta, indexer_events
│   │       ├── chain.go        # blocks, transactions, traces
│   │       ├── transfers.go    # transfers, token_analytics
│   │       ├── crosschain.go   # crosschain_events, fx_swaps
│   │       ├── agents.go       # agents, agent_jobs
│   │       ├── stats.go        # block_stats, analytics_snapshots
│   │       └── graph.go        # wallet_edges
│   │
│   ├── jobs/                   # Background scheduled jobs
│   │   ├── jobs.go             # RegisterJobs: wires all cron jobs
│   │   ├── analytics_snapshot.go  # Every 5 min: pre-aggregates 1h/24h/7d snapshot
│   │   ├── token_analytics.go  # Every 30 min: RPC enrichment, transfer counts per token
│   │   ├── indexer_health.go   # Every hour: logs row counts + indexer cursor
│   │   └── indexer_events_cleanup.go  # Every hour: deletes events older than 2h
│   │
│   └── utils/                  # Shared utilities
│       ├── convert.go          # WeiToUSDC, StablecoinHuman, TokenAmountHuman, FindCollection, address extraction
│       ├── analytics.go        # WindowBlockFilter, LoadFeeColumn, PercentileFloat, DomainName
│       └── meta.go             # GetLastIndexedBlock, SetLastIndexedBlock, SetMetaValue
│
├── frontend/                   # SvelteKit dashboard (separate build → pb_public/)
└── docs/                       # Arc network reference, HyperSync docs, IMPROVE tracker
```

---

## Indexer Pipeline

The indexer is the hot path. Every design decision optimizes for batch throughput.

### Fetch → Process loop (`runner.go`)

The main loop runs continuously:
1. **Prefetch goroutine** fetches the next HyperSync batch in parallel while the current batch is being processed — eliminates RTT waste
2. **`processBatch`** (`batch.go`) receives the prefetched result and runs the full save pipeline inside a single PocketBase transaction
3. **Adaptive pacing** — when the indexer is behind the chain tip, it sprints (no sleep). When caught up, it sleeps proportionally to expected block time (~380ms for Arc)
4. **Graceful shutdown** — a context derived from PocketBase's `OnTerminate` hook propagates through every blocking point (prefetch, sleep, HyperSync client creation)

### Batch processing (`batch.go`)

Each batch covers ~200 blocks and processes in this order:

1. **Bulk dedupe** (`seen.go` → `loadBatchSeen`) — one SQL query per collection to get all existing block numbers, tx hashes, transfer keys, crosschain keys, and agent addresses within the batch range. The save functions check these in-memory sets instead of doing per-row SELECTs
2. **Save blocks** — creates block records, computes utilization %
3. **Save transactions** — creates tx records with fee calculations, detects contract deploys
4. **Route logs** (`save_trace.go` → `routeLog`) — inspects `topic0` to dispatch each log to the correct handler:
   - `Transfer` → token transfer (with ERC-20/721/1155 classification via RPC)
   - `DepositForBurn` / `MintAndWithdraw` / `MessageReceived` → CCTP cross-chain events
   - `GatewayDeposited` / `GatewayBurned` / `AttestationUsed` → Gateway cross-chain events
   - `AgentRegistered` → agent registration
   - `JobCreated` → agent job creation
   - `TradeRecorded` / `MakerFunded` / `TakerFunded` / `TradeStatusChanged` / `FeesProcessed` → FX swaps
5. **Save traces** — internal transaction traces (CALL, DELEGATECALL, etc.)
6. **Accumulate** — in-memory aggregators (`blockAcc`, `agentDelta`, `edgeDelta`) collect per-block and per-entity stats during the save loop
7. **Flush** — backfill derived fields on block records, insert `block_stats` rows, update agent counters, and flush wallet edge deltas — all in the same transaction

### Token metadata (`chain/arc/erc.go`)

When a transfer log hits an unknown token address, `LookupTokenInfo` resolves it:
1. In-memory cache (FIFO-evicted, 5K max, seeded stablecoins never evicted)
2. `token_analytics` collection (previously persisted metadata)
3. Live JSON-RPC `eth_call` to detect ERC-20 vs ERC-721 vs ERC-1155, then fetch symbol/name/decimals

The resolved metadata is persisted to `token_analytics` so future batches skip the RPC call entirely.

---

## Data Model

14 PocketBase collections organized in layers:

| Layer | Collections | Source |
|---|---|---|
| **Meta** | `indexer_meta`, `indexer_events` | Internal cursor and event log |
| **Chain** | `blocks`, `transactions`, `traces` | HyperSync blocks/transactions/traces |
| **Transfers** | `transfers`, `token_analytics` | ERC-20/721/1155 Transfer events + token metadata |
| **Agents** | `agents`, `agent_jobs` | ERC-8004 registrations + ERC-8183 job lifecycle |
| **Cross-chain** | `crosschain_events`, `fx_swaps` | CCTP + Gateway + StableFX events |
| **Derived** | `block_stats`, `analytics_snapshots` | Per-block aggregates + windowed snapshots |
| **Graph** | `wallet_edges` | Aggregated wallet-to-wallet transfer edges |

Each collection has unique indexes for deduplication and covering indexes for the common query patterns (block number ranges, address lookups, token symbol filters). High-precision values (uint256 amounts, fees) are stored as text with a numeric mirror column for indexed range queries and sorting.

---

## Repo Layer

All database reads go through `internal/repo/`. Handlers and jobs never call PocketBase's `FindRecordsByFilter` directly — they use typed functions that encapsulate collection names, filter syntax, and query construction.

Key patterns:
- **`FindRecords(app, collection, filter, sort, limit, offset, params)`** — core helper that wraps `FindRecordsByFilter` with error wrapping
- **`LatestRecord(app, collection, filter, sort, params)`** — returns a single record or nil
- **Dynamic filter structs** — `TransactionFilter`, `TransferFilter`, `JobFilter`, etc. build the filter string from optional fields, so handlers just populate a struct from query params
- **`RowCount(app, table)`** — `SELECT COUNT(*)` without loading records
- **Complex aggregations** — `AgentJobStats` wraps a raw SQL UNION query for the leaderboard; analytics snapshots use raw SQL for their aggregation queries since they're not simple CRUD patterns

The indexer write path (`save_*` functions) does **not** go through the repo layer — it uses PocketBase's record API directly inside transactions for maximum throughput, with bulk dedupe via `seen.go`.

---

## Background Jobs

| Job | Schedule | What it does |
|---|---|---|
| `analyticsSnapshot` | Every 5 min | Aggregates transfers, fees, bridge flows, agent counts into `analytics_snapshots` for 1h / 24h / 7d windows |
| `tokenAnalytics` | Every 10 min | 6-worker pool enriches token metadata via RPC, counts transfers/holders per token |
| `indexerHealth` | Every hour | Logs row counts for all collections and the current indexer cursor |
| `indexerEventsCleanup` | Every hour | Deletes `indexer_events` records older than 2 hours |

---

## API

Versioned REST API under `/api/v1` with auto-generated OpenAPI 3.0 spec (public Swagger UI). Default pagination: `limit=50&offset=0`, max `limit=500`.

Hot endpoints have `Cache-Control` headers: live data gets `max-age=2`, snapshot-backed analytics get `max-age=30`, leaderboard gets `max-age=60`.

See the Swagger UI at `/api/v1/swagger` for the full endpoint reference, request/response schemas, and query parameter docs.

### Realtime (SSE)

Custom PocketBase subscriptions broadcast pre-computed payloads to connected clients via SSE, replacing HTTP polling. Three topics cover the entire dashboard:

| Topic | Trigger | Payload | Who subscribes |
|---|---|---|---|
| `indexer` | After each indexer batch commit (~1Hz throttle) | `{stats, health, blocks[], transactions[]}` | Every tab (subscribed in `+layout.svelte`) |
| `charts` | After each indexer batch commit (~1Hz throttle) | `{block_stats[]}` (50 rows) | Only tabs rendering the overview charts |
| `analytics` | After each analytics snapshot job (every 5 min) | `{window, overview, bridge_flow, volume}` | Every tab |

`internal/server/realtime/broadcaster.go` builds and fans out these payloads. `BroadcastIndexerUpdate` is called after each batch commit and throttles more aggressively when the indexer is catching up (10s interval at lag > 100 blocks, 5s at lag > 20, 1s otherwise) to avoid competing with the write transaction for SQLite read bandwidth. The same function also populates the in-memory REST cache so API handlers can serve responses without hitting SQLite between broadcasts.

The frontend subscribes in `+layout.svelte` via `connectRealtime()` (`src/lib/realtime.ts`), which uses the PocketBase JS SDK's `pb.realtime.subscribe()`. The SDK manages the SSE connection lifecycle — auto-reconnect, clientId handshake, re-subscription on reconnect. Chart subscriptions are view-bound and managed separately via `connectCharts()` / `disconnectCharts()`.

---

## SQLite Tuning

PRAGMAs are applied at startup in `cmd/server/main.go`:

| PRAGMA | Value | Purpose |
|---|---|---|
| `journal_mode` | WAL (PocketBase default) | Concurrent reads during writes |
| `synchronous` | NORMAL | Crash-safe with WAL, better write throughput |
| `busy_timeout` | 5000 ms | Prevents SQLITE_BUSY under concurrent writer pressure |
| `cache_size` | -8000 (8 MB) | Page cache per connection |
| `temp_store` | 2 (memory) | Transient sorts/indexes in RAM |
| `mmap_size` | 256 MB | Memory-mapped reads to reduce syscall overhead |

---

## Stack

| Component | Technology |
|---|---|
| **Language** | Go 1.25 |
| **Indexer** | `github.com/enviodev/hypersync-client-go` (Arrow columnar streaming) |
| **Server** | `github.com/magooney-loon/pb-ext` (PocketBase extension: versioned API, job manager, OpenAPI) |
| **Database** | PocketBase v0.38 (SQLite WAL, PRAGMA-tuned) |
| **Chain** | Arc Testnet (Chain ID `5042002`) |
| **Frontend** | SvelteKit dashboard with 3D graph visualization |

---

## Key Arc Testnet References

- **Chain ID**: `5042002`
- **HyperSync**: `https://arc-testnet.hypersync.xyz`
- **Public RPCs** (round-robin, no key required):
  - `https://rpc.testnet.arc.network`
  - `https://rpc.blockdaemon.testnet.arc.network`
  - `https://rpc.drpc.testnet.arc.network`
  - `https://rpc.quicknode.testnet.arc.network`
- **Explorer**: `https://testnet.arcscan.app`
- **CCTP Domain**: `26`
