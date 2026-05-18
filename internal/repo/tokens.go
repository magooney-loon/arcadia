package repo

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ListTokens returns token analytics records, optionally filtered by search query.
func ListTokens(app core.App, search string, limit, offset int) ([]*core.Record, error) {
	filter, params := buildTokenFilter(search)
	return FindRecords(app, "token_analytics", filter, "-transfer_count", limit, offset, params)
}

// CountTokens returns the total number of token_analytics rows matching the search.
func CountTokens(app core.App, search string) (int, error) {
	if strings.TrimSpace(search) == "" {
		return RowCount(app, "token_analytics")
	}
	q := strings.ToLower(strings.TrimSpace(search))
	var row struct {
		N int `db:"n"`
	}
	err := app.DB().NewQuery(`SELECT COUNT(*) AS n FROM token_analytics
		WHERE LOWER(symbol) LIKE {:s} OR LOWER(name) LIKE {:s} OR LOWER(token_address) LIKE {:s}`).
		Bind(dbx.Params{"s": "%" + q + "%"}).
		One(&row)
	if err != nil {
		return 0, fmt.Errorf("count token_analytics: %w", err)
	}
	return row.N, nil
}

// buildTokenFilter returns a PocketBase filter expression (uses the `~` contains
// operator which PB translates to a case-insensitive LIKE — raw SQL functions
// like LOWER() are not understood by the PB filter parser).
func buildTokenFilter(search string) (string, map[string]any) {
	s := strings.TrimSpace(search)
	if s == "" {
		return "", nil
	}
	return "(symbol ~ {:s} || name ~ {:s} || token_address ~ {:s})",
		map[string]any{"s": s}
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
