package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
	"arcadia/internal/server/realtime"
	"arcadia/internal/utils"
)

// Cadence per window. The job manager fires every 5 minutes; we skip windows
// that don't meaningfully change at that tick.
//
//   1h:   every 5 minutes  (always)
//   24h:  every 10 minutes (minute % 10 == 0)
//   7d:   every 30 minutes (minute % 30 == 0)
//
// The 7d window scans the most data, so dropping its cadence by 6x is the
// single biggest reduction in steady-state SQLite pressure from this job.
func snapshotWindowsForTick(now time.Time) []string {
	m := now.Minute()
	out := []string{"1h"}
	if m%10 == 0 {
		out = append(out, "24h")
	}
	if m%30 == 0 {
		out = append(out, "7d")
	}
	return out
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
			windows := snapshotWindowsForTick(time.Now())
			n := 0
			for _, win := range windows {
				if err := takeAnalyticsSnapshot(app, win); err != nil {
					el.Info("snapshot %s failed: %s", win, err)
				} else {
					n++
					go realtime.BroadcastAnalyticsUpdate(app, win)
				}
			}
			el.Statistics(map[string]interface{}{"windows": n, "considered": len(windows)})
			el.Complete(fmt.Sprintf("took %d snapshots", n))
		},
	)
}

func takeAnalyticsSnapshot(app core.App, window string) error {
	// The 7d window is the only one that touches enough rows to risk
	// blocking the indexer write path; serialize it against token_analytics.
	// 1h and 24h are cheap enough to run anytime.
	if window == "7d" {
		if !heavyJobMu.TryLock() {
			log.Println("[analytics-snapshot] 7d skipped — heavy job in flight")
			return nil
		}
		defer heavyJobMu.Unlock()
	}

	_, wParams := utils.WindowBlockFilter(app, window)
	fromBlock := wParams["from"]

	latest, err := repo.LatestBlockStats(app)
	if err != nil {
		return fmt.Errorf("block_stats query: %w", err)
	}
	if latest == nil {
		return fmt.Errorf("no block_stats yet")
	}
	blockNumber := latest.GetInt("block_number")

	// ── transfers / volume ────────────────────────────────────────────────────
	//
	// Single scan: per-symbol counts/volume/whales + per-symbol unique sender
	// and receiver counts. Globally-distinct counts are approximated by
	// summing per-symbol distincts (an address that transfers in two
	// stablecoins is counted twice — acceptable for a dashboard metric, and
	// the alternative is a second full-window scan with COUNT(DISTINCT)).

	type groupRow struct {
		Symbol    string  `db:"token_symbol"`
		Vol       float64 `db:"vol"`
		Cnt       int     `db:"cnt"`
		Whales    int     `db:"whales"`
		Senders   int     `db:"senders"`
		Receivers int     `db:"receivers"`
	}
	var groupRows []groupRow
	_ = app.DB().NewQuery(
		`SELECT token_symbol,
		        COALESCE(SUM(amount_num), 0)                              AS vol,
		        COUNT(*)                                                  AS cnt,
		        SUM(CASE WHEN amount_num >= 10000 THEN 1 ELSE 0 END)      AS whales,
		        COUNT(DISTINCT from_addr)                                 AS senders,
		        COUNT(DISTINCT to_addr)                                   AS receivers
		 FROM transfers
		 WHERE block_number >= {:from} AND token_symbol != 'OTHER'
		 GROUP BY token_symbol`).
		Bind(dbx.Params{"from": fromBlock}).All(&groupRows)

	volByToken := map[string]float64{}
	cntByToken := map[string]int{}
	totalWhales, totalSenders, totalReceivers := 0, 0, 0
	tStats := struct {
		Count  int
		Volume float64
	}{}
	for _, g := range groupRows {
		volByToken[g.Symbol] = g.Vol
		cntByToken[g.Symbol] = g.Cnt
		totalWhales += g.Whales
		totalSenders += g.Senders
		totalReceivers += g.Receivers
		tStats.Count += g.Cnt
		tStats.Volume += g.Vol
	}

	// Largest transfer: index-backed (idx_transfers_amount_num), cheap.
	var largest struct {
		Amount float64 `db:"amt"`
		Block  int     `db:"block_number"`
	}
	_ = app.DB().NewQuery(
		`SELECT amount_num AS amt, block_number
		 FROM transfers WHERE block_number >= {:from} AND token_symbol != 'OTHER'
		 ORDER BY amount_num DESC LIMIT 1`).
		Bind(dbx.Params{"from": fromBlock}).One(&largest)

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
			COALESCE(SUM(total_fee_num), 0) AS fees_total,
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
			COALESCE(SUM(amount_usdc_num), 0) AS vol
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

	c, err := utils.FindCollection(app, "analytics_snapshots")
	if err != nil {
		return err
	}
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
	row.Set("unique_senders", totalSenders)
	row.Set("unique_receivers", totalReceivers)
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
