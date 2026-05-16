package indexer

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	hypersyncgo "github.com/enviodev/hypersync-client-go"
	"github.com/enviodev/hypersync-client-go/options"
	"github.com/enviodev/hypersync-client-go/types"
	hsutils "github.com/enviodev/hypersync-client-go/utils"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/chain"
	"arcadia/internal/server/realtime"
	"arcadia/internal/utils"
)

// fetchResult carries one HyperSync batch fetch outcome.
// res == nil means the fetch found no new blocks (at chain tip).
type fetchResult struct {
	res     *types.QueryResponse
	tip     uint64
	start   uint64
	toBlock uint64
	err     error
	atTip   bool
}

func runIndexer(ctx context.Context, app core.App, attempt int) error {
	apiToken := chain.EnvioAPIToken()
	if apiToken == "" {
		return fmt.Errorf("ENVIO_API_TOKEN not set — get one at envio.dev")
	}

	rpc := chain.NextRPCURL()
	log.Printf("[indexer] connecting — hypersync: %s  rpc: %s", chain.ArcHyperSyncURL, rpc)

	hyper, err := hypersyncgo.NewHyper(ctx, options.Options{
		Blockchains: []options.Node{
			{
				Type:           hsutils.EthereumNetwork,
				NetworkId:      chain.ArcNetworkID,
				Endpoint:       chain.ArcHyperSyncURL,
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

	client, ok := hyper.GetClient(chain.ArcNetworkID)
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

	// startPrefetch kicks off an async tip-check + batch fetch. Returns a
	// buffered channel that yields exactly one result. The goroutine is
	// automatically cancelled via ctx when runIndexer returns.
	startPrefetch := func(start uint64) <-chan fetchResult {
		ch := make(chan fetchResult, 1)
		go func() {
			tipCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			tip, err := getChainTip(tipCtx, client)
			cancel()
			if err != nil {
				ch <- fetchResult{err: fmt.Errorf("prefetch tip: %w", err)}
				return
			}
			if start >= tip {
				ch <- fetchResult{tip: tip, start: start, atTip: true}
				return
			}
			// 50 blocks per batch keeps the SQLite write transaction short
			// (typically <200ms) so API reads aren't starved.
			toBlock := start + 50
			if toBlock > tip+1 {
				toBlock = tip + 1
			}
			res, err := getIndexerBatch(ctx, client, newIndexerQuery(start, toBlock))
			ch <- fetchResult{res: res, tip: tip, start: start, toBlock: toBlock, err: err}
		}()
		return ch
	}

	log.Println("[indexer] prefetch loop running")

	// Kick off the first fetch before entering the loop.
	pending := startPrefetch(fromBlock)

	for {
		if ctx.Err() != nil {
			return nil
		}

		// Wait for the in-flight fetch.
		var fr fetchResult
		select {
		case fr = <-pending:
		case <-ctx.Done():
			return nil
		}

		if fr.err != nil {
			return fr.err
		}

		if fr.atTip {
			// Nothing to process — wait for a new block then re-fetch.
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return nil
			}
			pending = startPrefetch(fr.start)
			continue
		}

		res := fr.res
		start, toBlock, tip := fr.start, fr.toBlock, fr.tip

		if res == nil {
			return fmt.Errorf("batch range [%d,%d) returned nil response", start, toBlock)
		}
		if res.NextBlock == nil {
			return fmt.Errorf("batch range [%d,%d) returned nil next_block", start, toBlock)
		}
		next := res.NextBlock.Uint64()
		if next <= start {
			return fmt.Errorf("batch range [%d,%d) did not advance next_block: %d", start, toBlock, next)
		}

		remainingLag := uint64(0)
		if tip > next {
			remainingLag = tip - next
		}

		// ── Start prefetch of batch N+1 BEFORE processing batch N ────────────
		// The goroutine runs concurrently with processBatch below, overlapping
		// the WAN round-trip with the SQLite write time.
		pending = startPrefetch(next)

		nextBatch := completedBatches.Load() + 1
		batchStart := time.Now()
		processingBatch.Store(nextBatch)
		processingStartedAtUnixNano.Store(batchStart.UnixNano())
		log.Printf("[indexer] batch #%d processing | block %d | next_block=%d | blocks=%d txs=%d logs=%d | lag=%d",
			nextBatch, start, next,
			len(res.Data.Blocks), len(res.Data.Transactions), len(res.Data.Logs), remainingLag)

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
			len(res.Data.Blocks), len(res.Data.Transactions), len(res.Data.Logs), elapsed)
		recordIndexerEvent(app, "info", "batch_done", "finished processing indexer batch", indexerEventFields{
			"attempt":      attempt,
			"batch":        batchCount,
			"block":        currentBlock.Load(),
			"duration_ms":  elapsed,
			"blocks":       len(res.Data.Blocks),
			"transactions": len(res.Data.Transactions),
			"logs":         len(res.Data.Logs),
		})

		// Push live dashboard payload to SSE subscribers. Fire-and-forget
		// so it never blocks the indexer loop; the broadcaster also self-
		// throttles to ~1Hz when catching up.
		go realtime.BroadcastIndexerUpdate(app)

		// Adaptive pacing: sprint when behind, ease off near the tip.
		// The prefetch goroutine is already running — the sleep here just
		// throttles throughput when caught up; it overlaps with the fetch.
		// When next >= tip (remainingLag == 0) the at-tip branch handles pacing.
		if next < tip {
			var sleepDur time.Duration
			switch {
			case remainingLag >= 500:
				// sprint — no sleep; let the prefetch win the race
			case remainingLag >= 100:
				sleepDur = 50 * time.Millisecond
			default:
				sleepDur = 200 * time.Millisecond
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
}
