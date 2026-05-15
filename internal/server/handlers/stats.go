package handlers

// API_SOURCE

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
)

// API_DESC Latest live chain stats (TPS, fees, transfer volumes, agent activity)
// API_TAGS Stats
func statsHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 2)
	// latest block_stats row
	rec, err := repo.LatestBlockStats(c.App)
	if err != nil || rec == nil {
		return c.JSON(http.StatusOK, map[string]any{"syncing": true})
	}
	latest := rec.PublicExport()

	// rolling 10-block avg for tps + block_time_ms (stored values may be 0 for old rows)
	recent, _ := repo.RecentBlockStats(c.App, 10)
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

	// indexer cursor
	val, _ := repo.MetaValue(c.App, "lastBlock")
	if val != "" {
		latest["indexed_block"] = val
	}

	return c.JSON(http.StatusOK, latest)
}

// API_DESC Historical block stats for time-series charts (sorted newest first)
// API_TAGS Stats
func blockStatsHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 2)
	limit, offset := limitOffset(c)
	records, err := repo.FindRecords(c.App, "block_stats", "", "-block_number", limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"stats": recordsToMaps(records),
		"count": len(records),
	})
}

// API_DESC Indexer health: lag, error rate, last indexed block
// API_TAGS Stats
func healthHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 2)
	// Single query for all meta keys instead of 3 separate lookups.
	metaMap, _ := repo.AllMeta(c.App)
	lastBlock, _ := strconv.Atoi(metaMap["lastBlock"])
	tip, _ := strconv.Atoi(metaMap["chainTip"])
	lag, _ := strconv.Atoi(metaMap["lagBlocks"])

	since := time.Now().UTC().Add(-time.Hour).Unix()
	errEvents, _ := repo.ErrorEventsSince(c.App, since)

	batches, _ := repo.RecentBatchDones(c.App, since, 20)
	var avgBatchMs float64
	if len(batches) > 0 {
		var total int64
		for _, r := range batches {
			total += int64(r.GetInt("duration_ms"))
		}
		avgBatchMs = float64(total) / float64(len(batches))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"last_indexed_block": lastBlock,
		"chain_tip":          tip,
		"lag_blocks":         lag,
		"syncing":            lag > 10,
		"errors_1h":          len(errEvents),
		"avg_batch_ms":       avgBatchMs,
	})
}
