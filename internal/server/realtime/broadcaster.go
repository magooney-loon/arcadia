package realtime

import (
	"encoding/json"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
	"arcadia/internal/server/cache"
)

// minIndexerInterval throttles indexer broadcasts when the indexer is
// catching up (batches every ~400ms). Frontends only need ~1Hz updates.
const minIndexerInterval = time.Second

var lastIndexerBroadcast atomic.Int64

// indexerTopic carries the small per-batch payload every tab needs:
// header stats, health pills, latest blocks/transactions lists.
const indexerTopic = "indexer"

// chartsTopic carries the 200-row block_stats series used only by the
// overview page's charts. Kept separate so tabs that don't render
// charts don't pay the ~30KB-per-tick cost.
const chartsTopic = "charts"

// analyticsTopic carries snapshot-derived aggregates per window.
const analyticsTopic = "analytics"

// BroadcastIndexerUpdate fans out both the small `indexer` payload and
// the larger `charts` payload to whichever clients are subscribed.
// Also populates the in-memory REST cache so API handlers can serve
// responses without hitting SQLite.
func BroadcastIndexerUpdate(app core.App) {
	now := time.Now().UnixMilli()
	last := lastIndexerBroadcast.Load()
	if now-last < minIndexerInterval.Milliseconds() {
		return
	}
	if !lastIndexerBroadcast.CompareAndSwap(last, now) {
		return
	}

	// Always build and cache payloads — even if no SSE subscribers,
	// REST handlers need the cache.
	indexerPayload := buildIndexerPayload(app)

	// Cache the individual components for REST handlers.
	// TTL is generous: the next broadcast (≤1s) will overwrite.
	const ttl = 5 * time.Second
	if s, ok := indexerPayload["stats"]; ok {
		cache.Default.Set("stats", s, ttl)
	}
	if h, ok := indexerPayload["health"]; ok {
		cache.Default.Set("health", h, ttl)
	}
	if b, ok := indexerPayload["blocks"]; ok {
		cache.Default.Set("blocks:10", b, ttl)
	}
	if t, ok := indexerPayload["transactions"]; ok {
		cache.Default.Set("transactions:10", t, ttl)
	}

	// Charts payload
	chartsData := map[string]any{
		"block_stats": buildBlockStatsList(app, 200),
	}
	cache.Default.Set("block_stats:200", chartsData, ttl)

	// Also cache common block/tx list sizes used by list pages.
	cache.Default.Set("blocks:50", buildBlocksList(app, 50), ttl)
	cache.Default.Set("transactions:100", buildTransactionsList(app, 100), ttl)
	cache.Default.Set("block_stats:50", buildBlockStatsList(app, 50), ttl)

	// Send to SSE subscribers.
	_ = Broadcast(app, indexerTopic, indexerPayload)
	_ = Broadcast(app, chartsTopic, chartsData)
}

// BroadcastAnalyticsUpdate fires after a snapshot job finishes for the
// given window. Also caches the payloads for REST handlers.
func BroadcastAnalyticsUpdate(app core.App, window string) {
	payload := buildAnalyticsPayload(app, window)

	const ttl = 30 * time.Second
	if o, ok := payload["overview"]; ok {
		cache.Default.Set("analytics:overview:"+window, o, ttl)
	}
	if bf, ok := payload["bridge_flow"]; ok {
		cache.Default.Set("analytics:bridge_flow:"+window, bf, ttl)
	}
	if v, ok := payload["volume"]; ok {
		cache.Default.Set("analytics:volume:"+window, v, ttl)
	}

	_ = Broadcast(app, analyticsTopic, payload)
}

// ── payload builders ──────────────────────────────────────────────────────────

func buildIndexerPayload(app core.App) map[string]any {
	return map[string]any{
		"stats":        buildStats(app),
		"health":       buildHealth(app),
		"blocks":       buildBlocksList(app, 10),
		"transactions": buildTransactionsList(app, 10),
	}
}

// buildStats mirrors handlers.statsHandler: latest block_stats + rolling
// 10-block avg + indexer cursor.
func buildStats(app core.App) map[string]any {
	rec, err := repo.LatestBlockStats(app)
	if err != nil || rec == nil {
		return map[string]any{"syncing": true}
	}
	latest := rec.PublicExport()

	recent, _ := repo.RecentBlockStats(app, 10)
	if len(recent) >= 2 {
		var totalTxs, totalBms int64
		var count int
		for _, r := range recent {
			bms := r.GetInt("block_time_ms")
			if bms > 0 {
				totalTxs += int64(r.GetInt("tx_count"))
				totalBms += int64(bms)
				count++
			}
		}
		if count > 0 && totalBms > 0 {
			avgBms := totalBms / int64(count)
			latest["block_time_ms"] = avgBms
			latest["tps"] = float64(totalTxs) / float64(count) / (float64(avgBms) / 1000.0)
		}
	}

	if val, _ := repo.MetaValue(app, "lastBlock"); val != "" {
		latest["indexed_block"] = val
	}
	return latest
}

// buildHealth mirrors handlers.healthHandler.
func buildHealth(app core.App) map[string]any {
	metaMap, _ := repo.AllMeta(app)
	lastBlock, _ := strconv.Atoi(metaMap["lastBlock"])
	tip, _ := strconv.Atoi(metaMap["chainTip"])
	lag, _ := strconv.Atoi(metaMap["lagBlocks"])

	since := time.Now().UTC().Add(-time.Hour).Unix()
	errEvents, _ := repo.ErrorEventsSince(app, since)

	batches, _ := repo.RecentBatchDones(app, since, 20)
	var avgBatchMs float64
	if len(batches) > 0 {
		var total int64
		for _, r := range batches {
			total += int64(r.GetInt("duration_ms"))
		}
		avgBatchMs = float64(total) / float64(len(batches))
	}

	return map[string]any{
		"last_indexed_block": lastBlock,
		"chain_tip":          tip,
		"lag_blocks":         lag,
		"syncing":            lag > 10,
		"errors_1h":          len(errEvents),
		"avg_batch_ms":       avgBatchMs,
	}
}

func buildBlocksList(app core.App, limit int) map[string]any {
	records, err := repo.ListBlocks(app, limit, 0)
	if err != nil {
		return map[string]any{"blocks": []any{}, "count": 0}
	}
	return map[string]any{"blocks": repo.RecordMaps(records), "count": len(records)}
}

func buildTransactionsList(app core.App, limit int) map[string]any {
	records, err := repo.ListTransactions(app, repo.TransactionFilter{}, limit, 0)
	if err != nil {
		return map[string]any{"transactions": []any{}, "count": 0}
	}
	return map[string]any{"transactions": repo.RecordMaps(records), "count": len(records)}
}

func buildBlockStatsList(app core.App, limit int) map[string]any {
	records, err := repo.FindRecords(app, "block_stats", "", "-block_number", limit, 0)
	if err != nil {
		return map[string]any{"stats": []any{}, "count": 0}
	}
	return map[string]any{"stats": repo.RecordMaps(records), "count": len(records)}
}

// buildAnalyticsPayload mirrors the three snapshot-backed handlers
// (overview, bridge_flow, volume) for one window.
func buildAnalyticsPayload(app core.App, window string) map[string]any {
	snap, err := repo.LatestSnapshot(app, window)
	if err != nil || snap == nil {
		return map[string]any{"window": window, "syncing": true}
	}

	overview := map[string]any{
		"window":                 window,
		"snapshot_at":            snap.GetInt("snapshot_at"),
		"transfers_count":        snap.GetInt("transfers_count"),
		"transfer_volume":        snap.GetFloat("transfer_volume"),
		"largest_transfer":       snap.GetFloat("largest_transfer"),
		"largest_transfer_block": snap.GetInt("largest_transfer_block"),
		"fees_total":             snap.GetFloat("fees_total"),
		"fee_p50":                snap.GetFloat("fee_p50"),
		"fee_p95":                snap.GetFloat("fee_p95"),
		"failed_tx_ratio":        snap.GetFloat("failed_tx_ratio"),
		"bridge_inbound_vol":     snap.GetFloat("bridge_inbound_vol"),
		"bridge_inbound_count":   snap.GetInt("bridge_inbound_count"),
		"bridge_outbound_vol":    snap.GetFloat("bridge_outbound_vol"),
		"bridge_outbound_count":  snap.GetInt("bridge_outbound_count"),
		"bridge_net_flow":        snap.GetFloat("bridge_net_flow"),
		"agent_count":            snap.GetInt("agent_count"),
	}

	var byChain any
	if s := snap.GetString("bridge_by_chain"); s != "" {
		_ = json.Unmarshal([]byte(s), &byChain)
	}
	bridgeFlow := map[string]any{
		"window":         window,
		"inbound_vol":    snap.GetFloat("bridge_inbound_vol"),
		"inbound_count":  snap.GetInt("bridge_inbound_count"),
		"outbound_vol":   snap.GetFloat("bridge_outbound_vol"),
		"outbound_count": snap.GetInt("bridge_outbound_count"),
		"net_flow":       snap.GetFloat("bridge_net_flow"),
		"by_chain":       byChain,
		"snapshot_at":    snap.GetInt("snapshot_at"),
	}

	type tokenStats struct {
		Volume     float64 `json:"volume"`
		Count      int     `json:"count"`
		WhaleCount int     `json:"whale_count"`
	}
	volume := map[string]any{
		"window":           window,
		"total_transfers":  snap.GetInt("total_transfers"),
		"unique_senders":   snap.GetInt("unique_senders"),
		"unique_receivers": snap.GetInt("unique_receivers"),
		"whale_transfers":  snap.GetInt("whale_transfers"),
		"by_token": map[string]*tokenStats{
			"USDC": {Volume: snap.GetFloat("usdc_volume"), Count: snap.GetInt("usdc_count")},
			"EURC": {Volume: snap.GetFloat("eurc_volume"), Count: snap.GetInt("eurc_count")},
			"USYC": {Volume: snap.GetFloat("usyc_volume"), Count: snap.GetInt("usyc_count")},
		},
		"snapshot_at": snap.GetInt("snapshot_at"),
	}

	return map[string]any{
		"window":      window,
		"overview":    overview,
		"bridge_flow": bridgeFlow,
		"volume":      volume,
	}
}
