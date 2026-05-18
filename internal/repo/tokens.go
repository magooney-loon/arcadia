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

// TokenSummary holds aggregate stats across the whole token_analytics table
// (optionally filtered by the same search string used by ListTokens).
type TokenSummary struct {
	Total          int   `db:"total"`
	TotalTransfers int64 `db:"total_transfers"`
	Active         int   `db:"active"`
	Failed         int   `db:"failed"`
}

// TokensSummary returns aggregate counts/sums across all matching token rows.
func TokensSummary(app core.App, search string) (TokenSummary, error) {
	sql := `SELECT
		COUNT(*) AS total,
		COALESCE(SUM(transfer_count), 0) AS total_transfers,
		SUM(CASE WHEN lookup_failed THEN 0 ELSE 1 END) AS active,
		SUM(CASE WHEN lookup_failed THEN 1 ELSE 0 END) AS failed
	FROM token_analytics`
	params := dbx.Params{}
	if s := strings.ToLower(strings.TrimSpace(search)); s != "" {
		sql += " WHERE LOWER(symbol) LIKE {:s} OR LOWER(name) LIKE {:s} OR LOWER(token_address) LIKE {:s}"
		params["s"] = "%" + s + "%"
	}
	var out TokenSummary
	if err := app.DB().NewQuery(sql).Bind(params).One(&out); err != nil {
		return out, fmt.Errorf("token summary: %w", err)
	}
	return out, nil
}
