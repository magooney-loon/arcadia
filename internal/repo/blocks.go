package repo

import (
	"github.com/pocketbase/pocketbase/core"
)

// ListBlocks returns blocks sorted by number descending with pagination.
func ListBlocks(app core.App, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "blocks", "", "-number", limit, offset)
}

// CountBlocks returns the total number of indexed blocks.
func CountBlocks(app core.App) (int, error) {
	return CountWithFilter(app, "blocks", "", nil)
}

// BlockByNumber returns the block with the given number, or nil if not found.
func BlockByNumber(app core.App, number int64) (*core.Record, error) {
	return LatestRecord(app, "blocks", "number = {:n}", "", map[string]any{"n": number})
}
