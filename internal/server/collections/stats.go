package collections

import (
	"github.com/pocketbase/pocketbase/core"
)

// blockStatsCollection — Layer 7: pre-aggregated per-block metrics.
func blockStatsCollection(app core.App) error {
	if collectionExists(app, "block_stats") {
		c, err := app.FindCollectionByNameOrId("block_stats")
		if err != nil {
			return err
		}
		changed := false
		// Indexed numeric mirrors of the text fee/amount columns. The text
		// columns keep precision; the numeric ones make analytics ORDER BY
		// / SUM index-backed instead of CAST-on-every-row.
		needsBackfill := false
		addNum := func(name string) {
			if c.Fields.GetByName(name) == nil {
				c.Fields.Add(&core.NumberField{Name: name})
				changed = true
				needsBackfill = true
			}
		}
		addNum("total_fee_num")
		addNum("avg_fee_num")
		addNum("largest_usdc_num")
		if changed {
			if err := app.Save(c); err != nil {
				return err
			}
		}
		if needsBackfill {
			if _, err := app.DB().NewQuery(
				`UPDATE block_stats
				 SET total_fee_num    = COALESCE(CAST(total_fee_usdc       AS REAL), 0),
				     avg_fee_num      = COALESCE(CAST(avg_fee_usdc         AS REAL), 0),
				     largest_usdc_num = COALESCE(CAST(largest_usdc_transfer AS REAL), 0)
				 WHERE total_fee_num IS NULL OR avg_fee_num IS NULL OR largest_usdc_num IS NULL`).Execute(); err != nil {
				app.Logger().Warn("block_stats numeric backfill failed", "error", err)
			}
		}
		return nil
	}
	c := core.NewBaseCollection("block_stats")
	c.Fields.Add(&core.NumberField{Name: "block_number", Required: true})
	c.Fields.Add(&core.NumberField{Name: "timestamp"})
	c.Fields.Add(&core.NumberField{Name: "tps"})
	c.Fields.Add(&core.NumberField{Name: "tx_count"})
	c.Fields.Add(&core.NumberField{Name: "failed_tx_count"})
	c.Fields.Add(&core.TextField{Name: "avg_fee_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "total_fee_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.NumberField{Name: "avg_fee_num"})
	c.Fields.Add(&core.NumberField{Name: "total_fee_num"})
	c.Fields.Add(&core.TextField{Name: "total_usdc_transferred", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "total_eurc_transferred", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "total_usyc_transferred", Required: false, Max: 40})
	c.Fields.Add(&core.NumberField{Name: "unique_senders"})
	c.Fields.Add(&core.NumberField{Name: "unique_receivers"})
	c.Fields.Add(&core.NumberField{Name: "new_contracts"})
	c.Fields.Add(&core.TextField{Name: "largest_usdc_transfer", Required: false, Max: 40})
	c.Fields.Add(&core.NumberField{Name: "largest_usdc_num"})
	c.Fields.Add(&core.NumberField{Name: "utilization_pct"})
	c.Fields.Add(&core.NumberField{Name: "block_time_ms"})
	c.AddIndex("idx_bstats_number", true, "block_number", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created block_stats collection")
	return nil
}

// analyticsSnapshotsCollection — Layer 9: pre-aggregated window analytics.
// One row per window per snapshot time. Handlers read the latest row; history
// endpoint returns all rows for time-series charting.
func analyticsSnapshotsCollection(app core.App) error {
	if collectionExists(app, "analytics_snapshots") {
		return nil
	}
	c := core.NewBaseCollection("analytics_snapshots")
	c.Fields.Add(&core.NumberField{Name: "snapshot_at", Required: true})
	c.Fields.Add(&core.NumberField{Name: "block_number"})
	c.Fields.Add(&core.TextField{Name: "window", Required: true, Max: 10})
	// transfers / volume
	c.Fields.Add(&core.NumberField{Name: "transfers_count"})
	c.Fields.Add(&core.NumberField{Name: "transfer_volume"})
	c.Fields.Add(&core.NumberField{Name: "largest_transfer"})
	c.Fields.Add(&core.NumberField{Name: "largest_transfer_block"})
	c.Fields.Add(&core.NumberField{Name: "usdc_volume"})
	c.Fields.Add(&core.NumberField{Name: "eurc_volume"})
	c.Fields.Add(&core.NumberField{Name: "usyc_volume"})
	c.Fields.Add(&core.NumberField{Name: "usdc_count"})
	c.Fields.Add(&core.NumberField{Name: "eurc_count"})
	c.Fields.Add(&core.NumberField{Name: "usyc_count"})
	c.Fields.Add(&core.NumberField{Name: "whale_transfers"})
	c.Fields.Add(&core.NumberField{Name: "unique_senders"})
	c.Fields.Add(&core.NumberField{Name: "unique_receivers"})
	c.Fields.Add(&core.NumberField{Name: "total_transfers"})
	// fees / tx stats
	c.Fields.Add(&core.NumberField{Name: "fees_total"})
	c.Fields.Add(&core.NumberField{Name: "fee_p25"})
	c.Fields.Add(&core.NumberField{Name: "fee_p50"})
	c.Fields.Add(&core.NumberField{Name: "fee_p75"})
	c.Fields.Add(&core.NumberField{Name: "fee_p95"})
	c.Fields.Add(&core.NumberField{Name: "failed_tx_ratio"})
	c.Fields.Add(&core.NumberField{Name: "total_txs"})
	c.Fields.Add(&core.NumberField{Name: "failed_txs"})
	c.Fields.Add(&core.NumberField{Name: "avg_block_time_ms"})
	c.Fields.Add(&core.NumberField{Name: "block_count"})
	// bridge
	c.Fields.Add(&core.NumberField{Name: "bridge_inbound_vol"})
	c.Fields.Add(&core.NumberField{Name: "bridge_inbound_count"})
	c.Fields.Add(&core.NumberField{Name: "bridge_outbound_vol"})
	c.Fields.Add(&core.NumberField{Name: "bridge_outbound_count"})
	c.Fields.Add(&core.NumberField{Name: "bridge_net_flow"})
	c.Fields.Add(&core.TextField{Name: "bridge_by_chain", Required: false, Max: 8000})
	// agents
	c.Fields.Add(&core.NumberField{Name: "agent_count"})
	c.AddIndex("idx_snapshots_window_at", false, "window, snapshot_at", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created analytics_snapshots collection")
	return nil
}
