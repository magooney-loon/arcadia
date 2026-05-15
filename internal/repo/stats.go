package repo

import (
	"github.com/pocketbase/pocketbase/core"
)

// LatestBlockStats returns the most recent block_stats row, or nil.
func LatestBlockStats(app core.App) (*core.Record, error) {
	return LatestRecord(app, "block_stats", "", "-block_number")
}

// RecentBlockStats returns the N most recent block_stats rows.
func RecentBlockStats(app core.App, n int) ([]*core.Record, error) {
	return FindRecords(app, "block_stats", "", "-block_number", n, 0)
}

// BlockStatsByNumber returns block stats for a specific block number.
func BlockStatsByNumber(app core.App, number int64) (*core.Record, error) {
	return LatestRecord(app, "block_stats", "block_number = {:n}", "", map[string]any{"n": number})
}
