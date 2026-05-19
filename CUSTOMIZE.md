# Customization Guide

How to extend the Go backend: index new contracts, add API endpoints, run background jobs, and connect external scripts or agents to the live data.

---

## Index a new contract event

This is the most common extension. Say you have a new contract and want to index one of its events.

### 1. Add the contract address and event topic — `internal/chain/arc/arc.go`

```go
// in the var block of contract addresses
AddrMyContract = common.HexToAddress("0xYourContractAddress")

// in the var block of event topics
TopicMyEvent = crypto.Keccak256Hash([]byte("MyEvent(address,uint256)"))
```

### 2. Tell HyperSync to stream that log — `internal/indexer/query.go`

Add a new `LogSelection` entry inside `newIndexerQuery`:

```go
{
    Address: []common.Address{arc.AddrMyContract},
    Topics:  [][]common.Hash{{arc.TopicMyEvent}},
},
```

HyperSync only streams logs you explicitly ask for, so this is required before any data arrives.

### 3. Define the collection — `internal/server/collections/`

Create a new file (e.g. `myevents.go`) or add to an existing one:

```go
func myEventsCollection(app core.App) error {
    if collectionExists(app, "my_events") {
        return nil
    }
    c := core.NewBaseCollection("my_events")
    c.Fields.Add(&core.TextField{Name: "tx_hash",      Required: true,  Max: 66})
    c.Fields.Add(&core.NumberField{Name: "block_number"})
    c.Fields.Add(&core.NumberField{Name: "log_index"})
    c.Fields.Add(&core.TextField{Name: "from_addr",    Required: false, Max: 42})
    c.Fields.Add(&core.NumberField{Name: "amount"})
    // unique index prevents re-inserting the same log on restart
    c.AddIndex("idx_my_events_unique", true, "tx_hash, log_index", "")
    c.AddIndex("idx_my_events_block",  false, "block_number", "")
    c.ViewRule = nil
    return app.Save(c)
}
```

Then register it in `internal/server/collections/register.go` inside `RegisterCollections`:

```go
if err := myEventsCollection(app); err != nil {
    return err
}
```

### 4. Write the save function — `internal/indexer/save_myevent.go`

```go
package indexer

import (
    "fmt"
    "math/big"

    "github.com/enviodev/hypersync-client-go/types"
    "github.com/pocketbase/pocketbase/core"

    "arcadia/internal/utils"
)

func saveMyEvent(app core.App, log *types.Log, seen *batchSeen) error {
    if log.Topic1 == nil || log.TransactionHash == nil || log.LogIndex == nil {
        return nil
    }

    txHash := log.TransactionHash.Hex()
    logIdx := *log.LogIndex

    // skip if already indexed (seen map prevents re-inserting on restart)
    if _, dup := seen.myEvents[txLogKey{txHash, logIdx}]; dup {
        return nil
    }

    var amount *big.Int
    if log.Data != nil && len(*log.Data) >= 32 {
        amount = new(big.Int).SetBytes((*log.Data)[:32])
    }

    coll, err := utils.FindCollection(app, "my_events")
    if err != nil {
        return err
    }
    r := core.NewRecord(coll)
    r.Set("tx_hash", txHash)
    if log.BlockNumber != nil {
        r.Set("block_number", log.BlockNumber.Uint64())
    }
    r.Set("log_index", logIdx)
    r.Set("from_addr", common.BytesToAddress(log.Topic1.Bytes()[12:]).Hex())
    if amount != nil {
        r.Set("amount", amount.Int64())
    }
    if err := app.Save(r); err != nil {
        return fmt.Errorf("save my_event %s/%d: %w", txHash, logIdx, err)
    }
    seen.myEvents[txLogKey{txHash, logIdx}] = struct{}{}
    return nil
}
```

### 5. Add the seen map for deduplication — `internal/indexer/seen.go`

In `batchSeen`:
```go
myEvents map[txLogKey]struct{}
```

In `newBatchSeen()`:
```go
myEvents: map[txLogKey]struct{}{},
```

In `loadBatchSeen` add a range query to pre-populate the set from existing rows, following the same pattern as `transfers` or `crosschain`.

### 6. Wire into the log router — `internal/indexer/save_trace.go`

In `routeLog`, add a case:

```go
case arc.TopicMyEvent:
    if addr == arc.AddrMyContract {
        return nil, saveMyEvent(app, log, seen)
    }
```

That's it. On the next indexer run the new logs will be streamed, saved, and deduplicated.

---

## Add a new API endpoint

### 1. Add a repo function — `internal/repo/`

Create or extend a file. Follow the existing pattern:

```go
// internal/repo/myevents.go
package repo

import "github.com/pocketbase/pocketbase/core"

func ListMyEvents(app core.App, limit, offset int) ([]*core.Record, error) {
    return FindRecords(app, "my_events", "", "-block_number", limit, offset)
}
```

### 2. Add a handler — `internal/server/handlers/`

Create or extend a handler file:

```go
// internal/server/handlers/myevents.go
package handlers

import (
    "net/http"

    "github.com/pocketbase/pocketbase/core"

    "arcadia/internal/repo"
)

func myEventsHandler(e *core.RequestEvent) error {
    limit, offset := limitOffset(e)
    records, err := repo.ListMyEvents(e.App, limit, offset)
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }
    return e.JSON(http.StatusOK, map[string]any{
        "items":  repo.RecordMaps(records),
        "limit":  limit,
        "offset": offset,
    })
}
```

### 3. Register the route — `internal/server/handlers/routes.go`

Inside `registerV1Routes`:

```go
v1.GET("/my_events", myEventsHandler)
```

The endpoint is now live at `/api/v1/my_events` and auto-documented in the Swagger UI.

---

## Add a background job

Jobs run on a cron schedule and are visible in the pb-ext admin dashboard.

### 1. Create the job file — `internal/jobs/myjob.go`

```go
package jobs

import (
    "fmt"

    "github.com/magooney-loon/pb-ext/core/jobs"
    "github.com/pocketbase/pocketbase/core"
)

func myJob(app core.App) error {
    jm := jobs.GetManager()
    if jm == nil {
        return fmt.Errorf("job manager not initialized")
    }
    return jm.RegisterJob(
        "myJob",                          // internal ID (unique)
        "My Job",                         // display name in admin UI
        "Does something useful every hour", // description
        "0 * * * *",                      // cron: every hour on the hour
        func(el *jobs.ExecutionLogger) {
            el.Start("My Job")
            // ... do work using app ...
            el.Complete("done")
        },
    )
}
```

### 2. Register it — `internal/jobs/jobs.go`

```go
if err := myJob(app); err != nil {
    app.Logger().Error("Failed to register my job", "error", err)
    return err
}
```

---

## Connect external scripts, agents, or bots

You don't need to modify the Go code at all to consume data. Three options, in order of simplicity:

### Option A — REST API

The versioned REST API is always running. Poll any endpoint from any language:

```bash
# latest chain stats
curl http://localhost:8090/api/v1/stats

# recent transfers, paginated
curl "http://localhost:8090/api/v1/transfers?limit=50&offset=0"

# all events for a wallet
curl http://localhost:8090/api/v1/wallet/0xYourAddress

# agent detail + job history
curl http://localhost:8090/api/v1/agents/0xAgentAddress
```

Full schema and query parameters are in the Swagger UI at <http://localhost:8090/api/docs/v1/swagger>.

### Option B — SSE (real-time push)

Subscribe to the live feed and receive a push after every indexed batch (~1 Hz). Works from any language that supports Server-Sent Events.

```bash
# subscribe to the indexer topic (blocks, txs, stats after every batch)
curl -N -H "Accept: text/event-stream" \
  "http://localhost:8090/api/realtime?topics=indexer"
```

```python
# Python example using sseclient-py
import sseclient, requests, json

url = "http://localhost:8090/api/realtime"
r = requests.get(url, params={"topics": "indexer"}, stream=True)
client = sseclient.SSEClient(r)
for event in client.events():
    data = json.loads(event.data)
    print(data["stats"])      # live chain metrics
    print(data["blocks"])     # latest blocks in this batch
```

Available topics:
| Topic | Payload | Frequency |
|---|---|---|
| `indexer` | `{stats, health, blocks[], transactions[]}` | Every batch (~1 Hz) |
| `analytics` | `{window, overview, bridge_flow, volume}` | Every 5 min |
| `charts` | `{block_stats[]}` | Every batch |

### Option C — Direct SQLite access (read-only)

The database is a standard SQLite file. Any SQLite client can open it in read-only WAL mode alongside the running server — reads never block the indexer.

```python
import sqlite3, json

# Open read-only; WAL mode allows concurrent reads with the live writer
con = sqlite3.connect("file:pb_data/data.db?mode=ro", uri=True)

# Latest 100 USDC transfers over $10K
rows = con.execute("""
    SELECT tx_hash, block_number, from_addr, to_addr, amount_human
    FROM   transfers
    WHERE  token_symbol = 'USDC'
      AND  amount_num   > 10000
    ORDER  BY block_number DESC
    LIMIT  100
""").fetchall()
```

```go
// Go example — open a second read-only connection
db, _ := sql.Open("sqlite3", "file:pb_data/data.db?mode=ro&_journal_mode=WAL")
rows, _ := db.Query(
    `SELECT from_addr, to_addr, amount_human FROM transfers
     WHERE token_symbol = 'USDC' ORDER BY block_number DESC LIMIT 10`)
```

**Tables you can query directly:**

| Table | What's in it |
|---|---|
| `blocks` | Every indexed block |
| `transactions` | Every transaction |
| `transfers` | ERC-20/721/1155 Transfer events |
| `traces` | Internal calls (CALL, DELEGATECALL, …) |
| `agents` | ERC-8004 registered agents |
| `agent_jobs` | ERC-8183 job lifecycle |
| `crosschain_events` | CCTP + Gateway bridge events |
| `fx_swaps` | StableFX trades |
| `wallet_edges` | Aggregated wallet-to-wallet flows |
| `block_stats` | Per-block derived metrics (tps, fees, utilization) |
| `analytics_snapshots` | Pre-aggregated 1h/24h/7d windows |
| `token_analytics` | Token metadata + transfer counts |
| `indexer_meta` | Cursor + key-value metadata (e.g. last indexed block) |

---

## Push custom data to the dashboard via SSE

If you want your custom collection or job to appear live in the frontend, broadcast after writing:

```go
import "arcadia/internal/server/realtime"

// after saving your records...
go realtime.BroadcastIndexerUpdate(app)   // triggers the "indexer" topic
go realtime.BroadcastAnalyticsUpdate(app, "24h") // triggers the "analytics" topic
```

To add a completely new SSE topic, add a broadcast function in `internal/server/realtime/broadcaster.go` following the existing pattern, then subscribe to it in the frontend's `src/lib/realtime.ts`.

---

## Add a new chain

1. Create `internal/chain/<name>/` with `package <name>`
2. Define constants, addresses, and event topics (mirror `internal/chain/arc/arc.go`)
3. Add ERC token resolution if needed (mirror `internal/chain/arc/erc.go`)
4. Add a new `internal/indexer/` entry point that dials the new chain's HyperSync URL
5. Register it alongside the Arc indexer in `cmd/server/main.go`

Each chain gets its own package under `internal/chain/` so constants never collide.
