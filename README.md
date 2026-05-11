# arcadia


This is the "Deep Dive" implementation. We will build a **Streaming Indexer** in Go that runs as a background process within your PocketBase application.

We will extract:
1.  **Blocks** (The Spine).
2.  **Transactions** (The Pulses).
3.  **USDC Transfers** (The Lifeblood - Arc is USDC-native).
4.  **Contract Events** (The Signals).

### Step 1: Define PocketBase Collections

You need tables to store this high-fidelity data. Run this once to set up your schema (or do it via the Admin UI).

**`cmd/server/collections.go`**
```go
package main

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func registerCollections(app core.App) {
	// 1. Blocks: The skeleton of the chain
	blocks := &models.Collection{
		Name: "blocks",
		Type: models.CollectionTypeBase,
		Schema: []*models.SchemaField{
			{Name: "number", Type: models.FieldTypeNumber, Required: true},
			{Name: "hash", Type: models.FieldTypeText, Required: true},
			{Name: "timestamp", Type: models.FieldTypeNumber},
			{Name: "gas_used", Type: models.FieldTypeNumber},
			{Name: "size", Type: models.FieldTypeNumber},
		},
		Indexes: []string{"CREATE INDEX idx_block_number ON blocks (number)"},
	}

	// 2. Transactions: The particles
	txs := &models.Collection{
		Name: "transactions",
		Type: models.CollectionTypeBase,
		Schema: []*models.SchemaField{
			{Name: "hash", Type: models.FieldTypeText, Required: true},
			{Name: "block_number", Type: models.FieldTypeNumber},
			{Name: "from", Type: models.FieldTypeText},
			{Name: "to", Type: models.FieldTypeText},
			{Name: "value", Type: models.FieldTypeText}, // Store as string to handle big Int
			{Name: "gas_price", Type: models.FieldTypeNumber},
			{Name: "status", Type: models.FieldTypeBool},
		},
		Indexes: []string{"CREATE INDEX idx_tx_block ON transactions (block_number)"},
	}

	// 3. Transfers: The "Blood Flow" (USDC Movements)
	transfers := &models.Collection{
		Name: "transfers",
		Type: models.CollectionTypeBase,
		Schema: []*models.SchemaField{
			{Name: "tx_hash", Type: models.FieldTypeText},
			{Name: "token", Type: models.FieldTypeText}, // USDC, EURC, etc.
			{Name: "from", Type: models.FieldTypeText},
			{Name: "to", Type: models.FieldTypeText},
			{Name: "amount", Type: models.FieldTypeText},
			{Name: "block_number", Type: models.FieldTypeNumber},
		},
	}

	app.Dao().SaveCollection(blocks)
	app.Dao().SaveCollection(txs)
	app.Dao().SaveCollection(transfers)
}
```

---

### Step 2: The Streaming Indexer Logic

We will create a "Background Worker" that streams data from HyperSync continuously.

**`cmd/server/indexer.go`**
```go
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/enviodev/hypersync-client-go/hypersync"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// Event Signatures (Keccak256 Hashes)
var (
	// Transfer(address indexed from, address indexed to, uint256 value)
	TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

func StartIndexer(app core.App) {
	app.Logger().Info("Starting HyperSync Streamer...")

	// 1. Create HyperSync Client
	// Arc Testnet URL: https://arc-testnet.hypersync.xyz
	client := hypersync.NewClient(hypersync.ClientConfig{
		URL: "https://arc-testnet.hypersync.xyz",
	})

	// 2. Run the stream loop in a goroutine
	go func() {
		for {
			err := runStreamCycle(app, client)
			if err != nil {
				app.Logger().Error("Indexer stream error", "error", err)
				// Wait 5 seconds before retrying to avoid tight error loops
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func runStreamCycle(app core.App, client hypersync.Client) error {
	ctx := context.Background()

	// A. Get last processed block from DB
	lastBlock := getLastBlock(app)
	app.Logger().Info("Resuming stream from block", "block", lastBlock)

	// B. Define the Query
	// We want Blocks, Transactions, AND Logs (for Transfer events)
	query := hypersync.Query{
		FromBlock: lastBlock,
		FieldSelection: hypersync.FieldSelection{
			Block: []string{"number", "hash", "timestamp", "gas_used", "size"},
			Transaction: []string{"hash", "block_number", "from", "to", "value", "gas_price", "status"},
			Log: []string{"block_number", "transaction_hash", "address", "topics", "data"},
		},
		Logs: []hypersync.LogSelection{
			{
				// We specifically want ALL Transfer events to visualize value flow
				Topics: [][]string{{TransferTopic}},
			},
		},
		// Include blocks even if they have no logs (for the "heartbeat" visualization)
		IncludeAllBlocks: true, 
	}

	// C. Create the Stream
	stream, err := client.Stream(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	// D. Process Stream Data
	for {
		res, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("stream recv error: %w", err)
		}

		// If data is returned, process it
		if res.Data != nil {
			// Use a transaction for batch writing to SQLite
			err := app.Dao().RunInTransaction(func(txDao core.App) error {
				// 1. Save Blocks
				for _, blk := range res.Data.Blocks {
					record := models.Record{}
					record.Set("number", blk.Number)
					record.Set("hash", hex.EncodeToString(blk.Hash))
					record.Set("timestamp", blk.Timestamp)
					record.Set("gas_used", blk.GasUsed)
					record.Set("size", blk.Size)
					
					if err := txDao.Dao().SaveRecord("blocks", &record); err != nil {
						app.Logger().Error("Failed to save block", "number", blk.Number, "err", err)
					}
				}

				// 2. Save Transactions
				for _, tx := range res.Data.Transactions {
					record := models.Record{}
					record.Set("hash", hex.EncodeToString(tx.Hash))
					record.Set("block_number", tx.BlockNumber)
					record.Set("from", hex.EncodeToString(tx.From))
					record.Set("to", hex.EncodeToString(tx.To)) // Handle null?
					record.Set("value", new(big.Int).SetBytes(tx.Value).String())
					record.Set("gas_price", tx.GasPrice)
					record.Set("status", tx.Status == 1)

					if err := txDao.Dao().SaveRecord("transactions", &record); err != nil {
						app.Logger().Error("Failed to save tx", "hash", hex.EncodeToString(tx.Hash), "err", err)
					}
				}

				// 3. Save & Decode Transfers (The Cool Part)
				for _, log := range res.Data.Logs {
					// Check if it's a Transfer event (Topic0 match is implied by query, but let's be safe)
					if len(log.Topics) >= 3 {
						from := hex.EncodeToString(log.Topics[1][12:]) // Address is last 20 bytes of 32byte topic
						to := hex.EncodeToString(log.Topics[2][12:])
						
						// Decode Amount from Data field
						amount := new(big.Int).SetBytes(log.Data)

						record := models.Record{}
						record.Set("tx_hash", hex.EncodeToString(log.TransactionHash))
						record.Set("token", hex.EncodeToString(log.Address))
						record.Set("from", from)
						record.Set("to", to)
						record.Set("amount", amount.String())
						record.Set("block_number", log.BlockNumber)

						if err := txDao.Dao().SaveRecord("transfers", &record); err != nil {
							app.Logger().Error("Failed to save transfer", "err", err)
						}
					}
				}
				return nil
			})

			if err != nil {
				app.Logger().Error("Transaction batch failed", "error", err)
			} else {
				app.Logger().Info("Synced batch", "next_block", res.NextBlock)
			}
		}

		// E. Handle Next Block (Pagination)
		// Update our cursor if the stream advanced
		if res.NextBlock > lastBlock {
			// In a real app, update a '_meta' table with res.NextBlock
			// For this hack, we just let the loop continue
		}
	}
}

func getLastBlock(app core.App) uint64 {
	// Simple implementation: Find the highest block in DB
	// In prod, store this in a separate 'meta' table
	record, err := app.Dao().FindRecordsByFilter(
		"blocks", 
		"", // filter
		"-number", // sort desc
		1,  // limit
		0,  // offset
	)
	if err != nil || len(record) == 0 {
		return 0
	}
	val, _ := record[0].Get("number").(float64) // JSON unmarshal usually uses float64 for numbers
	return uint64(val)
}
```

---

### Step 3: Hook it up to Main

Integrate the streamer into your app lifecycle.

**`cmd/server/main.go`**
```go
func initApp(devMode bool) {
    // ... existing setup ...
    
    srv := app.New(opts...)

    app.SetupLogging(srv)

    registerCollections(srv.App())
    registerRoutes(srv.App())
    registerJobs(srv.App())

    // START THE INDEXER
    // We hook into OnServe so it starts after the DB is ready
    srv.App().OnServe().BindFunc(func(e *core.ServeEvent) error {
        // Run the streaming indexer
        StartIndexer(srv.App())
        
        return e.Next()
    })

    if err := srv.Start(); err != nil {
        // ... error handling
    }
}
```

### Summary of "Cool Data" Extracted

With this setup, your PocketBase instance is now a high-fidelity data engine.

1.  **`blocks` Collection:**
    *   Visualize: **Chain Growth / Spine.**
    *   Data: Block height, Gas Used (Heat), Timestamp.
2.  **`transactions` Collection:**
    *   Visualize: **Particles.**
    *   Data: Sender/Receiver (Nodes), Value (Particle Size).
3.  **`transfers` Collection:**
    *   Visualize: **The Blood Flow.**
    *   Data: This captures every ERC-20 transfer (including USDC). You can draw lines between wallets representing money moving. This is the "Agentic Data" layer showing economic activity.

This runs entirely within your Go binary, uses no external dependencies other than Arc/HyperSync, and updates your frontend in real-time via PocketBase's websocket.
