package handlers

// API_SOURCE

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
	"arcadia/internal/server/cache"
	"arcadia/internal/utils"
)

// cachedCountTTL is how long an unfiltered COUNT(*) result is reused before
// being recomputed. Lists grow monotonically so a stale-by-a-minute total is
// fine and dramatically reduces full-table scans under dashboard polling.
const cachedCountTTL = 60 * time.Second

// cachedCount returns the count for an unfiltered list endpoint, using the
// shared in-memory cache. On miss the supplied fetch runs and the result is
// cached for cachedCountTTL. Errors from fetch fall through as 0.
func cachedCount(key string, fetch func() (int, error)) int {
	if cached, ok := cache.Default.Get(key); ok {
		if n, ok := cached.(int); ok {
			return n
		}
	}
	n, _ := fetch()
	cache.Default.Set(key, n, cachedCountTTL)
	return n
}

// ── helpers ───────────────────────────────────────────────────────────────────

func qp(c *core.RequestEvent, key, fallback string) string {
	val := c.Request.URL.Query().Get(key)
	if val == "" {
		return fallback
	}
	return val
}

func limitOffset(c *core.RequestEvent) (int, int) {
	limit, _ := strconv.Atoi(qp(c, "limit", "50"))
	offset, _ := strconv.Atoi(qp(c, "offset", "0"))
	if limit > 500 {
		limit = 500
	}
	return limit, offset
}

// recordsToMaps converts a slice of records to public-exported maps.
// Equivalent to repo.RecordMaps; kept here for convenience since many handlers call it.
func recordsToMaps(records []*core.Record) []map[string]any {
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = r.PublicExport()
	}
	return out
}

// cacheHeaders writes a public Cache-Control header. Use small TTLs for
// indexer-tip data (1–5 s) and longer ones for snapshot-backed endpoints
// (30–60 s) — the frontend's polling rate dominates DB load otherwise.
func cacheHeaders(c *core.RequestEvent, maxAgeSeconds int) {
	c.Response.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeSeconds))
}

// enrichEdgeRecord adds total_usdc_human (stablecoin 6-decimal raw → human string).
func enrichEdgeRecord(r *core.Record) map[string]any {
	m := r.PublicExport()
	if raw := r.GetString("total_usdc"); raw != "" && raw != "0" {
		if n, ok := new(big.Int).SetString(raw, 10); ok {
			m["total_usdc_human"] = utils.StablecoinHuman(n)
		}
	}
	return m
}

// enrichAgentRecord adds human-readable conversions for the raw big.Int fields
// stored on agent records: usdc_spent_fees (wei, 18 dec) and usdc_transferred
// (raw ERC-20 units, 6 dec).
func enrichAgentRecord(r *core.Record) map[string]any {
	m := r.PublicExport()
	if raw := r.GetString("usdc_spent_fees"); raw != "" && raw != "0" {
		if n, ok := new(big.Int).SetString(raw, 10); ok {
			m["usdc_spent_fees_human"] = utils.WeiToUSDC(n)
		}
	}
	if raw := r.GetString("usdc_transferred"); raw != "" && raw != "0" {
		if n, ok := new(big.Int).SetString(raw, 10); ok {
			m["usdc_transferred_human"] = utils.StablecoinHuman(n)
		}
	}
	return m
}

// ── analytics helpers ─────────────────────────────────────────────────────────

func parseUSDC(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// latestSnapshot returns the most recent analytics_snapshots row for the given
// window, or (nil, false) if none exists yet (e.g. fresh install before first job run).
func latestSnapshot(app core.App, window string) (*core.Record, bool) {
	row, err := repo.LatestSnapshot(app, window)
	if err != nil || row == nil {
		return nil, false
	}
	return row, true
}
