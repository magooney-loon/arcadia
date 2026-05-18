package repo

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// ListTokens returns token analytics records, optionally filtered by search query.
func ListTokens(app core.App, search string, limit, offset int) ([]*core.Record, error) {
	filter, params := buildTokenFilter(search)
	return FindRecords(app, "token_analytics", filter, "-transfer_count", limit, offset, params)
}

// CountTokens returns the total number of token_analytics rows matching the search.
func CountTokens(app core.App, search string) (int, error) {
	filter, params := buildTokenFilter(search)
	return CountWithFilter(app, "token_analytics", filter, params)
}

func buildTokenFilter(search string) (string, map[string]any) {
	if search == "" {
		return "", nil
	}
	q := strings.ToLower(strings.TrimSpace(search))
	return "(LOWER(symbol) LIKE {:s} OR LOWER(name) LIKE {:s} OR LOWER(token_address) LIKE {:s})",
		map[string]any{"s": "%" + q + "%"}
}

// TokenByAddress returns token analytics for a specific address.
func TokenByAddress(app core.App, addr string) (*core.Record, error) {
	return LatestRecord(app, "token_analytics", "token_address = {:a}", "", map[string]any{"a": addr})
}

// SearchTokens searches tokens by symbol/name/address, limited to the given number of results.
func SearchTokens(app core.App, q string, limit int) ([]*core.Record, error) {
	filter, params := buildTokenFilter(q)
	return FindRecords(app, "token_analytics", filter, "-transfer_count", limit, 0, params)
}

// AllTokenAnalytics returns all token analytics records (used for bulk updates).
func AllTokenAnalytics(app core.App) ([]*core.Record, error) {
	return FindRecords(app, "token_analytics", "", "", 0, 0)
}
