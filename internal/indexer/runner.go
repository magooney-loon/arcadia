package indexer

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	hypersyncgo "github.com/enviodev/hypersync-client-go"
	"github.com/enviodev/hypersync-client-go/options"
	hsutils "github.com/enviodev/hypersync-client-go/utils"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

func runIndexer(ctx context.Context, app core.App, attempt int) error {

	apiToken := utils.EnvioAPIToken()
	if apiToken == "" {
		return fmt.Errorf("ENVIO_API_TOKEN not set — get one at envio.dev")
	}

	rpc := utils.NextRPCURL()
	log.Printf("[indexer] connecting — hypersync: %s  rpc: %s", utils.ArcHyperSyncURL, rpc)

	hyper, err := hypersyncgo.NewHyper(ctx, options.Options{
		Blockchains: []options.Node{
			{
				Type:           hsutils.EthereumNetwork,
				NetworkId:      utils.ArcNetworkID,
				Endpoint:       utils.ArcHyperSyncURL,
				RpcEndpoint:    rpc,
				ApiToken:       apiToken,
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

	client, ok := hyper.GetClient(utils.ArcNetworkID)
	if !ok {
		return fmt.Errorf("arc client not found in hyper")
	}

	fromBlock := resolveStartBlock(ctx, app, client)
	log.Printf("[indexer] streaming from block %d", fromBlock)
	recordIndexerEvent(app, "info", "stream_start", "streaming from saved start block", indexerEventFields{"attempt": attempt, "block": fromBlock})

	var currentBlock atomic.Uint64
	currentBlock.Store(fromBlock)
	var completedBatches atomic.Uint64
	var lastBatchAtUnixNano atomic.Int64
	lastBatchAtUnixNano.Store(time.Now().UnixNano())
	var processingBatch atomic.Uint64
	var processingStartedAtUnixNano atomic.Int64
	var lastPersistedHeartbeatUnixNano atomic.Int64

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
				persistHeartbeat := false
				if activeBatch == 0 {
					lastPersisted := time.Unix(0, lastPersistedHeartbeatUnixNano.Load())
					if lastPersisted.IsZero() || time.Since(lastPersisted) >= time.Minute {
						persistHeartbeat = true
						lastPersistedHeartbeatUnixNano.Store(time.Now().UnixNano())
					}
				}
				logIndexerHeartbeat(ctx, app, client, attempt, completedBatches.Load(), currentBlock.Load(), lastBatchAt, activeBatch, processingStartedAt, persistHeartbeat)
			case <-heartbeatStop:
				return
			}
		}
	}()

	log.Println("[indexer] explicit GetArrow loop running")

	for {
		if ctx.Err() != nil {
			return nil
		}
		start := currentBlock.Load()
		tipCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		tip, tipErr := getChainTip(tipCtx, client)
		cancel()
		if tipErr != nil {
			return fmt.Errorf("fetch chain tip before batch: %w", tipErr)
		}
		if start >= tip {
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return nil
			}
			continue
		}

		toBlock := start + 200
		if toBlock <= start || toBlock > tip+1 {
			toBlock = tip + 1
		}

		nextBatch := completedBatches.Load() + 1
		batchStart := time.Now()
		processingBatch.Store(nextBatch)
		processingStartedAtUnixNano.Store(batchStart.UnixNano())
		log.Printf("[indexer] batch #%d fetching | range=[%d,%d) | tip=%d | lag=%d", nextBatch, start, toBlock, tip, tip-start)

		res, err := getIndexerBatch(ctx, client, newIndexerQuery(start, toBlock))
		if err != nil {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			return fmt.Errorf("fetch batch range [%d,%d): %w", start, toBlock, err)
		}

		nextBlock := "<nil>"
		if res.NextBlock != nil {
			nextBlock = res.NextBlock.String()
		}
		log.Printf("[indexer] batch #%d processing | current block %d | next_block=%s | blocks=%d txs=%d logs=%d",
			nextBatch, start, nextBlock, len(res.Data.Blocks), len(res.Data.Transactions), len(res.Data.Logs))
		// batch_start dropped — batch_done carries the same fields and twice the
		// event-write pressure was hitting the same SQLite writer as the batch.
		if err := processBatch(app, res); err != nil {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			log.Printf("[indexer] batch processing error: %v", err)
			recordIndexerEvent(app, "error", "batch_error", "batch failed; cursor was not advanced", indexerEventFields{
				"attempt":      attempt,
				"batch":        nextBatch,
				"block":        start,
				"blocks":       len(res.Data.Blocks),
				"transactions": len(res.Data.Transactions),
				"logs":         len(res.Data.Logs),
				"error":        err,
			})
			return err
		}

		if res.NextBlock == nil {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			return fmt.Errorf("batch range [%d,%d) returned nil next_block", start, toBlock)
		}
		next := res.NextBlock.Uint64()
		if next <= start {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			return fmt.Errorf("batch range [%d,%d) did not advance next_block: %d", start, toBlock, next)
		}

		currentBlock.Store(next)
		if err := utils.SetLastIndexedBlock(app, next); err != nil {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			recordIndexerEvent(app, "error", "cursor_error", "failed to persist cursor after batch", indexerEventFields{"attempt": attempt, "batch": nextBatch, "block": next, "error": err})
			return err
		}

		batchCount := completedBatches.Add(1)
		elapsed := time.Since(batchStart).Milliseconds()
		lastBatchAtUnixNano.Store(time.Now().UnixNano())
		processingBatch.Store(0)
		processingStartedAtUnixNano.Store(0)
		log.Printf("[indexer] batch #%d | block %d | range=[%d,%d) | blocks=%d txs=%d logs=%d | %dms",
			batchCount, currentBlock.Load(), start, toBlock,
			len(res.Data.Blocks), len(res.Data.Transactions), len(res.Data.Logs),
			elapsed,
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

		// Adaptive pacing: sprint when behind, ease off at the tip.
		// Arc averages ~380 ms/block; the indexer fetches 200-block batches.
		newTip := next
		remainingLag := uint64(0)
		if tip > newTip {
			remainingLag = tip - newTip
		}
		var sleepDur time.Duration
		switch {
		case remainingLag >= 200:
			// sprint — no sleep
		case remainingLag >= 50:
			sleepDur = 100 * time.Millisecond
		default:
			sleepDur = 400 * time.Millisecond
		}
		if sleepDur > 0 {
			select {
			case <-time.After(sleepDur):
			case <-ctx.Done():
				return nil
			}
		}
	}
}
