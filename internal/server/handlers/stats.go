package handlers

// API_SOURCE

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC Latest live chain stats (TPS, fees, transfer volumes, agent activity)
// API_TAGS Stats
func statsHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 2)
	// latest block_stats row
	rows, err := c.App.FindRecordsByFilter("block_stats", "", "-block_number", 1, 0)
	if err != nil || len(rows) == 0 {
		return c.JSON(http.StatusOK, map[string]any{"syncing": true})
	}
	latest := rows[0].PublicExport()

	// rolling 10-block avg for tps + block_time_ms (stored values may be 0 for old rows)
	recent, _ := c.App.FindRecordsByFilter("block_stats", "", "-block_number", 10, 0)
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
	cursor, _ := c.App.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if len(cursor) > 0 {
		latest["indexed_block"] = cursor[0].GetString("value")
	}

	return c.JSON(http.StatusOK, latest)
}

// API_DESC Historical block stats for time-series charts (sorted newest first)
// API_TAGS Stats
func blockStatsHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 2)
	limit, offset := limitOffset(c)
	records, err := c.App.FindRecordsByFilter("block_stats", "", "-block_number", limit, offset)
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
	metaRows, _ := c.App.FindRecordsByFilter("indexer_meta", "key != ''", "", 10, 0)
	metaMap := make(map[string]string, len(metaRows))
	for _, r := range metaRows {
		metaMap[r.GetString("key")] = r.GetString("value")
	}
	lastBlock, _ := strconv.Atoi(metaMap["lastBlock"])
	tip, _ := strconv.Atoi(metaMap["chainTip"])
	lag, _ := strconv.Atoi(metaMap["lagBlocks"])

	since := time.Now().UTC().Add(-time.Hour).Format("2006-01-02 15:04:05.000Z")
	errEvents, _ := c.App.FindRecordsByFilter("indexer_events",
		"level = 'error' && created >= {:since}", "", 500, 0,
		map[string]any{"since": since})

	batches, _ := c.App.FindRecordsByFilter("indexer_events",
		"event = 'batch_done' && created >= {:since}", "-created", 20, 0,
		map[string]any{"since": since})
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
