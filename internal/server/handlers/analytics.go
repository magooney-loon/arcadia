package handlers

// API_SOURCE

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC Fee analytics: P25/P50/P75/P95 percentiles, failed tx ratio, avg block time
// API_TAGS Stats
func analyticsFeesHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 30)
	window := qp(c, "window", "24h")
	snap, ok := latestSnapshot(c.App, window)
	if !ok {
		return c.JSON(http.StatusOK, map[string]any{"window": window, "block_count": 0})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"window":            window,
		"block_count":       snap.GetInt("block_count"),
		"total_fees":        snap.GetFloat("fees_total"),
		"avg_fee_p25":       snap.GetFloat("fee_p25"),
		"avg_fee_p50":       snap.GetFloat("fee_p50"),
		"avg_fee_p75":       snap.GetFloat("fee_p75"),
		"avg_fee_p95":       snap.GetFloat("fee_p95"),
		"avg_block_time_ms": snap.GetFloat("avg_block_time_ms"),
		"failed_tx_ratio":   snap.GetFloat("failed_tx_ratio"),
		"snapshot_at":       snap.GetInt("snapshot_at"),
	})
}

// API_DESC Transfer volume aggregates with whale count and per-token breakdown
// API_TAGS Transfers
func analyticsVolumeHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 30)
	window := qp(c, "window", "24h")
	token := qp(c, "token", "")
	snap, ok := latestSnapshot(c.App, window)
	if !ok {
		return c.JSON(http.StatusOK, map[string]any{"window": window, "syncing": true})
	}

	type tokenStats struct {
		Volume     float64 `json:"volume"`
		Count      int     `json:"count"`
		WhaleCount int     `json:"whale_count"`
	}
	byToken := map[string]*tokenStats{
		"USDC": {Volume: snap.GetFloat("usdc_volume"), Count: snap.GetInt("usdc_count")},
		"EURC": {Volume: snap.GetFloat("eurc_volume"), Count: snap.GetInt("eurc_count")},
		"USYC": {Volume: snap.GetFloat("usyc_volume"), Count: snap.GetInt("usyc_count")},
	}
	if token != "" {
		filtered := map[string]*tokenStats{}
		if ts, ok := byToken[token]; ok {
			filtered[token] = ts
		}
		byToken = filtered
	}

	return c.JSON(http.StatusOK, map[string]any{
		"window":           window,
		"token":            token,
		"total_transfers":  snap.GetInt("total_transfers"),
		"unique_senders":   snap.GetInt("unique_senders"),
		"unique_receivers": snap.GetInt("unique_receivers"),
		"whale_transfers":  snap.GetInt("whale_transfers"),
		"by_token":         byToken,
		"snapshot_at":      snap.GetInt("snapshot_at"),
	})
}

// API_DESC Cross-chain net flow: inbound vs outbound USDC grouped by chain
// API_TAGS CrossChain
func analyticsBridgeFlowHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 30)
	window := qp(c, "window", "24h")
	snap, ok := latestSnapshot(c.App, window)
	if !ok {
		return c.JSON(http.StatusOK, map[string]any{"window": window, "syncing": true})
	}

	var byChain any
	if s := snap.GetString("bridge_by_chain"); s != "" {
		_ = json.Unmarshal([]byte(s), &byChain)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"window":         window,
		"inbound_vol":    snap.GetFloat("bridge_inbound_vol"),
		"inbound_count":  snap.GetInt("bridge_inbound_count"),
		"outbound_vol":   snap.GetFloat("bridge_outbound_vol"),
		"outbound_count": snap.GetInt("bridge_outbound_count"),
		"net_flow":       snap.GetFloat("bridge_net_flow"),
		"by_chain":       byChain,
		"snapshot_at":    snap.GetInt("snapshot_at"),
	})
}

// API_DESC Single-request 24h dashboard summary: transfer count, volume, fees, bridge, agents
// API_TAGS Stats
func analyticsOverviewHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 30)
	window := qp(c, "window", "24h")
	snap, ok := latestSnapshot(c.App, window)
	if !ok {
		return c.JSON(http.StatusOK, map[string]any{"window": window, "syncing": true})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"window":                 window,
		"snapshot_at":            snap.GetInt("snapshot_at"),
		"transfers_count":        snap.GetInt("transfers_count"),
		"transfer_volume":        snap.GetFloat("transfer_volume"),
		"largest_transfer":       snap.GetFloat("largest_transfer"),
		"largest_transfer_block": snap.GetInt("largest_transfer_block"),
		"fees_total":             snap.GetFloat("fees_total"),
		"fee_p50":                snap.GetFloat("fee_p50"),
		"fee_p95":                snap.GetFloat("fee_p95"),
		"failed_tx_ratio":        snap.GetFloat("failed_tx_ratio"),
		"bridge_inbound_vol":     snap.GetFloat("bridge_inbound_vol"),
		"bridge_inbound_count":   snap.GetInt("bridge_inbound_count"),
		"bridge_outbound_vol":    snap.GetFloat("bridge_outbound_vol"),
		"bridge_outbound_count":  snap.GetInt("bridge_outbound_count"),
		"bridge_net_flow":        snap.GetFloat("bridge_net_flow"),
		"agent_count":            snap.GetInt("agent_count"),
	})
}

// API_DESC Historical analytics snapshots for time-series charting
// API_TAGS Stats
func analyticsHistoryHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 60)
	window := qp(c, "window", "24h")
	limit, _ := limitOffset(c)
	if limit > 1000 {
		limit = 1000
	}
	rows, _ := c.App.FindRecordsByFilter("analytics_snapshots",
		"window = {:w}", "-snapshot_at", limit, 0, map[string]any{"w": window})

	// reverse to ascending order (oldest first) for chart rendering
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		m := r.PublicExport()
		if s, ok := m["bridge_by_chain"].(string); ok && s != "" {
			var parsed any
			if err := json.Unmarshal([]byte(s), &parsed); err == nil {
				m["bridge_by_chain"] = parsed
			}
		}
		out[i] = m
	}

	return c.JSON(http.StatusOK, map[string]any{
		"window":    window,
		"snapshots": out,
		"count":     len(out),
	})
}
