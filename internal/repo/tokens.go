package repo

import (
	"github.com/pocketbase/pocketbase/core"
)

// ListTokens returns token analytics records, optionally filtered by search query.
func ListTokens(app core.App, search string, limit, offset int) ([]*core.Record, error) {
	if search != "" {
		return FindRecords(app, "token_analytics",
			"(LOWER(symbol) LIKE {:s} OR LOWER(name) LIKE {:s} OR LOWER(token_address) LIKE {:s})",
			"-transfer_count", limit, offset, map[string]any{"s": "%" + search + "%"})
	}
	return FindRecords(app, "token_analytics", "", "-transfer_count", limit, offset)
}

// TokenByAddress returns token analytics for a specific address.
func TokenByAddress(app core.App, addr string) (*core.Record, error) {
	return LatestRecord(app, "token_analytics", "token_address = {:a}", "", map[string]any{"a": addr})
}

// AllTokenAnalytics returns all token analytics records (used for bulk updates).
func AllTokenAnalytics(app core.App) ([]*core.Record, error) {
	return FindRecords(app, "token_analytics", "", "", 0, 0)
}
