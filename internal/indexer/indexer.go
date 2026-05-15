package indexer

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

// ── Indexer entry point ───────────────────────────────────────────────────────

func StartIndexer(app core.App) {
	const startupDelay = 10 * time.Second
	log.Printf("[indexer] scheduled Arcadia HyperSync indexer startup in %s", startupDelay)
	utils.SeedKnownTokens()
	go func() {
		time.Sleep(startupDelay)
		log.Println("[indexer] starting Arcadia HyperSync indexer")
		attempt := 0
		for {
			attempt++
			log.Printf("[indexer] run attempt #%d", attempt)
			recordIndexerEvent(app, "info", "run_start", "starting indexer run", indexerEventFields{"attempt": attempt})
			if err := runIndexer(app, attempt); err != nil {
				msg := err.Error()
				if strings.Contains(msg, "429") {
					log.Printf("[indexer] rate-limited (429) — waiting 30s before retry (attempt #%d)", attempt)
					recordIndexerEvent(app, "warn", "rate_limited", "HyperSync returned 429; backing off before retry", indexerEventFields{"attempt": attempt, "error": err})
					time.Sleep(30 * time.Second)
				} else {
					log.Printf("[indexer] crashed: %v — restarting in 5s (attempt #%d)", err, attempt)
					recordIndexerEvent(app, "error", "run_error", "indexer run failed; restarting", indexerEventFields{"attempt": attempt, "error": err})
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
}

// ── Indexer event logging ────────────────────────────────────────────────────

type indexerEventFields map[string]any

func recordIndexerEvent(app core.App, level, event, message string, fields indexerEventFields) {
	c, err := app.FindCollectionByNameOrId("indexer_events")
	if err != nil {
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
		log.Printf("[indexer] failed to persist indexer event %q: %v", event, err)
	}
}

// ── Chain tip & heartbeat ────────────────────────────────────────────────────

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
		if persist {
			recordIndexerEvent(app, "warn", "heartbeat", "indexer heartbeat tip check failed", indexerEventFields{"attempt": attempt, "batch": batchCount, "block": currentBlock, "error": tipErr})
		}
		return
	}

	if processingBatch > 0 {
		log.Printf("[indexer] heartbeat | processing batch #%d for %s | block %d | tip %d | lag %d | completed_batches=%d", processingBatch, processingFor, currentBlock, tip, lag, batchCount)
	} else {
		log.Printf("[indexer] heartbeat | idle %s | block %d | tip %d | lag %d | batches=%d", idleFor, currentBlock, tip, lag, batchCount)
	}
	utils.SetMetaValue(app, "chainTip", strconv.FormatUint(tip, 10))
	utils.SetMetaValue(app, "lagBlocks", strconv.FormatUint(lag, 10))
	if persist {
		recordIndexerEvent(app, "info", "heartbeat", "indexer heartbeat", indexerEventFields{"attempt": attempt, "batch": batchCount, "block": currentBlock, "tip": tip, "lag": lag})
	}
}

// ── Cursor management ─────────────────────────────────────────────────────────

const arcCatchupLookback = uint64(3_600)

func resolveStartBlock(ctx context.Context, app core.App, client interface {
	GetHeight(context.Context) (*big.Int, error)
}) uint64 {
	last := utils.GetLastIndexedBlock(app)
	if last > 0 {
		log.Printf("[indexer] resuming from saved cursor %d", last)
		return last
	}

	height, err := client.GetHeight(ctx)
	if err != nil || height == nil {
		log.Printf("[indexer] could not fetch chain height, starting from block 0: %v", err)
		return 0
	}

	tip := height.Uint64()
	lookback := arcCatchupLookback
	start := uint64(0)
	if tip > lookback {
		start = tip - lookback
	}

	log.Printf("[indexer] fresh start | tip=%d from_block=%d lookback_blocks=%d", tip, start, lookback)
	return start
}
