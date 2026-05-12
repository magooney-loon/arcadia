package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	hypersyncgo "github.com/enviodev/hypersync-client-go"
	"github.com/enviodev/hypersync-client-go/options"
	"github.com/enviodev/hypersync-client-go/types"
	"github.com/enviodev/hypersync-client-go/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"
)

// ── Indexer entry point ───────────────────────────────────────────────────────

func StartIndexer(app core.App) {
	log.Println("[indexer] starting Arcadia HyperSync indexer")
	go func() {
		attempt := 0
		for {
			attempt++
			log.Printf("[indexer] run attempt #%d", attempt)
			recordIndexerEvent(app, "info", "run_start", "starting indexer run", indexerEventFields{"attempt": attempt})
			if err := runIndexer(app, attempt); err != nil {
				msg := err.Error()
				if strings.Contains(msg, "429") {
					log.Printf("[indexer] rate-limited (429) — waiting 30s before retry (attempt #%d)", attempt)
					app.Logger().Warn("Indexer rate-limited", "attempt", attempt)
					recordIndexerEvent(app, "warn", "rate_limited", "HyperSync returned 429; backing off before retry", indexerEventFields{"attempt": attempt, "error": err})
					time.Sleep(30 * time.Second)
				} else {
					log.Printf("[indexer] crashed: %v — restarting in 5s (attempt #%d)", err, attempt)
					app.Logger().Error("Indexer crashed", "error", err, "attempt", attempt)
					recordIndexerEvent(app, "error", "run_error", "indexer run failed; restarting", indexerEventFields{"attempt": attempt, "error": err})
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
}

type indexerEventFields map[string]any

func recordIndexerEvent(app core.App, level, event, message string, fields indexerEventFields) {
	c, err := app.FindCollectionByNameOrId("indexer_events")
	if err != nil {
		app.Logger().Debug("Indexer event collection unavailable", "event", event, "error", err)
		return
	}

	r := core.NewRecord(c)
	r.Set("timestamp", time.Now().Unix())
	r.Set("level", level)
	r.Set("event", event)
	r.Set("message", message)
	for key, val := range fields {
		switch key {
		case "attempt", "batch", "block", "tip", "lag", "duration_ms", "blocks", "transactions", "logs", "error":
			if key == "error" {
				if val != nil {
					r.Set("error", fmt.Sprint(val))
				}
				continue
			}
			r.Set(key, val)
		}
	}
	if err := app.Save(r); err != nil {
		app.Logger().Warn("Failed to persist indexer event", "event", event, "error", err)
	}
}

func getChainTip(ctx context.Context, client interface {
	GetHeight(context.Context) (*big.Int, error)
}) (uint64, error) {
	height, err := client.GetHeight(ctx)
	if err != nil {
		return 0, err
	}
	if height == nil {
		return 0, fmt.Errorf("chain height response was nil")
	}
	return height.Uint64(), nil
}

func logIndexerHeartbeat(ctx context.Context, app core.App, client interface {
	GetHeight(context.Context) (*big.Int, error)
}, attempt int, batchCount, currentBlock uint64, lastBatchAt time.Time, processingBatch uint64, processingStartedAt time.Time, persist bool) {
	idleFor := time.Since(lastBatchAt).Round(time.Second)
	tipCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	tip, tipErr := getChainTip(tipCtx, client)
	cancel()
	lag := uint64(0)
	if tip > currentBlock {
		lag = tip - currentBlock
	}

	processingFor := time.Duration(0)
	if processingBatch > 0 && !processingStartedAt.IsZero() {
		processingFor = time.Since(processingStartedAt).Round(time.Second)
	}

	if tipErr != nil {
		if processingBatch > 0 {
			log.Printf("[indexer] heartbeat | processing batch #%d for %s | block %d | completed_batches=%d | tip=? err=%v", processingBatch, processingFor, currentBlock, batchCount, tipErr)
		} else {
			log.Printf("[indexer] heartbeat | idle %s | block %d | completed_batches=%d | tip=? err=%v", idleFor, currentBlock, batchCount, tipErr)
		}
		app.Logger().Warn("Indexer heartbeat tip check failed", "block", currentBlock, "batches", batchCount, "idle_for", idleFor.String(), "processing_batch", processingBatch, "processing_for", processingFor.String(), "error", tipErr)
		if persist {
			recordIndexerEvent(app, "warn", "heartbeat", "indexer heartbeat tip check failed", indexerEventFields{"attempt": attempt, "batch": batchCount, "block": currentBlock, "error": tipErr})
		}
		return
	}

	if processingBatch > 0 {
		log.Printf("[indexer] heartbeat | processing batch #%d for %s | block %d | tip %d | lag %d | completed_batches=%d", processingBatch, processingFor, currentBlock, tip, lag, batchCount)
		app.Logger().Info("Indexer heartbeat", "block", currentBlock, "tip", tip, "lag", lag, "batches", batchCount, "processing_batch", processingBatch, "processing_for", processingFor.String())
	} else {
		log.Printf("[indexer] heartbeat | idle %s | block %d | tip %d | lag %d | batches=%d", idleFor, currentBlock, tip, lag, batchCount)
		app.Logger().Info("Indexer heartbeat", "block", currentBlock, "tip", tip, "lag", lag, "batches", batchCount, "idle_for", idleFor.String())
	}
	if persist {
		recordIndexerEvent(app, "info", "heartbeat", "indexer heartbeat", indexerEventFields{"attempt": attempt, "batch": batchCount, "block": currentBlock, "tip": tip, "lag": lag})
	}
}

func runIndexer(app core.App, attempt int) error {
	ctx := context.Background()

	apiToken := EnvioAPIToken()
	if apiToken == "" {
		return fmt.Errorf("ENVIO_API_TOKEN not set — get one at envio.dev")
	}

	rpc := NextRPCURL()
	log.Printf("[indexer] connecting — hypersync: %s  rpc: %s", ArcHyperSyncURL, rpc)

	hyper, err := hypersyncgo.NewHyper(ctx, options.Options{
		Blockchains: []options.Node{
			{
				Type:        utils.EthereumNetwork,
				NetworkId:   ArcNetworkID,
				Endpoint:    ArcHyperSyncURL,
				RpcEndpoint: rpc,
				ApiToken:    apiToken,
				// Fail fast on 429 so our outer backoff+rotation kicks in quickly.
				// Library retries: 3 attempts × ≤3s ceiling = ≤9s before error surfaces.
				MaxNumRetries:  3,
				RetryBaseMs:    500 * time.Millisecond,
				RetryBackoffMs: 500 * time.Millisecond,
				RetryCeilingMs: 3 * time.Second,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create HyperSync client: %w", err)
	}

	client, ok := hyper.GetClient(ArcNetworkID)
	if !ok {
		return fmt.Errorf("arc client not found in hyper")
	}

	fromBlock := resolveStartBlock(ctx, app, client)
	log.Printf("[indexer] streaming from block %d", fromBlock)
	recordIndexerEvent(app, "info", "stream_start", "streaming from saved start block", indexerEventFields{"attempt": attempt, "block": fromBlock})

	query := &types.Query{
		FromBlock:        new(big.Int).SetUint64(fromBlock),
		ToBlock:          new(big.Int).SetUint64(^uint64(0)), // stream to chain tip indefinitely
		IncludeAllBlocks: true,
		FieldSelection: types.FieldSelection{
			Block: []string{
				"number", "hash", "parent_hash", "timestamp",
				"gas_used", "gas_limit", "base_fee_per_gas", "miner", "size",
			},
			Transaction: []string{
				"hash", "block_number", "transaction_index",
				"from", "to", "value", "nonce", "input",
				"gas_price", "gas_used", "effective_gas_price",
				"type", "contract_address",
			},
			Log: []string{
				"block_number", "transaction_hash", "log_index",
				"address", "topic0", "topic1", "topic2", "topic3", "data",
			},
		},
		Logs: []types.LogSelection{
			{Topics: [][]common.Hash{{TopicTransfer}}},
			{
				Address: []common.Address{AddrCCTPTokenMessenger},
				Topics:  [][]common.Hash{{TopicDepositForBurn}},
			},
			{
				Address: []common.Address{AddrCCTPMessageTransmitter},
				Topics:  [][]common.Hash{{TopicMessageReceived}},
			},
			{Address: []common.Address{AddrGatewayWallet, AddrGatewayMinter}},
			{Address: []common.Address{AddrFxEscrow}},
		},
	}

	streamOpts := &options.StreamOptions{
		Concurrency: big.NewInt(1),   // single in-flight request — avoids 429 bursts
		BatchSize:   big.NewInt(200), // blocks per batch; small enough for free tier
	}

	log.Println("[indexer] creating stream...")
	stream, err := client.Stream(ctx, query, streamOpts)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	// Subscribe() is blocking — it does the initial GetArrow fetch then calls
	// g.Wait() which blocks until the worker finishes. Run it in a goroutine
	// and forward its return value so our select loop can detect fatal errors.
	subErrCh := make(chan error, 1)
	go func() {
		log.Println("[indexer] Subscribe() starting (initial fetch + worker)...")
		subErrCh <- stream.Subscribe()
	}()

	var currentBlock atomic.Uint64
	currentBlock.Store(fromBlock)
	var completedBatches atomic.Uint64
	var lastBatchAtUnixNano atomic.Int64
	lastBatchAtUnixNano.Store(time.Now().UnixNano())
	var processingBatch atomic.Uint64
	var processingStartedAtUnixNano atomic.Int64

	heartbeatStop := make(chan struct{})
	defer close(heartbeatStop)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				lastBatchAt := time.Unix(0, lastBatchAtUnixNano.Load())
				processingStartedAt := time.Time{}
				if started := processingStartedAtUnixNano.Load(); started > 0 {
					processingStartedAt = time.Unix(0, started)
				}
				activeBatch := processingBatch.Load()
				// Persist idle heartbeats. During a batch, SQLite may be inside a
				// long write transaction, so keep in-flight heartbeats to stdout.
				logIndexerHeartbeat(ctx, app, client, attempt, completedBatches.Load(), currentBlock.Load(), lastBatchAt, activeBatch, processingStartedAt, activeBatch == 0)
			case <-heartbeatStop:
				return
			}
		}
	}()

	log.Println("[indexer] event loop running — waiting for first batch")

	for {
		select {
		case subErr := <-subErrCh:
			if subErr != nil {
				errMsg := subErr.Error()
				if strings.Contains(errMsg, "429") {
					log.Printf("[indexer] 429 rate-limited after %d batches at block %d — handing off to backoff", completedBatches.Load(), currentBlock.Load())
				} else {
					log.Printf("[indexer] Subscribe() error after %d batches at block %d: %v", completedBatches.Load(), currentBlock.Load(), subErr)
				}
				return subErr
			}
			// Subscribe() returned nil — stream exhausted its planned range. Restart from cursor.
			log.Printf("[indexer] Subscribe() finished cleanly after %d batches at block %d — reconnecting", completedBatches.Load(), currentBlock.Load())
			return nil

		case res, ok := <-stream.Channel():
			if !ok {
				log.Println("[indexer] channel closed")
				return nil
			}
			nextBatch := completedBatches.Load() + 1
			batchStart := time.Now()
			processingBatch.Store(nextBatch)
			processingStartedAtUnixNano.Store(batchStart.UnixNano())
			log.Printf("[indexer] batch #%d starting | current block %d | blocks=%d txs=%d logs=%d",
				nextBatch, currentBlock.Load(), len(res.Data.Blocks), len(res.Data.Transactions), len(res.Data.Logs))
			recordIndexerEvent(app, "info", "batch_start", "started processing indexer batch", indexerEventFields{
				"attempt":      attempt,
				"batch":        nextBatch,
				"block":        currentBlock.Load(),
				"blocks":       len(res.Data.Blocks),
				"transactions": len(res.Data.Transactions),
				"logs":         len(res.Data.Logs),
			})
			if err := processBatch(app, res); err != nil {
				processingBatch.Store(0)
				processingStartedAtUnixNano.Store(0)
				log.Printf("[indexer] batch processing error: %v", err)
				app.Logger().Error("Batch processing error", "error", err)
				recordIndexerEvent(app, "error", "batch_error", "batch failed; cursor was not advanced", indexerEventFields{
					"attempt":      attempt,
					"batch":        nextBatch,
					"block":        currentBlock.Load(),
					"blocks":       len(res.Data.Blocks),
					"transactions": len(res.Data.Transactions),
					"logs":         len(res.Data.Logs),
					"error":        err,
				})
				return err
			}
			if res.NextBlock != nil {
				next := res.NextBlock.Uint64()
				currentBlock.Store(next)
				if err := setLastIndexedBlock(app, next); err != nil {
					processingBatch.Store(0)
					processingStartedAtUnixNano.Store(0)
					recordIndexerEvent(app, "error", "cursor_error", "failed to persist cursor after batch", indexerEventFields{"attempt": attempt, "batch": nextBatch, "block": next, "error": err})
					return err
				}
			}
			batchCount := completedBatches.Add(1)
			elapsed := time.Since(batchStart).Milliseconds()
			lastBatchAtUnixNano.Store(time.Now().UnixNano())
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			log.Printf("[indexer] batch #%d | block %d | blocks=%d txs=%d logs=%d | %dms",
				batchCount, currentBlock.Load(),
				len(res.Data.Blocks), len(res.Data.Transactions), len(res.Data.Logs),
				elapsed,
			)
			app.Logger().Info("Indexer progress",
				"batch", batchCount,
				"block", currentBlock.Load(),
				"duration_ms", elapsed,
				"blocks", len(res.Data.Blocks),
				"transactions", len(res.Data.Transactions),
				"logs", len(res.Data.Logs),
			)
			recordIndexerEvent(app, "info", "batch_done", "finished processing indexer batch", indexerEventFields{
				"attempt":      attempt,
				"batch":        batchCount,
				"block":        currentBlock.Load(),
				"duration_ms":  elapsed,
				"blocks":       len(res.Data.Blocks),
				"transactions": len(res.Data.Transactions),
				"logs":         len(res.Data.Logs),
			})
			if batchCount%10 == 0 {
				app.Logger().Info("Indexer progress",
					"batch", batchCount,
					"block", currentBlock.Load(),
				)
				recordIndexerEvent(app, "info", "progress", "processed indexer batch", indexerEventFields{
					"attempt":      attempt,
					"batch":        batchCount,
					"block":        currentBlock.Load(),
					"duration_ms":  elapsed,
					"blocks":       len(res.Data.Blocks),
					"transactions": len(res.Data.Transactions),
					"logs":         len(res.Data.Logs),
				})
			}
			// Pace requests to avoid HyperSync free-tier burst throttling.
			// 400ms between batches ≈ 2.5 req/s — well within typical fair-use limits.
			time.Sleep(400 * time.Millisecond)
			stream.Ack()

		case <-stream.Done():
			log.Printf("[indexer] stream done at block %d — reconnecting", currentBlock.Load())
			return nil
		}
	}
}

// ── Batch processing ──────────────────────────────────────────────────────────

func processBatch(app core.App, res *types.QueryResponse) error {
	// aggregate per-block stats within this batch
	type blockAcc struct {
		txCount         int
		uniqueSenders   map[string]struct{}
		uniqueReceivers map[string]struct{}
		newContracts    int
		totalFee        *big.Int
		totalUSDC       *big.Int
		totalEURC       *big.Int
		totalUSYC       *big.Int
		largestUSDC     *big.Int
	}
	perBlock := make(map[uint64]*blockAcc)

	getAcc := func(blockNum uint64) *blockAcc {
		if _, ok := perBlock[blockNum]; !ok {
			perBlock[blockNum] = &blockAcc{
				uniqueSenders:   make(map[string]struct{}),
				uniqueReceivers: make(map[string]struct{}),
				totalFee:        new(big.Int),
				totalUSDC:       new(big.Int),
				totalEURC:       new(big.Int),
				totalUSYC:       new(big.Int),
				largestUSDC:     new(big.Int),
			}
		}
		return perBlock[blockNum]
	}

	return app.RunInTransaction(func(txApp core.App) error {
		// 1. Blocks
		for _, blk := range res.Data.Blocks {
			if blk.Number == nil {
				continue
			}
			if err := saveBlock(txApp, &blk); err != nil {
				return err
			}
			getAcc(blk.Number.Uint64()) // ensure acc exists even for empty blocks
		}

		// 2. Transactions
		for _, tx := range res.Data.Transactions {
			if tx.Hash == nil || tx.BlockNumber == nil {
				continue
			}
			fee, err := saveTransaction(txApp, &tx)
			if err != nil {
				return err
			}
			bn := tx.BlockNumber.Uint64()
			acc := getAcc(bn)
			acc.txCount++
			if tx.From != nil {
				acc.uniqueSenders[tx.From.Hex()] = struct{}{}
			}
			if tx.To != nil {
				acc.uniqueReceivers[tx.To.Hex()] = struct{}{}
			}
			if tx.ContractAddress != nil {
				acc.newContracts++
			}
			if fee != nil {
				acc.totalFee.Add(acc.totalFee, fee)
			}
		}

		// 3. Logs → transfers / crosschain / fx
		for _, log := range res.Data.Logs {
			if log.BlockNumber == nil {
				continue
			}
			bn := log.BlockNumber.Uint64()
			acc := getAcc(bn)
			amount, err := routeLog(txApp, &log)
			if err != nil {
				return err
			}
			if amount != nil && log.Address != nil {
				switch *log.Address {
				case AddrUSDC:
					acc.totalUSDC.Add(acc.totalUSDC, amount)
					if amount.Cmp(acc.largestUSDC) > 0 {
						acc.largestUSDC.Set(amount)
					}
				case AddrEURC:
					acc.totalEURC.Add(acc.totalEURC, amount)
				case AddrUSYC:
					acc.totalUSYC.Add(acc.totalUSYC, amount)
				}
			}
		}

		// 4. Block stats + tx_count back-fill onto blocks.
		//    Sort blocks ascending so we can compute consecutive block_time_ms.
		type blkTs struct {
			num uint64
			ts  int64
		}
		var sortedBlocks []blkTs
		for _, blk := range res.Data.Blocks {
			if blk.Number != nil && blk.Timestamp != nil {
				sortedBlocks = append(sortedBlocks, blkTs{blk.Number.Uint64(), blk.Timestamp.Unix()})
			}
		}
		// simple insertion sort (batches are small)
		for i := 1; i < len(sortedBlocks); i++ {
			for j := i; j > 0 && sortedBlocks[j].num < sortedBlocks[j-1].num; j-- {
				sortedBlocks[j], sortedBlocks[j-1] = sortedBlocks[j-1], sortedBlocks[j]
			}
		}

		// block_time_ms indexed by block number — look up prev from DB for the first block
		blockTimeMs := make(map[uint64]int64)
		for i, bt := range sortedBlocks {
			if i == 0 {
				prev, err := txApp.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": bt.num - 1})
				if err != nil {
					return fmt.Errorf("find previous block %d: %w", bt.num-1, err)
				}
				if len(prev) > 0 {
					prevTs := prev[0].GetInt("timestamp")
					blockTimeMs[bt.num] = (bt.ts - int64(prevTs)) * 1000
				}
			} else {
				blockTimeMs[bt.num] = (bt.ts - sortedBlocks[i-1].ts) * 1000
			}
		}

		for _, blk := range res.Data.Blocks {
			if blk.Number == nil || blk.Timestamp == nil {
				continue
			}
			bn := blk.Number.Uint64()
			acc := getAcc(bn)

			// back-fill tx_count onto the blocks record
			existingBlocks, err := txApp.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": bn})
			if err != nil {
				return fmt.Errorf("find block %d for stats backfill: %w", bn, err)
			}
			if len(existingBlocks) > 0 {
				existingBlocks[0].Set("tx_count", acc.txCount)
				if bms, ok := blockTimeMs[bn]; ok && bms > 0 {
					existingBlocks[0].Set("block_time_ms", bms)
				}
				if err := txApp.Save(existingBlocks[0]); err != nil {
					return fmt.Errorf("save block %d stats backfill: %w", bn, err)
				}
			}

			// skip block_stats if already persisted (indexer restart)
			existingStats, err := txApp.FindRecordsByFilter("block_stats", "block_number = {:n}", "", 1, 0, map[string]any{"n": bn})
			if err != nil {
				return fmt.Errorf("find block_stats %d: %w", bn, err)
			}
			if len(existingStats) > 0 {
				continue
			}

			avgFee := new(big.Int)
			if acc.txCount > 0 && acc.totalFee.Sign() > 0 {
				avgFee.Div(acc.totalFee, big.NewInt(int64(acc.txCount)))
			}

			gasUsed := uint64(0)
			gasLimit := uint64(1)
			if blk.GasUsed != nil {
				gasUsed = *blk.GasUsed
			}
			if blk.GasLimit != nil && *blk.GasLimit > 0 {
				gasLimit = *blk.GasLimit
			}
			utilPct := float64(gasUsed) / float64(gasLimit) * 100

			bms := blockTimeMs[bn]
			var tps float64
			if bms > 0 {
				tps = float64(acc.txCount) / (float64(bms) / 1000.0)
			}

			stats := core.NewRecord(mustCollection(txApp, "block_stats"))
			stats.Set("block_number", bn)
			stats.Set("timestamp", blk.Timestamp.Unix())
			stats.Set("tx_count", acc.txCount)
			stats.Set("block_time_ms", bms)
			stats.Set("tps", tps)
			stats.Set("avg_fee_usdc", weiToUSDC(avgFee))
			stats.Set("total_fee_usdc", weiToUSDC(acc.totalFee))
			stats.Set("total_usdc_transferred", stablecoinHuman(acc.totalUSDC))
			stats.Set("total_eurc_transferred", stablecoinHuman(acc.totalEURC))
			stats.Set("total_usyc_transferred", stablecoinHuman(acc.totalUSYC))
			stats.Set("unique_senders", len(acc.uniqueSenders))
			stats.Set("unique_receivers", len(acc.uniqueReceivers))
			stats.Set("new_contracts", acc.newContracts)
			stats.Set("largest_usdc_transfer", stablecoinHuman(acc.largestUSDC))
			stats.Set("utilization_pct", utilPct)

			if err := txApp.Save(stats); err != nil {
				txApp.Logger().Error("Failed to save block_stats", "block", bn, "error", err)
				return fmt.Errorf("save block_stats %d: %w", bn, err)
			}
		}

		return nil
	})
}

// ── Individual record savers ──────────────────────────────────────────────────

func saveBlock(app core.App, blk *types.Block) error {
	// skip if already exists
	existing, err := app.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": blk.Number.Uint64()})
	if err != nil {
		return fmt.Errorf("find block %d: %w", blk.Number.Uint64(), err)
	}
	if len(existing) > 0 {
		return nil
	}

	r := core.NewRecord(mustCollection(app, "blocks"))
	r.Set("number", blk.Number.Uint64())
	if blk.Hash != nil {
		r.Set("hash", blk.Hash.Hex())
	}
	if blk.ParentHash != nil {
		r.Set("parent_hash", blk.ParentHash.Hex())
	}
	if blk.Miner != nil {
		r.Set("miner", blk.Miner.Hex())
	}
	if blk.Timestamp != nil {
		r.Set("timestamp", blk.Timestamp.Unix())
	}
	if blk.GasUsed != nil {
		r.Set("gas_used", *blk.GasUsed)
	}
	if blk.GasLimit != nil {
		r.Set("gas_limit", *blk.GasLimit)
		if blk.GasUsed != nil && *blk.GasLimit > 0 {
			r.Set("utilization_pct", float64(*blk.GasUsed)/float64(*blk.GasLimit)*100)
		}
	}
	if blk.BaseFeePerGas != nil {
		r.Set("base_fee_per_gas", blk.BaseFeePerGas.String())
	}
	if blk.Size != nil {
		r.Set("size", *blk.Size)
	}

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save block", "number", blk.Number.Uint64(), "error", err)
		return fmt.Errorf("save block %d: %w", blk.Number.Uint64(), err)
	}

	app.Logger().Debug("Block", "number", blk.Number.Uint64(), "hash", func() string {
		if blk.Hash != nil {
			return blk.Hash.Hex()[:10] + "…"
		}
		return ""
	}())
	return nil
}

// saveTransaction saves the transaction and returns the fee in wei (for stats accumulation).
func saveTransaction(app core.App, tx *types.Transaction) (*big.Int, error) {
	existing, err := app.FindRecordsByFilter("transactions", "hash = {:h}", "", 1, 0, map[string]any{"h": tx.Hash.Hex()})
	if err != nil {
		return nil, fmt.Errorf("find transaction %s: %w", tx.Hash.Hex(), err)
	}
	if len(existing) > 0 {
		return nil, nil
	}

	r := core.NewRecord(mustCollection(app, "transactions"))
	r.Set("hash", tx.Hash.Hex())
	if tx.BlockNumber != nil {
		r.Set("block_number", tx.BlockNumber.Uint64())
	}
	if tx.TransactionIndex != nil {
		r.Set("transaction_index", *tx.TransactionIndex)
	}
	if tx.From != nil {
		r.Set("from_addr", tx.From.Hex())
	}
	if tx.To != nil {
		r.Set("to_addr", tx.To.Hex())
	}
	if tx.Value != nil {
		r.Set("value", tx.Value.String())
	}
	if tx.Nonce != nil {
		r.Set("nonce", *tx.Nonce)
	}
	if tx.Input != nil && len(*tx.Input) >= 4 {
		r.Set("sighash", fmt.Sprintf("0x%x", (*tx.Input)[:4]))
	}
	if tx.GasPrice != nil {
		r.Set("gas_price", tx.GasPrice.String())
	}
	if tx.GasUsed != nil {
		r.Set("gas_used", *tx.GasUsed)
	}

	var feeWei *big.Int
	if tx.GasUsed != nil && tx.EffectiveGasPrice != nil {
		feeWei = new(big.Int).Mul(new(big.Int).SetUint64(*tx.GasUsed), tx.EffectiveGasPrice)
		r.Set("effective_gas_price", tx.EffectiveGasPrice.String())
		r.Set("fee_usdc", weiToUSDC(feeWei))
	}

	if tx.Kind != nil {
		r.Set("tx_type", *tx.Kind)
	}
	isDeployment := tx.ContractAddress != nil
	if isDeployment {
		r.Set("contract_address", tx.ContractAddress.Hex())
	}
	r.Set("is_contract_deploy", isDeployment)

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save transaction", "hash", tx.Hash.Hex(), "error", err)
		return nil, fmt.Errorf("save transaction %s: %w", tx.Hash.Hex(), err)
	}
	return feeWei, nil
}

// routeLog decodes a log and routes it to the right handler.
// Returns the transfer amount (in raw uint256) if this is an ERC-20 Transfer, else nil.
func routeLog(app core.App, log *types.Log) (*big.Int, error) {
	if log.Topic0 == nil || log.Address == nil {
		return nil, nil
	}

	switch *log.Topic0 {
	case TopicTransfer:
		return saveTransfer(app, log)

	case TopicDepositForBurn:
		return nil, saveCCTPEvent(app, log, "cctp", "burn")

	case TopicMessageReceived:
		return nil, saveCCTPEvent(app, log, "cctp", "mint")

	default:
		addr := *log.Address
		if addr == AddrGatewayWallet || addr == AddrGatewayMinter {
			return nil, saveGatewayEvent(app, log)
		} else if addr == AddrFxEscrow {
			return nil, saveFxEvent(app, log)
		}
	}
	return nil, nil
}

func saveTransfer(app core.App, log *types.Log) (*big.Int, error) {
	if log.Topic1 == nil || log.Topic2 == nil || log.TransactionHash == nil || log.LogIndex == nil {
		return nil, nil
	}

	txHash := log.TransactionHash.Hex()
	logIdx := *log.LogIndex

	existing, err := app.FindRecordsByFilter("transfers",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": txHash, "i": logIdx})
	if err != nil {
		return nil, fmt.Errorf("find transfer %s/%d: %w", txHash, logIdx, err)
	}
	if len(existing) > 0 {
		return nil, nil
	}

	from := common.BytesToAddress(log.Topic1.Bytes()[12:])
	to := common.BytesToAddress(log.Topic2.Bytes()[12:])

	var amountRaw *big.Int
	if log.Data != nil && len(*log.Data) >= 32 {
		amountRaw = new(big.Int).SetBytes((*log.Data)[:32])
	} else {
		amountRaw = new(big.Int)
	}

	symbol := "OTHER"
	if s, ok := KnownTokens[*log.Address]; ok {
		symbol = s
	}

	r := core.NewRecord(mustCollection(app, "transfers"))
	r.Set("tx_hash", txHash)
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", logIdx)
	r.Set("token_address", log.Address.Hex())
	r.Set("token_symbol", symbol)
	r.Set("from_addr", from.Hex())
	r.Set("to_addr", to.Hex())
	r.Set("amount_raw", amountRaw.String())
	r.Set("amount_human", stablecoinHuman(amountRaw))

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save transfer", "tx", txHash, "error", err)
		return nil, fmt.Errorf("save transfer %s/%d: %w", txHash, logIdx, err)
	}

	app.Logger().Debug("Transfer",
		"token", symbol,
		"amount", stablecoinHuman(amountRaw),
		"from", from.Hex()[:10]+"…",
		"to", to.Hex()[:10]+"…",
		"tx", txHash[:10]+"…",
	)

	// update wallet graph edge
	if err := upsertWalletEdge(app, from.Hex(), to.Hex(), amountRaw, log.BlockNumber); err != nil {
		return nil, err
	}

	return amountRaw, nil
}

func saveCCTPEvent(app core.App, log *types.Log, protocol, eventType string) error {
	if log.TransactionHash == nil || log.LogIndex == nil {
		return nil
	}
	existing, err := app.FindRecordsByFilter("crosschain_events",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": log.TransactionHash.Hex(), "i": *log.LogIndex})
	if err != nil {
		return fmt.Errorf("find cctp event %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}
	if len(existing) > 0 {
		return nil
	}

	r := core.NewRecord(mustCollection(app, "crosschain_events"))
	r.Set("tx_hash", log.TransactionHash.Hex())
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", *log.LogIndex)
	r.Set("protocol", protocol)
	r.Set("event_type", eventType)
	r.Set("destination_domain", 26) // Arc testnet domain

	// DepositForBurn: topic1 = nonce, topic2 = burnToken, data contains amount+recipient
	// Full ABI decode can be added once the exact ABI is confirmed.

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save cctp event", "error", err)
		return fmt.Errorf("save cctp event %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}

	app.Logger().Debug("CCTP event",
		"type", eventType,
		"tx", log.TransactionHash.Hex()[:10]+"…",
		"block", func() uint64 {
			if log.BlockNumber != nil {
				return log.BlockNumber.Uint64()
			}
			return 0
		}(),
	)
	return nil
}

func saveGatewayEvent(app core.App, log *types.Log) error {
	if log.TransactionHash == nil || log.LogIndex == nil {
		return nil
	}
	existing, err := app.FindRecordsByFilter("crosschain_events",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": log.TransactionHash.Hex(), "i": *log.LogIndex})
	if err != nil {
		return fmt.Errorf("find gateway event %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}
	if len(existing) > 0 {
		return nil
	}

	r := core.NewRecord(mustCollection(app, "crosschain_events"))
	r.Set("tx_hash", log.TransactionHash.Hex())
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", *log.LogIndex)
	r.Set("protocol", "gateway")
	r.Set("event_type", "deposit") // refine once Gateway ABI is confirmed

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save gateway event", "error", err)
		return fmt.Errorf("save gateway event %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}

	app.Logger().Debug("Gateway event", "tx", log.TransactionHash.Hex()[:10]+"…")
	return nil
}

func saveFxEvent(app core.App, log *types.Log) error {
	if log.TransactionHash == nil || log.LogIndex == nil {
		return nil
	}
	existing, err := app.FindRecordsByFilter("fx_swaps",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": log.TransactionHash.Hex(), "i": *log.LogIndex})
	if err != nil {
		return fmt.Errorf("find fx event %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}
	if len(existing) > 0 {
		return nil
	}

	r := core.NewRecord(mustCollection(app, "fx_swaps"))
	r.Set("tx_hash", log.TransactionHash.Hex())
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", *log.LogIndex)
	r.Set("status", "created") // refine once FxEscrow ABI is confirmed

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save fx event", "error", err)
		return fmt.Errorf("save fx event %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}

	app.Logger().Debug("FX swap event", "tx", log.TransactionHash.Hex()[:10]+"…")
	return nil
}

// upsertWalletEdge increments the edge (from→to) in the wallet graph.
func upsertWalletEdge(app core.App, from, to string, amount *big.Int, blockNumber *big.Int) error {
	existing, err := app.FindRecordsByFilter("wallet_edges",
		"from_wallet = {:f} && to_wallet = {:t}", "", 1, 0,
		map[string]any{"f": from, "t": to})
	if err != nil {
		return fmt.Errorf("find wallet edge %s -> %s: %w", from, to, err)
	}

	var r *core.Record
	if len(existing) > 0 {
		r = existing[0]
		prevTotal, _ := new(big.Int).SetString(r.GetString("total_usdc"), 10)
		if prevTotal == nil {
			prevTotal = new(big.Int)
		}
		newTotal := new(big.Int).Add(prevTotal, amount)
		r.Set("total_usdc", newTotal.String())
		r.Set("tx_count", r.GetInt("tx_count")+1)
		if blockNumber != nil {
			r.Set("last_seen_block", blockNumber.Uint64())
		}
	} else {
		r = core.NewRecord(mustCollection(app, "wallet_edges"))
		r.Set("from_wallet", from)
		r.Set("to_wallet", to)
		r.Set("total_usdc", amount.String())
		r.Set("tx_count", 1)
		if blockNumber != nil {
			r.Set("last_seen_block", blockNumber.Uint64())
		}
	}

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to upsert wallet edge", "from", from, "to", to, "error", err)
		return fmt.Errorf("save wallet edge %s -> %s: %w", from, to, err)
	}
	return nil
}

// ── Cursor management ─────────────────────────────────────────────────────────

// arcBlocksPerDay is a conservative estimate based on Arc's ~1 second block time.
const arcBlocksPerDay = uint64(86_400)

// arcCatchupLookback is how far back we start on a fresh DB.
// 1 hour gives enough context data without blowing the free-tier rate limit
// during catch-up (18 batches of 200 blocks vs 3024 for 7 days).
const arcCatchupLookback = uint64(3_600)

// resolveStartBlock returns the block to stream from.
// If a cursor exists in the DB we resume from there. On a fresh start we fetch
// the current chain tip and walk back 7 days so we don't blow the free-tier
// Envio soft limits (100k events / 5GB) by replaying the entire chain history.
func resolveStartBlock(ctx context.Context, app core.App, client interface {
	GetHeight(context.Context) (*big.Int, error)
}) uint64 {
	last := getLastIndexedBlock(app)
	if last > 0 {
		app.Logger().Info("Resuming indexer from saved cursor", "from_block", last)
		return last
	}

	height, err := client.GetHeight(ctx)
	if err != nil || height == nil {
		app.Logger().Warn("Could not fetch chain height, starting from block 0", "error", err)
		return 0
	}

	tip := height.Uint64()
	lookback := arcCatchupLookback
	start := uint64(0)
	if tip > lookback {
		start = tip - lookback
	}

	app.Logger().Info("Fresh start — beginning 1 hour behind chain tip",
		"tip", tip, "from_block", start, "lookback_blocks", lookback)
	return start
}

func getLastIndexedBlock(app core.App) uint64 {
	records, err := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if err != nil || len(records) == 0 {
		return 0
	}
	val, _ := strconv.ParseUint(records[0].GetString("value"), 10, 64)
	return val
}

func setLastIndexedBlock(app core.App, block uint64) error {
	records, err := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if err != nil {
		return fmt.Errorf("find lastBlock cursor: %w", err)
	}

	var r *core.Record
	if len(records) > 0 {
		r = records[0]
	} else {
		c := mustCollection(app, "indexer_meta")
		r = core.NewRecord(c)
		r.Set("key", "lastBlock")
	}
	r.Set("value", strconv.FormatUint(block, 10))
	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to persist lastBlock cursor", "error", err)
		return fmt.Errorf("save lastBlock cursor %d: %w", block, err)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// weiToUSDC converts a fee in native USDC wei (18 decimals) to a human-readable string.
func weiToUSDC(wei *big.Int) string {
	if wei == nil || wei.Sign() == 0 {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	quot := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(divisor))
	return quot.Text('f', 8)
}

// stablecoinHuman converts an ERC-20 stablecoin amount (6 decimals) to a human-readable string.
func stablecoinHuman(raw *big.Int) string {
	if raw == nil || raw.Sign() == 0 {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	quot := new(big.Float).Quo(new(big.Float).SetInt(raw), new(big.Float).SetInt(divisor))
	return quot.Text('f', 6)
}

// mustCollection fetches a collection by name and panics if missing — collections are
// registered at startup so absence here is a programming error.
func mustCollection(app core.App, name string) *core.Collection {
	c, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		panic(fmt.Sprintf("collection %q not found: %v", name, err))
	}
	return c
}

// addressFromTopic extracts an Ethereum address from a 32-byte topic (last 20 bytes).
func addressFromTopic(h *common.Hash) string {
	if h == nil {
		return ""
	}
	return common.BytesToAddress(h.Bytes()[12:]).Hex()
}

// stripQuotes is a no-op helper kept for clarity when working with string values.
func stripQuotes(s string) string {
	return strings.Trim(s, `"`)
}
