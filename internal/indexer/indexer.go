package indexer

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/rpc"
	"arcadia/internal/utils"
)

// ── Indexer entry point ───────────────────────────────────────────────────────

func StartIndexer(app core.App) {
	const startupDelay = 10 * time.Second
	log.Printf("[indexer] scheduled Arcadia HyperSync indexer startup in %s", startupDelay)
	rpc.SeedKnownTokens()
	startIndexerEventWriter(app)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the indexer context on SIGTERM so the in-flight batch can finish
	// cleanly rather than being abandoned mid-transaction.
	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		log.Println("[indexer] shutdown signal received — draining in-flight batch")
		cancel()
		return e.Next()
	})

	go func() {
		defer cancel()
		time.Sleep(startupDelay)
		log.Println("[indexer] starting Arcadia HyperSync indexer")
		attempt := 0
		for {
			if ctx.Err() != nil {
				log.Println("[indexer] context cancelled — shutting down")
				return
			}
			attempt++
			log.Printf("[indexer] run attempt #%d", attempt)
			recordIndexerEvent(app, "info", "run_start", "starting indexer run", indexerEventFields{"attempt": attempt})
			if err := runIndexer(ctx, app, attempt); err != nil {
				if ctx.Err() != nil {
					log.Println("[indexer] stopped after context cancellation")
					return
				}
				msg := err.Error()
				if strings.Contains(msg, "429") {
					log.Printf("[indexer] rate-limited (429) — waiting 30s before retry (attempt #%d)", attempt)
					recordIndexerEvent(app, "warn", "rate_limited", "HyperSync returned 429; backing off before retry", indexerEventFields{"attempt": attempt, "error": err})
					select {
					case <-time.After(30 * time.Second):
					case <-ctx.Done():
						return
					}
				} else {
					log.Printf("[indexer] crashed: %v — restarting in 5s (attempt #%d)", err, attempt)
					recordIndexerEvent(app, "error", "run_error", "indexer run failed; restarting", indexerEventFields{"attempt": attempt, "error": err})
					select {
					case <-time.After(5 * time.Second):
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
}

// ── Indexer event logging ────────────────────────────────────────────────────
//
// Events are buffered through a single-writer goroutine so calls from the hot
// batch loop don't block on SQLite. The channel is sized generously; if it
// fills (writer stuck or shutting down) events are dropped — they're logging,
// not data.

type indexerEventFields map[string]any

type indexerEvent struct {
	level, event, message string
	fields                indexerEventFields
	ts                    int64
}

var (
	indexerEventCh   = make(chan indexerEvent, 256)
	indexerEventOnce sync.Once
)

func startIndexerEventWriter(app core.App) {
	indexerEventOnce.Do(func() {
		go func() {
			for ev := range indexerEventCh {
				writeIndexerEvent(app, ev)
			}
		}()
	})
}

// Field caps mirror the indexer_events collection schema. HyperSync
// 429 responses can carry response bodies well past these, so truncate
// before insert — otherwise Save() rejects the whole row and we lose
// the diagnostic record entirely.
const (
	maxIndexerEventMessage = 500
	maxIndexerEventError   = 1000
)

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func writeIndexerEvent(app core.App, ev indexerEvent) {
	c, err := app.FindCollectionByNameOrId("indexer_events")
	if err != nil {
		return
	}

	r := core.NewRecord(c)
	r.Set("timestamp", ev.ts)
	r.Set("level", ev.level)
	r.Set("event", ev.event)
	r.Set("message", truncate(ev.message, maxIndexerEventMessage))
	for key, val := range ev.fields {
		switch key {
		case "attempt", "batch", "block", "tip", "lag", "duration_ms", "blocks", "transactions", "logs", "error":
			if key == "error" {
				if val != nil {
					r.Set("error", truncate(fmt.Sprint(val), maxIndexerEventError))
				}
				continue
			}
			r.Set(key, val)
		}
	}
	if err := app.Save(r); err != nil {
		log.Printf("[indexer] failed to persist indexer event %q: %v", ev.event, err)
	}
}

func recordIndexerEvent(_ core.App, level, event, message string, fields indexerEventFields) {
	ev := indexerEvent{
		level:   level,
		event:   event,
		message: message,
		fields:  fields,
		ts:      time.Now().Unix(),
	}
	select {
	case indexerEventCh <- ev:
	default:
		// channel full — drop silently to keep the indexer running
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
		// Caught up + no batch in flight = safe window to truncate the WAL.
		// Passive autocheckpoints can't make progress while readers hold open
		// snapshots, so the WAL grows during sprint catch-up. A TRUNCATE
		// checkpoint blocks until readers drain and then resets the WAL file
		// to zero length.
		if lag == 0 {
			maybeTruncateWAL(app)
		}
	}
	_ = utils.SetMetaValue(app, "chainTip", strconv.FormatUint(tip, 10))
	_ = utils.SetMetaValue(app, "lagBlocks", strconv.FormatUint(lag, 10))
	if persist {
		recordIndexerEvent(app, "info", "heartbeat", "indexer heartbeat", indexerEventFields{"attempt": attempt, "batch": batchCount, "block": currentBlock, "tip": tip, "lag": lag})
	}
}

// ── WAL maintenance ──────────────────────────────────────────────────────────

// walCheckpointCooldown caps how often the idle TRUNCATE checkpoint runs.
// The PRAGMA blocks until all readers drain to the head of the WAL, so we
// don't want to issue one on every 15s heartbeat — once per minute is plenty
// to keep the WAL file bounded.
const walCheckpointCooldown = time.Minute

var (
	walCheckpointInflight atomic.Bool
	lastWALCheckpointNano atomic.Int64
)

// maybeTruncateWAL fires a `PRAGMA wal_checkpoint(TRUNCATE)` in a goroutine
// when (a) no other checkpoint is in flight and (b) the cooldown has elapsed.
// Called from the indexer heartbeat when lag==0 and no batch is processing.
func maybeTruncateWAL(app core.App) {
	last := lastWALCheckpointNano.Load()
	if last > 0 && time.Since(time.Unix(0, last)) < walCheckpointCooldown {
		return
	}
	if !walCheckpointInflight.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer walCheckpointInflight.Store(false)
		start := time.Now()
		var busy, logPages, checkpointed int
		// PRAGMA wal_checkpoint(TRUNCATE) returns one row: (busy, log, ckpt).
		// busy==1 means a reader/writer prevented full progress; that's
		// informational, not an error.
		err := app.DB().NewQuery("PRAGMA wal_checkpoint(TRUNCATE)").Row(&busy, &logPages, &checkpointed)
		elapsed := time.Since(start)
		lastWALCheckpointNano.Store(time.Now().UnixNano())
		if err != nil {
			log.Printf("[indexer] wal_checkpoint(TRUNCATE) failed in %s: %v", elapsed, err)
			return
		}
		log.Printf("[indexer] wal_checkpoint(TRUNCATE) busy=%d log_pages=%d checkpointed=%d in %s", busy, logPages, checkpointed, elapsed)
	}()
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
