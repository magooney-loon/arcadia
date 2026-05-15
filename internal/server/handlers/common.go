package handlers

// API_SOURCE

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

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
	rows, err := app.FindRecordsByFilter("analytics_snapshots",
		"window = {:w}", "-snapshot_at", 1, 0, map[string]any{"w": window})
	if err != nil || len(rows) == 0 {
		return nil, false
	}
	return rows[0], true
}
