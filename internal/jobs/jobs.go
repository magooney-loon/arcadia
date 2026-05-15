package jobs

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

func RegisterJobs(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := indexerHealthJob(app); err != nil {
			app.Logger().Error("Failed to register indexer health job", "error", err)
			return err
		}
		if err := indexerEventsCleanupJob(app); err != nil {
			app.Logger().Error("Failed to register indexer events cleanup job", "error", err)
			return err
		}
		if err := analyticsSnapshotJob(app); err != nil {
			app.Logger().Error("Failed to register analytics snapshot job", "error", err)
			return err
		}
		app.Logger().Info("All jobs registered")

		go func() {
			time.Sleep(5 * time.Second)
			for _, win := range []string{"1h", "24h", "7d"} {
				if err := takeAnalyticsSnapshot(app, win); err != nil {
					app.Logger().Warn("Initial analytics snapshot failed", "window", win, "error", err)
				}
			}
		}()

		return e.Next()
	})
}

func indexerEventsCleanupJob(app core.App) error {
	jm := jobs.GetManager()
	if jm == nil {
		return fmt.Errorf("job manager not initialized")
	}

	return jm.RegisterJob(
		"indexerEventsCleanup",
		"Indexer Events Cleanup",
		"Deletes indexer_events records older than 2 hours",
		"0 * * * *",
		func(el *jobs.ExecutionLogger) {
			el.Start("Indexer Events Cleanup")

			cutoff := time.Now().Add(-2 * time.Hour).Unix()
			records, err := app.FindRecordsByFilter(
				"indexer_events",
				fmt.Sprintf("timestamp < %d", cutoff),
				"", 0, 0,
			)
			if err != nil {
				el.Info("Failed to fetch old events: %s", err)
				el.Complete("Cleanup failed")
				return
			}

			deleted := 0
			for _, r := range records {
				if err := app.Delete(r); err == nil {
					deleted++
				}
			}

			el.Statistics(map[string]interface{}{"deleted": deleted})
			el.Complete(fmt.Sprintf("Deleted %d events older than 2 hours", deleted))
		},
	)
}

func indexerHealthJob(app core.App) error {
	jm := jobs.GetManager()
	if jm == nil {
		return fmt.Errorf("job manager not initialized")
	}

	return jm.RegisterJob(
		"indexerHealth",
		"Indexer Health Check",
		"Logs the current indexer cursor and row counts for all arcadia collections",
		"0 * * * *",
		func(el *jobs.ExecutionLogger) {
			el.Start("Indexer Health Check")
			el.Info("Checked at: %s", time.Now().Format(time.RFC3339))

			collections := []string{
				"indexer_events", "blocks", "transactions", "transfers", "traces",
				"crosschain_events", "fx_swaps", "agents", "agent_jobs",
				"block_stats", "wallet_edges",
			}

			counts := make(map[string]interface{})
			for _, name := range collections {
				records, err := app.FindRecordsByFilter(name, "", "", 0, 0)
				if err != nil {
					counts[name] = "error"
				} else {
					counts[name] = len(records)
				}
			}

			lastBlock := "unknown"
			if last := utils.GetLastIndexedBlock(app); last > 0 {
				lastBlock = fmt.Sprintf("%d", last)
			}

			el.Info("Last indexed block: %s", lastBlock)
			el.Statistics(counts)
			el.Complete(fmt.Sprintf("Health check done — last block: %s", lastBlock))
		},
	)
}

func analyticsSnapshotJob(app core.App) error {
	jm := jobs.GetManager()
	if jm == nil {
		return fmt.Errorf("job manager not initialized")
	}

	return jm.RegisterJob(
		"analyticsSnapshot",
		"Analytics Snapshot",
		"Materializes 1h/24h/7d window aggregates into analytics_snapshots",
		"*/5 * * * *",
		func(el *jobs.ExecutionLogger) {
			el.Start("Analytics Snapshot")
			n := 0
			for _, win := range []string{"1h", "24h", "7d"} {
				if err := takeAnalyticsSnapshot(app, win); err != nil {
					el.Info("snapshot %s failed: %s", win, err)
				} else {
					n++
				}
			}
			el.Statistics(map[string]interface{}{"windows": n})
			el.Complete(fmt.Sprintf("took %d snapshots", n))
		},
	)
}

func takeAnalyticsSnapshot(app core.App, window string) error {
	_, wParams := utils.WindowBlockFilter(app, window)
	fromBlock := wParams["from"]

	latest, _ := app.FindRecordsByFilter("block_stats", "", "-block_number", 1, 0)
	if len(latest) == 0 {
		return fmt.Errorf("no block_stats yet")
	}
	blockNumber := latest[0].GetInt("block_number")

	// ── transfers / volume ────────────────────────────────────────────────────

	var tStats struct {
		Count  int     `db:"cnt"`
		Volume float64 `db:"vol"`
	}
	_ = app.DB().NewQuery(
		`SELECT COUNT(*) AS cnt, COALESCE(SUM(CAST(amount_human AS REAL)), 0) AS vol
		 FROM transfers WHERE block_number >= {:from} AND token_symbol != 'OTHER'`).
		Bind(dbx.Params{"from": fromBlock}).One(&tStats)

	var largest struct {
		Amount float64 `db:"amt"`
		Block  int     `db:"block_number"`
	}
	_ = app.DB().NewQuery(
		`SELECT CAST(amount_human AS REAL) AS amt, block_number
		 FROM transfers WHERE block_number >= {:from} AND token_symbol != 'OTHER'
		 ORDER BY CAST(amount_human AS REAL) DESC LIMIT 1`).
		Bind(dbx.Params{"from": fromBlock}).One(&largest)

	type groupRow struct {
		Symbol string  `db:"token_symbol"`
		Vol    float64 `db:"vol"`
		Cnt    int     `db:"cnt"`
		Whales int     `db:"whales"`
	}
	var groupRows []groupRow
	_ = app.DB().NewQuery(
		`SELECT token_symbol,
		        COALESCE(SUM(CAST(amount_human AS REAL)), 0) AS vol,
		        COUNT(*) AS cnt,
		        SUM(CASE WHEN CAST(amount_human AS REAL) >= 10000 THEN 1 ELSE 0 END) AS whales
		 FROM transfers WHERE block_number >= {:from} AND token_symbol != 'OTHER'
		 GROUP BY token_symbol`).Bind(dbx.Params{"from": fromBlock}).All(&groupRows)

	volByToken := map[string]float64{}
	cntByToken := map[string]int{}
	totalWhales := 0
	for _, g := range groupRows {
		volByToken[g.Symbol] = g.Vol
		cntByToken[g.Symbol] = g.Cnt
		totalWhales += g.Whales
	}

	var addrs struct {
		Senders   int `db:"senders"`
		Receivers int `db:"receivers"`
	}
	_ = app.DB().NewQuery(
		`SELECT COUNT(DISTINCT from_addr) AS senders, COUNT(DISTINCT to_addr) AS receivers
		 FROM transfers WHERE block_number >= {:from} AND token_symbol != 'OTHER'`).
		Bind(dbx.Params{"from": fromBlock}).One(&addrs)

	// ── fees / tx stats ───────────────────────────────────────────────────────

	var fAgg struct {
		BlockCount     int     `db:"block_count"`
		FeesTotal      float64 `db:"fees_total"`
		TotalTxs       int64   `db:"total_txs"`
		FailedTxs      int64   `db:"failed_txs"`
		AvgBlockTimeMs float64 `db:"avg_bms"`
	}
	_ = app.DB().NewQuery(
		`SELECT
			COUNT(*) AS block_count,
			COALESCE(SUM(CAST(total_fee_usdc AS REAL)), 0) AS fees_total,
			COALESCE(SUM(tx_count), 0) AS total_txs,
			COALESCE(SUM(failed_tx_count), 0) AS failed_txs,
			COALESCE(AVG(CASE WHEN block_time_ms > 0 THEN block_time_ms END), 0) AS avg_bms
		 FROM block_stats WHERE block_number >= {:from}`).
		Bind(dbx.Params{"from": fromBlock}).One(&fAgg)

	fees := utils.LoadFeeColumn(app, fromBlock)
	sort.Float64s(fees)

	var failedRatio float64
	if fAgg.TotalTxs > 0 {
		failedRatio = float64(fAgg.FailedTxs) / float64(fAgg.TotalTxs)
	}

	// ── bridge ────────────────────────────────────────────────────────────────

	type chainFlow struct {
		InboundVol    float64 `json:"inbound_vol"`
		InboundCount  int     `json:"inbound_count"`
		OutboundVol   float64 `json:"outbound_vol"`
		OutboundCount int     `json:"outbound_count"`
	}
	type bridgeRow struct {
		Dir         string  `db:"dir"`
		ChainDomain int     `db:"chain_domain"`
		Cnt         int     `db:"cnt"`
		Vol         float64 `db:"vol"`
	}
	var bridgeRows []bridgeRow
	_ = app.DB().NewQuery(
		`SELECT
			CASE WHEN destination_domain = 26 THEN 'in' ELSE 'out' END AS dir,
			CASE WHEN destination_domain = 26 THEN source_domain ELSE destination_domain END AS chain_domain,
			COUNT(*) AS cnt,
			COALESCE(SUM(CAST(amount_usdc AS REAL)), 0) AS vol
		 FROM crosschain_events
		 WHERE block_number >= {:from} AND (destination_domain = 26 OR source_domain = 26)
		 GROUP BY dir, chain_domain`).
		Bind(dbx.Params{"from": fromBlock}).All(&bridgeRows)

	byChain := map[string]*chainFlow{}
	var totalIn, totalOut float64
	var countIn, countOut int
	for _, r := range bridgeRows {
		k := utils.DomainName(r.ChainDomain)
		if byChain[k] == nil {
			byChain[k] = &chainFlow{}
		}
		if r.Dir == "in" {
			byChain[k].InboundVol += r.Vol
			byChain[k].InboundCount += r.Cnt
			totalIn += r.Vol
			countIn += r.Cnt
		} else {
			byChain[k].OutboundVol += r.Vol
			byChain[k].OutboundCount += r.Cnt
			totalOut += r.Vol
			countOut += r.Cnt
		}
	}
	bridgeByChainJSON, _ := json.Marshal(byChain)

	// ── agents ────────────────────────────────────────────────────────────────

	var agentCount struct {
		Count int `db:"cnt"`
	}
	_ = app.DB().NewQuery(`SELECT COUNT(*) AS cnt FROM agents`).One(&agentCount)

	// ── write snapshot ────────────────────────────────────────────────────────

	c := utils.MustCollection(app, "analytics_snapshots")
	row := core.NewRecord(c)
	row.Set("snapshot_at", time.Now().Unix())
	row.Set("block_number", blockNumber)
	row.Set("window", window)
	row.Set("transfers_count", tStats.Count)
	row.Set("transfer_volume", tStats.Volume)
	row.Set("total_transfers", tStats.Count)
	row.Set("largest_transfer", largest.Amount)
	row.Set("largest_transfer_block", largest.Block)
	row.Set("usdc_volume", volByToken["USDC"])
	row.Set("eurc_volume", volByToken["EURC"])
	row.Set("usyc_volume", volByToken["USYC"])
	row.Set("usdc_count", cntByToken["USDC"])
	row.Set("eurc_count", cntByToken["EURC"])
	row.Set("usyc_count", cntByToken["USYC"])
	row.Set("whale_transfers", totalWhales)
	row.Set("unique_senders", addrs.Senders)
	row.Set("unique_receivers", addrs.Receivers)
	row.Set("fees_total", fAgg.FeesTotal)
	row.Set("fee_p25", utils.PercentileFloat(fees, 25))
	row.Set("fee_p50", utils.PercentileFloat(fees, 50))
	row.Set("fee_p75", utils.PercentileFloat(fees, 75))
	row.Set("fee_p95", utils.PercentileFloat(fees, 95))
	row.Set("failed_tx_ratio", failedRatio)
	row.Set("total_txs", fAgg.TotalTxs)
	row.Set("failed_txs", fAgg.FailedTxs)
	row.Set("avg_block_time_ms", fAgg.AvgBlockTimeMs)
	row.Set("block_count", fAgg.BlockCount)
	row.Set("bridge_inbound_vol", totalIn)
	row.Set("bridge_inbound_count", countIn)
	row.Set("bridge_outbound_vol", totalOut)
	row.Set("bridge_outbound_count", countOut)
	row.Set("bridge_net_flow", totalIn-totalOut)
	row.Set("bridge_by_chain", string(bridgeByChainJSON))
	row.Set("agent_count", agentCount.Count)

	return app.Save(row)
}
