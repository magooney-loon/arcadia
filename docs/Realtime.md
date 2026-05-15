# Realtime Subscriptions — Implementation Plan

> Replace HTTP polling with PocketBase's built-in SSE realtime system for live dashboard updates.

---

## Current State

The frontend polls the REST API on two intervals:

| Location | Interval | Endpoints |
|---|---|---|
| `+layout.svelte` | 8 s | `/stats`, `/health` |
| `overview/+page.svelte` | 2 s | `/stats`, `/blocks?limit=10`, `/transactions?limit=10` |
| `overview/+page.svelte` | 10 s | `/block_stats`, `/analytics/overview`, `/analytics/bridge_flow`, `/analytics/volume`, `/analytics/agent_leaderboard` |

That's **9 HTTP requests every 2 seconds** per open browser tab when viewing the overview. Each request goes through PocketBase → repo → SQLite. For a dashboard that might sit open in a background tab all day, this is wasteful — the vast majority of polls return unchanged data.

---

## Design Decision: Custom Subscriptions vs. Native Record Broadcast

PocketBase v0.38 has two mechanisms for realtime:

### Option A: Native Record Broadcast (collection-level subscriptions)

PocketBase automatically broadcasts `create`/`update`/`delete` events for any collection record via the built-in `OnModelAfterCreateSuccess` / `OnModelAfterUpdateSuccess` hooks. The frontend subscribes to collection names:

```js
pb.realtime.subscribe('blocks', (e) => { ... })
pb.realtime.subscribe('transfers', (e) => { ... })
```

**Pros:**
- Zero server-side code — PocketBase handles it automatically
- Individual record-level granularity (you get the exact record that changed)
- Built-in access control via collection `viewRule`/`listRule`

**Cons:**
- Too granular for our use case — we'd fire SSE events for every single `block_stats` insert, every `analytics_snapshots` insert, etc.
- The indexer writes hundreds of records per batch; broadcasting all of them to clients would create SSE storms
- Our frontend doesn't care about individual records — it wants aggregated results ("latest stats changed", "new blocks available")
- Access rules don't matter here since this is a public read-only dashboard

### Option B: Custom Subscriptions (recommended ✅)

We send targeted, high-level messages on custom topics using `app.SubscriptionsBroker()`. The frontend subscribes to our topic names and receives pre-computed payloads — essentially the same data the REST endpoints return, but pushed only when something changes.

**Pros:**
- **Batch-level deduplication** — we send ONE event after a full indexer batch completes, not 200 individual record events
- **Pre-computed payloads** — we send the same JSON the REST endpoint would return, so the frontend just updates state directly
- **Full control over timing** — emit events at the exact right moment (after batch commit, after snapshot job, after token analytics)
- **No noise** — only the topics the frontend actually cares about
- **Simpler frontend** — replace polling intervals with a single `pb.realtime.subscribe()` call per topic

**Cons:**
- Requires new Go code (broker notification helpers)
- Must wire notifications into existing write paths

### Verdict: **Custom subscriptions**. The indexer's batch-write pattern and the frontend's aggregate-data needs make native record broadcast a poor fit. Custom topics let us send one fat message per batch instead of 200 thin ones.

---

## Subscription Topics

Based on the current polling endpoints, we define these topics:

| Topic | Trigger | Payload | Notes |
|---|---|---|---|
| `stats` | After indexer batch commits | Same as `GET /api/v1/stats` | TPS, fees, block number, indexed block |
| `health` | After indexer batch commits + heartbeat | Same as `GET /api/v1/health` | Lag, error count, batch timing |
| `blocks` | After indexer batch commits | Latest 10 blocks (same as `GET /api/v1/blocks?limit=10`) | For the "latest blocks" live feed |
| `transactions` | After indexer batch commits | Latest 10 transactions | For the "latest transactions" live feed |
| `analytics` | After `analyticsSnapshot` job completes | Same as `GET /api/v1/analytics/overview?window=24h` | Snapshot-driven, only fires every 5 min |
| `block_stats` | After indexer batch commits | Latest 200 block stats rows | For charts |
| `snapshot` | After `takeAnalyticsSnapshot` completes per window | `{window, overview, bridge_flow, volume}` | Fires every 5 min per window |

### Optimized topic design

We can collapse these into fewer, richer messages to minimize SSE overhead:

| Topic | When | Payload shape |
|---|---|---|
| `indexer` | After every batch commit | `{stats, health, latestBlocks[], latestTxs[], blockStats[]}` |
| `analytics` | After snapshot job fires | `{overview, bridgeFlow, volume, window}` |
| `token_analytics` | After token analytics job completes | `{tokens[]}` (optional, low priority) |

This gives us **2 topics for the entire dashboard** instead of 7 separate polling loops. The `indexer` topic fires at indexer pace (every ~400ms when catching up, ~2s at tip). The `analytics` topic fires every 5 minutes.

---

## Architecture

```
                    ┌─────────────┐
                    │  HyperSync  │
                    └──────┬──────┘
                           │
                           ▼
                ┌─────────────────────┐
                │   Indexer Pipeline   │
                │   processBatch()     │
                └──────┬──────────────┘
                       │ after tx commit
                       ▼
              ┌─────────────────────┐
              │  notify.go           │  ← NEW: broadcast to subscribers
              │  BroadcastIndexer()  │     uses SubscriptionsBroker
              └──────┬──────────────┘
                     │ SSE
              ┌──────┴──────┐
              ▼             ▼
        ┌──────────┐  ┌──────────┐
        │ SvelteKit │  │ SvelteKit │  (multiple tabs/browsers)
        │  tab 1    │  │  tab 2    │
        └──────────┘  └──────────┘

Also wired:
  analyticsSnapshot job ──► BroadcastAnalytics()
  tokenAnalytics job   ──► BroadcastTokenAnalytics()
```

---

## Server-Side Changes

### 1. New file: `internal/server/realtime/notify.go`

A thin notification layer that wraps `app.SubscriptionsBroker()`:

```go
package realtime

import (
    "encoding/json"
    "golang.org/x/sync/errgroup"
    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/pocketbase/tools/subscriptions"
)

// Broadcast sends a JSON payload to all clients subscribed to the given topic.
func Broadcast(app core.App, topic string, payload any) error {
    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    msg := subscriptions.Message{
        Name: topic,
        Data: data,
    }

    chunks := app.SubscriptionsBroker().ChunkedClients(300)
    group := new(errgroup.Group)

    for _, chunk := range chunks {
        group.Go(func() error {
            for _, client := range chunk {
                if !client.HasSubscription(topic) {
                    continue
                }
                client.Send(msg)
            }
            return nil
        })
    }

    return group.Wait()
}
```

### 2. New file: `internal/server/realtime/broadcaster.go`

Functions that compute the same payloads the REST handlers return, then broadcast them:

```go
// BroadcastIndexerUpdate is called after each indexer batch commits.
// Sends the "indexer" topic with {stats, health, latestBlocks, latestTxs, blockStats}.
func BroadcastIndexerUpdate(app core.App) error {
    // Reuse existing repo functions to build the payload
    // (same data that statsHandler, healthHandler, etc. return)
    ...
    return Broadcast(app, "indexer", payload)
}

// BroadcastAnalyticsUpdate is called after analytics snapshot job completes.
func BroadcastAnalyticsUpdate(app core.App, window string) error {
    ...
    return Broadcast(app, "analytics", payload)
}
```

**Key design choice**: The broadcaster calls the same `repo.*` functions the handlers use. This avoids duplicating query logic. The payload is computed once and fanned out to all SSE clients.

### 3. Wire into indexer: `internal/indexer/runner.go`

After a successful batch commit (right after `utils.SetLastIndexedBlock`), call the broadcaster:

```go
// After successful batch commit:
go realtime.BroadcastIndexerUpdate(app)  // fire-and-forget in goroutine
```

The broadcast is non-blocking — it just writes to each client's buffered channel. We run it in a goroutine so it doesn't slow down the indexer loop.

### 4. Wire into jobs: `internal/jobs/analytics_snapshot.go`

After `takeAnalyticsSnapshot` succeeds:

```go
go realtime.BroadcastAnalyticsUpdate(app, window)
```

### 5. Wire into jobs: `internal/jobs/token_analytics.go`

After `RunTokenAnalytics` completes:

```go
go realtime.BroadcastTokenAnalyticsUpdate(app)
```

---

## Frontend Changes

### 1. New file: `frontend/src/lib/realtime.ts`

A thin wrapper around PocketBase's realtime SDK:

```ts
import { pb } from '$lib/stores/config.svelte';
import { stats, fetchStats } from '$lib/stores/stats.svelte';
// ... import other stores

export async function connectRealtime() {
    // PocketBase JS SDK handles SSE reconnects automatically
    await pb.realtime.subscribe('indexer', (e) => {
        // e.record contains the full payload from BroadcastIndexerUpdate
        stats.data = e.record.stats;
        health.data = e.record.health;
        blocks.data = e.record.blocks;
        transactions.data = e.record.transactions;
        blockStats.data = e.record.blockStats;
    });

    await pb.realtime.subscribe('analytics', (e) => {
        analyticsOverview.data = e.record.overview;
        analyticsBridgeFlow.data = e.record.bridgeFlow;
        analyticsVolume.data = e.record.volume;
    });
}

export async function disconnectRealtime() {
    await pb.realtime.unsubscribe('indexer');
    await pb.realtime.unsubscribe('analytics');
}
```

### 2. Replace polling in `+layout.svelte`

```diff
 onMount(() => {
-    fetchStats();
-    fetchHealth();
-    const id = setInterval(() => {
-        fetchStats();
-        fetchHealth();
-    }, 8000);
-    return () => clearInterval(id);
+    connectRealtime();
+    return () => disconnectRealtime();
 });
```

### 3. Replace polling in `overview/+page.svelte`

```diff
 onMount(() => {
-    const refreshLive = () => { fetchStats(); fetchBlocks(10); fetchTransactions({ limit: 10 }); };
-    refreshLive();
-    refreshAnalytics();
-    const liveId = setInterval(refreshLive, 2000);
-    const analyticsId = setInterval(refreshAnalytics, 10000);
-    return () => { clearInterval(liveId); clearInterval(analyticsId); };
+    // Initial load via REST, then realtime takes over
+    refreshAnalytics();
+    // No more intervals — SSE handles updates
 });
```

### 4. Keep REST as fallback

The REST API stays as-is for:
- First page load (initial data fetch before SSE connects)
- Pages that don't need realtime (paginated list views, detail pages)
- External API consumers
- SSE reconnection gap fills

---

## Implementation Steps

### Phase 1: Server-side broker (Go)

| Step | File | Description |
|---|---|---|
| 1.1 | `internal/server/realtime/notify.go` | `Broadcast()` helper — wraps SubscriptionsBroker with errgroup |
| 1.2 | `internal/server/realtime/broadcaster.go` | `BroadcastIndexerUpdate()`, `BroadcastAnalyticsUpdate()` — reuse repo layer |
| 1.3 | `internal/indexer/runner.go` | Call `realtime.BroadcastIndexerUpdate(app)` after batch commit |
| 1.4 | `internal/jobs/analytics_snapshot.go` | Call `realtime.BroadcastAnalyticsUpdate()` after each window snapshot |
| 1.5 | Test with `pb.realtime.subscribe()` in browser console |

### Phase 2: Frontend SSE integration (TypeScript/Svelte)

| Step | File | Description |
|---|---|---|
| 2.1 | `frontend/src/lib/realtime.ts` | Subscribe to `indexer` + `analytics` topics, update stores |
| 2.2 | `frontend/src/routes/+layout.svelte` | Replace 8s polling interval with SSE connection |
| 2.3 | `frontend/src/routes/overview/+page.svelte` | Remove 2s/10s intervals; rely on SSE |
| 2.4 | Test across tabs, reconnection, background tabs |

### Phase 3: Polish & edge cases

| Step | Description |
|---|---|
| 3.1 | Add connection status indicator in the status bar |
| 3.2 | Handle SSE disconnect: fall back to polling for 30s, then retry SSE |
| 3.3 | Rate-limit broadcasts if indexer is catching up (batch every 400ms → debounce to max 1 event/sec) |
| 3.4 | Add `pb.realtime.subscribe('token_analytics', ...)` for the tokens page |
| 3.5 | Remove `Cache-Control` headers from endpoints that are now SSE-only (or keep them for external consumers) |

---

## Performance Impact

### Before (polling)

| Metric | Value |
|---|---|
| HTTP requests per tab per minute | ~270 (overview page) |
| SQLite queries per tab per minute | ~270 (each request = at least 1 query) |
| Payload per request | 1–50 KB |
| Bandwidth per tab per hour | ~50 MB |
| Server CPU per tab | Constant polling overhead |

### After (SSE)

| Metric | Value |
|---|---|
| HTTP requests per tab per minute | 0 (after initial SSE connect) |
| SSE events per tab per minute | ~30 (at tip) to ~150 (catching up) — or ~0.2/min for analytics |
| Payload per event | ~5–20 KB (batched, pre-computed) |
| Bandwidth per tab per hour | ~5–10 MB |
| Server CPU per tab | Near-zero (channel write, no query) |

The key insight: **broadcasting computes the payload once and fans out to N clients**. Polling computes the same payload N times (once per client). With 5 open tabs, that's a 5× reduction in SQLite queries.

---

## Rate Limiting Strategy

When the indexer is catching up (behind the chain tip), it processes batches every ~400ms. We don't want to broadcast 2.5 times per second. Options:

1. **Debounce** — only broadcast if the last broadcast was > 1 second ago, otherwise skip. Simple but means the frontend misses mid-debounce batches.
2. **Throttle with latest** — broadcast at most once per second, but always send the latest data. The frontend never misses the current state, just intermediate states.
3. **Conditional broadcast** — only broadcast if there are active subscribers. Check `app.SubscriptionsBroker().TotalClients() > 0` before computing the payload.

**Recommendation**: Option 2 (throttle) + Option 3 (skip if no subscribers). This gives the frontend smooth ~1Hz updates at tip and avoids computing payloads when nobody is listening.

```go
var lastBroadcast atomic.Int64

func shouldBroadcast() bool {
    now := time.Now().UnixMilli()
    last := lastBroadcast.Load()
    if now - last < 1000 {  // 1 second minimum between broadcasts
        return false
    }
    return lastBroadcast.CompareAndSwap(last, now)
}
```

---

## Reconnection & Reliability

PocketBase's JS SDK handles SSE reconnection automatically. The flow:

1. Client connects to `/api/realtime` (GET, SSE stream)
2. Server sends `PB_CONNECT` event with `clientId`
3. Client POSTs to `/api/realtime` with `{clientId, subscriptions: ["indexer", "analytics"]}`
4. Server pushes events on the SSE stream
5. If connection drops, SDK auto-reconnects and re-subscribes

**Gap handling**: Between disconnect and reconnect, the client might miss events. On reconnect, the frontend should:
1. Do a one-shot REST fetch for current state (same as initial load)
2. Resume SSE listening

This is already the natural flow since the initial REST load provides the baseline.

---

## Topic Payload Schemas

### `indexer` topic

```json
{
  "stats": {
    "block_number": 12345,
    "tps": 12.5,
    "block_time_ms": 380,
    "avg_fee_usdc": "0.001234",
    "total_fee_usdc": "0.012345",
    "total_usdc_transferred": "150000.000000",
    "indexed_block": 12344,
    "utilization_pct": 3.2,
    "tx_count": 5,
    ...
  },
  "health": {
    "last_indexed_block": 12344,
    "chain_tip": 12345,
    "lag_blocks": 1,
    "syncing": false,
    "errors_1h": 0,
    "avg_batch_ms": 45
  },
  "blocks": [
    {"number": 12345, "timestamp": 1700000000, "tx_count": 5, "utilization_pct": 3.2, ...},
    ...
  ],
  "transactions": [
    {"hash": "0x...", "from_addr": "0x...", "status": 1, ...},
    ...
  ],
  "block_stats": [
    {"block_number": 12345, "tps": 12.5, "avg_fee_usdc": "0.001234", ...},
    ...
  ]
}
```

### `analytics` topic

```json
{
  "window": "24h",
  "overview": {
    "snapshot_at": 1700000000,
    "transfers_count": 1234,
    "transfer_volume": 5600000.0,
    "fees_total": 1.234,
    "bridge_net_flow": 100000.0,
    "agent_count": 42,
    ...
  },
  "bridge_flow": {
    "inbound_vol": 500000.0,
    "inbound_count": 23,
    "outbound_vol": 400000.0,
    "outbound_count": 18,
    "by_chain": {"Ethereum": {"inbound_vol": 500000, ...}, ...}
  },
  "volume": {
    "whale_transfers": 5,
    "by_token": {
      "USDC": {"volume": 4000000, "count": 800, "whale_count": 3},
      ...
    }
  }
}
```

---

## File Structure After Implementation

```
internal/server/realtime/
├── notify.go            # Broadcast() — generic SSE fan-out via SubscriptionsBroker
└── broadcaster.go       # BroadcastIndexerUpdate(), BroadcastAnalyticsUpdate()

internal/indexer/
├── runner.go            # +1 line: go realtime.BroadcastIndexerUpdate(app) after batch

internal/jobs/
├── analytics_snapshot.go  # +1 line: go realtime.BroadcastAnalyticsUpdate() after snapshot

frontend/src/lib/
├── realtime.ts           # SSE connection manager, subscribes to topics, updates stores

frontend/src/routes/
├── +layout.svelte        # Remove 8s polling interval, use connectRealtime()
└── overview/+page.svelte # Remove 2s/10s intervals, rely on SSE
```

---

## Open Questions

1. **Should the `indexer` topic always include `block_stats` (200 rows)?** That's a big payload. Alternative: send only the latest 10 block stats and let the frontend request the full chart data via REST. Or: only include block_stats when there are subscribers (check `HasSubscription`).

2. **Should we expose native record-level subscriptions too?** For power users who want to watch a specific wallet address or agent. This would use PocketBase's built-in collection subscriptions with no extra server code — just set `viewRule: ""` (public) on the collections and let clients subscribe to e.g. `transfers?options={"query":{"from_addr":"0x..."}}`. This could be a Phase 4 addition.

3. **Token analytics page**: The tokens list changes slowly (every 30 min). Low priority for SSE — keep polling with a long interval (60s) or add a `token_analytics` topic.

4. **Graph page**: The 3D force graph uses wallet edges. Edges update per-batch. A dedicated `edges` topic could push edge deltas for real-time graph updates, but this is a nice-to-have.
