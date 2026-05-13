package main

// API_SOURCE

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
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

// ── handlers ──────────────────────────────────────────────────────────────────

// API_DESC Latest live chain stats (TPS, fees, transfer volumes, agent activity)
// API_TAGS Stats
func statsHandler(c *core.RequestEvent) error {
	// latest block_stats row
	rows, err := c.App.FindRecordsByFilter("block_stats", "", "-block_number", 1, 0)
	if err != nil || len(rows) == 0 {
		return c.JSON(http.StatusOK, map[string]any{"syncing": true})
	}
	latest := rows[0].PublicExport()

	// rolling 10-block avg for tps + block_time_ms (stored values may be 0 for old rows)
	recent, _ := c.App.FindRecordsByFilter("block_stats", "", "-block_number", 10, 0)
	if len(recent) >= 2 {
		var totalTxs, totalBms int64
		var count int
		for _, r := range recent {
			bms := r.GetInt("block_time_ms")
			if bms > 0 {
				totalTxs += int64(r.GetInt("tx_count"))
				totalBms += int64(bms)
				count++
			}
		}
		if count > 0 && totalBms > 0 {
			avgBms := totalBms / int64(count)
			latest["block_time_ms"] = avgBms
			latest["tps"] = float64(totalTxs) / float64(count) / (float64(avgBms) / 1000.0)
		}
	}

	// indexer cursor
	cursor, _ := c.App.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if len(cursor) > 0 {
		latest["indexed_block"] = cursor[0].GetString("value")
	}

	return c.JSON(http.StatusOK, latest)
}

// API_DESC Recent blocks with derived stats
// API_TAGS Chain
func blocksHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)
	records, err := c.App.FindRecordsByFilter("blocks", "", "-number", limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"blocks": recordsToMaps(records),
		"count":  len(records),
	})
}

// API_DESC Recent transactions
// API_TAGS Chain
func transactionsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	if block := qp(c, "block", ""); block != "" {
		filter = "block_number = {:b}"
		params["b"] = block
	}
	if from := qp(c, "from", ""); from != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "from_addr = {:f}"
		params["f"] = from
	}
	if to := qp(c, "to", ""); to != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "to_addr = {:t}"
		params["t"] = to
	}

	records, err := c.App.FindRecordsByFilter("transactions", filter, "-block_number", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"transactions": recordsToMaps(records),
		"count":        len(records),
	})
}

// API_DESC Token transfers — filterable by block, token symbol, sender, or receiver
// API_TAGS Transfers
func transfersHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	if block := qp(c, "block", ""); block != "" {
		filter = "block_number = {:b}"
		params["b"] = block
	}
	if token := qp(c, "token", ""); token != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "token_symbol = {:sym}"
		params["sym"] = token
	}
	if from := qp(c, "from", ""); from != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "from_addr = {:f}"
		params["f"] = from
	}
	if to := qp(c, "to", ""); to != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "to_addr = {:t}"
		params["t"] = to
	}

	records, err := c.App.FindRecordsByFilter("transfers", filter, "-block_number", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"transfers": recordsToMaps(records),
		"count":     len(records),
	})
}

// API_DESC Wallet profile: transaction history + edges + agent status
// API_TAGS Wallets
func walletHandler(c *core.RequestEvent) error {
	address := c.Request.PathValue("address")
	if address == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "address required"})
	}

	limit, offset := limitOffset(c)

	// outgoing + incoming transfers
	sent, _ := c.App.FindRecordsByFilter("transfers", "from_addr = {:a}", "-block_number", limit, offset, map[string]any{"a": address})
	received, _ := c.App.FindRecordsByFilter("transfers", "to_addr = {:a}", "-block_number", limit, offset, map[string]any{"a": address})

	// graph edges
	outEdges, _ := c.App.FindRecordsByFilter("wallet_edges", "from_wallet = {:a}", "-tx_count", 20, 0, map[string]any{"a": address})
	inEdges, _ := c.App.FindRecordsByFilter("wallet_edges", "to_wallet = {:a}", "-tx_count", 20, 0, map[string]any{"a": address})

	// transactions
	txsSent, _ := c.App.FindRecordsByFilter("transactions", "from_addr = {:a}", "-block_number", limit, offset, map[string]any{"a": address})
	txsReceived, _ := c.App.FindRecordsByFilter("transactions", "to_addr = {:a}", "-block_number", limit, offset, map[string]any{"a": address})

	// agent status
	agentRecords, _ := c.App.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": address})
	var agentData any
	if len(agentRecords) > 0 {
		agentData = agentRecords[0].PublicExport()
	}

	return c.JSON(http.StatusOK, map[string]any{
		"address":        address,
		"is_agent":       agentData != nil,
		"agent":          agentData,
		"txs_sent":       recordsToMaps(txsSent),
		"txs_received":   recordsToMaps(txsReceived),
		"sent":           recordsToMaps(sent),
		"received":       recordsToMaps(received),
		"outgoing_edges": recordsToMaps(outEdges),
		"incoming_edges": recordsToMaps(inEdges),
	})
}

// API_DESC Recent cross-chain events (CCTP + Gateway)
// API_TAGS CrossChain
func crosschainHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	addFilter := func(clause string) {
		if filter != "" {
			filter += " && "
		}
		filter += clause
	}

	if proto := qp(c, "protocol", ""); proto != "" {
		addFilter("protocol = {:p}")
		params["p"] = proto
	}
	if et := qp(c, "event_type", ""); et != "" {
		addFilter("event_type = {:et}")
		params["et"] = et
	}
	if sender := qp(c, "sender", ""); sender != "" {
		addFilter("sender = {:s}")
		params["s"] = sender
	}
	if recipient := qp(c, "recipient", ""); recipient != "" {
		addFilter("recipient = {:r}")
		params["r"] = recipient
	}
	// direction=inbound  → USDC arriving on Arc (destination_domain = 26)
	// direction=outbound → USDC leaving Arc   (source_domain = 26, destination != 26)
	switch qp(c, "direction", "") {
	case "inbound":
		addFilter("destination_domain = 26")
	case "outbound":
		addFilter("source_domain = 26 && destination_domain != 26")
	}

	records, err := c.App.FindRecordsByFilter("crosschain_events", filter, "-block_number", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"events": recordsToMaps(records),
		"count":  len(records),
	})
}

// API_DESC StableFX trades — filterable by status, maker, taker, or quote_id
// API_TAGS FX
func fxHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	if status := qp(c, "status", ""); status != "" {
		filter = "status = {:s}"
		params["s"] = status
	}
	if maker := qp(c, "maker", ""); maker != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "maker = {:m}"
		params["m"] = maker
	}
	if taker := qp(c, "taker", ""); taker != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "taker = {:t}"
		params["t"] = taker
	}
	if quoteID := qp(c, "quote_id", ""); quoteID != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "quote_id = {:q}"
		params["q"] = quoteID
	}

	records, err := c.App.FindRecordsByFilter("fx_swaps", filter, "-block_number", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"trades": recordsToMaps(records),
		"count":  len(records),
	})
}

// API_DESC List all registered AI agents (ERC-8004)
// API_TAGS Agents
func agentsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)
	records, err := c.App.FindRecordsByFilter("agents", "", "-registered_at_block", limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"agents": recordsToMaps(records),
		"count":  len(records),
	})
}

// API_DESC Single agent profile + job history
// API_TAGS Agents
func agentHandler(c *core.RequestEvent) error {
	address := c.Request.PathValue("address")
	if address == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "address required"})
	}

	agentRows, _ := c.App.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": address})
	if len(agentRows) == 0 {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "agent not found"})
	}

	jobs, _ := c.App.FindRecordsByFilter("agent_jobs",
		"employer_address = {:a} || worker_address = {:a}", "-created_at_block", 50, 0,
		map[string]any{"a": address})

	return c.JSON(http.StatusOK, map[string]any{
		"agent": agentRows[0].PublicExport(),
		"jobs":  recordsToMaps(jobs),
	})
}

// API_DESC Agent job marketplace — filter by status or worker/employer
// API_TAGS Agents
func agentJobsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}
	if status := qp(c, "status", ""); status != "" {
		filter = "status = {:s}"
		params["s"] = status
	}
	if employer := qp(c, "employer", ""); employer != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "employer_address = {:e}"
		params["e"] = employer
	}
	if worker := qp(c, "worker", ""); worker != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "worker_address = {:w}"
		params["w"] = worker
	}

	records, err := c.App.FindRecordsByFilter("agent_jobs", filter, "-created_at_block", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"jobs":  recordsToMaps(records),
		"count": len(records),
	})
}

// API_DESC Internal contract-to-contract calls — filterable by tx hash or address
// API_TAGS Chain
func tracesHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}
	if tx := qp(c, "tx", ""); tx != "" {
		filter = "tx_hash = {:tx}"
		params["tx"] = tx
	}
	if from := qp(c, "from", ""); from != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "from_addr = {:f}"
		params["f"] = from
	}
	if to := qp(c, "to", ""); to != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "to_addr = {:t}"
		params["t"] = to
	}

	records, err := c.App.FindRecordsByFilter("traces", filter, "-block_number", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"traces": recordsToMaps(records),
		"count":  len(records),
	})
}

// API_DESC Historical block stats for time-series charts (sorted newest first)
// API_TAGS Stats
func blockStatsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)
	records, err := c.App.FindRecordsByFilter("block_stats", "", "-block_number", limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"stats": recordsToMaps(records),
		"count": len(records),
	})
}

// API_DESC Wallet graph edges for 3D visualization — filterable by wallet address
// API_TAGS Graph
func edgesHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}
	if wallet := qp(c, "wallet", ""); wallet != "" {
		filter = "from_wallet = {:w} || to_wallet = {:w}"
		params["w"] = wallet
	}

	records, err := c.App.FindRecordsByFilter("wallet_edges", filter, "-tx_count", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"edges": recordsToMaps(records),
		"count": len(records),
	})
}

// ── analytics helpers ─────────────────────────────────────────────────────────

func parseUSDC(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// windowToBlockCount converts a time window string to an approximate block count.
// Arc L1 averages ~380 ms per block.
func windowToBlockCount(window string) int {
	const avgBlockMs = 380
	switch window {
	case "1h":
		return 3_600_000 / avgBlockMs
	case "7d":
		return 7 * 86_400_000 / avgBlockMs
	default: // "24h"
		return 86_400_000 / avgBlockMs
	}
}

// windowBlockFilter returns a PocketBase filter + params scoped to the last N blocks.
// It fetches the current tip from block_stats so callers don't need to.
func windowBlockFilter(app core.App, window string) (string, map[string]any) {
	blockCount := windowToBlockCount(window)
	latest, _ := app.FindRecordsByFilter("block_stats", "", "-block_number", 1, 0)
	fromBlock := 0
	if len(latest) > 0 {
		fromBlock = latest[0].GetInt("block_number") - blockCount
		if fromBlock < 0 {
			fromBlock = 0
		}
	}
	return "block_number >= {:from}", map[string]any{"from": fromBlock}
}

func percentileFloat(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := int(p / 100.0 * float64(n))
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
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

var domainNames = map[int]string{
	0:  "Ethereum",
	1:  "Avalanche",
	2:  "OP Mainnet",
	3:  "Arbitrum",
	5:  "Solana",
	6:  "Base",
	7:  "Polygon PoS",
	10: "Unichain",
	11: "Linea",
	12: "Codex",
	13: "Sonic",
	14: "World Chain",
	15: "Monad",
	16: "Sei",
	17: "BNB Smart Chain",
	18: "XDC",
	19: "HyperEVM",
	21: "Ink",
	22: "Plume",
	25: "Starknet",
	26: "Arc Testnet",
	27: "Stellar",
	28: "EDGE",
	29: "Injective",
	30: "Morph",
	31: "Pharos",
}

func domainName(id int) string {
	if n, ok := domainNames[id]; ok {
		return n
	}
	return strconv.Itoa(id)
}

// ── new endpoints ─────────────────────────────────────────────────────────────

// API_DESC Indexer health: lag, error rate, last indexed block
// API_TAGS Stats
func healthHandler(c *core.RequestEvent) error {
	cursor, _ := c.App.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	var lastBlock int
	if len(cursor) > 0 {
		lastBlock, _ = strconv.Atoi(cursor[0].GetString("value"))
	}

	// latest heartbeat carries tip + lag written by the indexer loop
	heartbeats, _ := c.App.FindRecordsByFilter("indexer_events", "event = 'heartbeat'", "-created", 1, 0)
	var tip, lag int
	if len(heartbeats) > 0 {
		tip = heartbeats[0].GetInt("tip")
		lag = heartbeats[0].GetInt("lag")
	}

	since := time.Now().UTC().Add(-time.Hour).Format("2006-01-02 15:04:05.000Z")
	errEvents, _ := c.App.FindRecordsByFilter("indexer_events",
		"level = 'error' && created >= {:since}", "", 500, 0,
		map[string]any{"since": since})

	batches, _ := c.App.FindRecordsByFilter("indexer_events",
		"event = 'batch_done' && created >= {:since}", "-created", 20, 0,
		map[string]any{"since": since})
	var avgBatchMs float64
	if len(batches) > 0 {
		var total int64
		for _, r := range batches {
			total += int64(r.GetInt("duration_ms"))
		}
		avgBatchMs = float64(total) / float64(len(batches))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"last_indexed_block": lastBlock,
		"chain_tip":          tip,
		"lag_blocks":         lag,
		"syncing":            lag > 10,
		"errors_1h":          len(errEvents),
		"avg_batch_ms":       avgBatchMs,
	})
}

// API_DESC Unified search by tx hash (0x+64), address (0x+40), or block number
// API_TAGS Chain
func searchHandler(c *core.RequestEvent) error {
	q := strings.TrimSpace(qp(c, "q", ""))
	if q == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "q required"})
	}

	if len(q) == 66 && strings.HasPrefix(q, "0x") {
		rows, _ := c.App.FindRecordsByFilter("transactions", "hash = {:h}", "", 1, 0, map[string]any{"h": q})
		if len(rows) > 0 {
			return c.JSON(http.StatusOK, map[string]any{"type": "tx", "result": rows[0].PublicExport()})
		}
		return c.JSON(http.StatusOK, map[string]any{"type": "not_found"})
	}

	if len(q) == 42 && strings.HasPrefix(q, "0x") {
		agents, _ := c.App.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": q})
		if len(agents) > 0 {
			return c.JSON(http.StatusOK, map[string]any{
				"type":   "agent",
				"result": map[string]any{"address": q, "is_agent": true, "agent": agents[0].PublicExport()},
			})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"type":   "wallet",
			"result": map[string]any{"address": q, "is_agent": false},
		})
	}

	if isNumeric(q) {
		num, _ := strconv.Atoi(q)
		rows, _ := c.App.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": num})
		if len(rows) > 0 {
			return c.JSON(http.StatusOK, map[string]any{"type": "block", "result": rows[0].PublicExport()})
		}
		return c.JSON(http.StatusOK, map[string]any{"type": "not_found"})
	}

	return c.JSON(http.StatusOK, map[string]any{"type": "unknown"})
}

// API_DESC Single transaction detail with related transfers and traces
// API_TAGS Chain
func txDetailHandler(c *core.RequestEvent) error {
	hash := c.Request.PathValue("hash")
	if hash == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "hash required"})
	}

	txs, _ := c.App.FindRecordsByFilter("transactions", "hash = {:h}", "", 1, 0, map[string]any{"h": hash})
	if len(txs) == 0 {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "transaction not found"})
	}

	transfers, _ := c.App.FindRecordsByFilter("transfers", "tx_hash = {:h}", "-block_number", 50, 0, map[string]any{"h": hash})
	traces, _ := c.App.FindRecordsByFilter("traces", "tx_hash = {:h}", "", 50, 0, map[string]any{"h": hash})

	return c.JSON(http.StatusOK, map[string]any{
		"transaction": txs[0].PublicExport(),
		"transfers":   recordsToMaps(transfers),
		"traces":      recordsToMaps(traces),
	})
}

// API_DESC Single block detail with its transactions and pre-aggregated stats
// API_TAGS Chain
func blockDetailHandler(c *core.RequestEvent) error {
	numStr := c.Request.PathValue("number")
	num, err := strconv.Atoi(numStr)
	if err != nil || numStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "block number required"})
	}

	blocks, _ := c.App.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": num})
	if len(blocks) == 0 {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "block not found"})
	}

	txs, _ := c.App.FindRecordsByFilter("transactions", "block_number = {:n}", "", 500, 0, map[string]any{"n": num})
	stats, _ := c.App.FindRecordsByFilter("block_stats", "block_number = {:n}", "", 1, 0, map[string]any{"n": num})

	result := map[string]any{
		"block":        blocks[0].PublicExport(),
		"transactions": recordsToMaps(txs),
	}
	if len(stats) > 0 {
		result["stats"] = stats[0].PublicExport()
	}
	return c.JSON(http.StatusOK, result)
}

// API_DESC Fee analytics: P25/P50/P75/P95 percentiles, failed tx ratio, avg block time
// API_TAGS Stats
func analyticsFeesHandler(c *core.RequestEvent) error {
	window := qp(c, "window", "24h")
	filterStr, params := windowBlockFilter(c.App, window)

	records, err := c.App.FindRecordsByFilter("block_stats", filterStr, "-block_number", 500, 0, params)
	if err != nil || len(records) == 0 {
		return c.JSON(http.StatusOK, map[string]any{"window": window, "block_count": 0})
	}

	fees := make([]float64, 0, len(records))
	var totalFees, totalTxs, failedTxs, totalBms float64
	var bmsCount int

	for _, r := range records {
		fees = append(fees, parseUSDC(r.GetString("avg_fee_usdc")))
		totalFees += parseUSDC(r.GetString("total_fee_usdc"))
		totalTxs += float64(r.GetInt("tx_count"))
		failedTxs += float64(r.GetInt("failed_tx_count"))
		if bms := float64(r.GetInt("block_time_ms")); bms > 0 {
			totalBms += bms
			bmsCount++
		}
	}

	sort.Float64s(fees)

	var avgBms, failedRatio float64
	if bmsCount > 0 {
		avgBms = totalBms / float64(bmsCount)
	}
	if totalTxs > 0 {
		failedRatio = failedTxs / totalTxs
	}

	return c.JSON(http.StatusOK, map[string]any{
		"window":            window,
		"block_count":       len(records),
		"total_fees":        totalFees,
		"avg_fee_p25":       percentileFloat(fees, 25),
		"avg_fee_p50":       percentileFloat(fees, 50),
		"avg_fee_p75":       percentileFloat(fees, 75),
		"avg_fee_p95":       percentileFloat(fees, 95),
		"avg_block_time_ms": avgBms,
		"failed_tx_ratio":   failedRatio,
	})
}

// API_DESC Transfer volume aggregates with whale count and per-token breakdown
// API_TAGS Transfers
func analyticsVolumeHandler(c *core.RequestEvent) error {
	window := qp(c, "window", "24h")
	token := qp(c, "token", "")
	filterStr, params := windowBlockFilter(c.App, window)

	if token != "" {
		filterStr += " && token_symbol = {:sym}"
		params["sym"] = token
	}

	records, err := c.App.FindRecordsByFilter("transfers", filterStr, "-block_number", 500, 0, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	type tokenStats struct {
		Volume     float64 `json:"volume"`
		Count      int     `json:"count"`
		WhaleCount int     `json:"whale_count"`
	}
	byToken := map[string]*tokenStats{}
	senders := map[string]bool{}
	receivers := map[string]bool{}
	var whaleCount int
	const whaleThreshold = 10_000.0

	for _, r := range records {
		sym := r.GetString("token_symbol")
		amt := parseUSDC(r.GetString("amount_human"))

		if byToken[sym] == nil {
			byToken[sym] = &tokenStats{}
		}
		byToken[sym].Volume += amt
		byToken[sym].Count++
		senders[r.GetString("from_addr")] = true
		receivers[r.GetString("to_addr")] = true

		if amt >= whaleThreshold {
			whaleCount++
			byToken[sym].WhaleCount++
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"window":           window,
		"token":            token,
		"total_transfers":  len(records),
		"unique_senders":   len(senders),
		"unique_receivers": len(receivers),
		"whale_transfers":  whaleCount,
		"by_token":         byToken,
	})
}

// API_DESC Cross-chain net flow: inbound vs outbound USDC grouped by chain
// API_TAGS CrossChain
func analyticsBridgeFlowHandler(c *core.RequestEvent) error {
	window := qp(c, "window", "24h")
	filterStr, params := windowBlockFilter(c.App, window)

	records, err := c.App.FindRecordsByFilter("crosschain_events", filterStr, "-block_number", 500, 0, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	type chainFlow struct {
		InboundVol    float64 `json:"inbound_vol"`
		InboundCount  int     `json:"inbound_count"`
		OutboundVol   float64 `json:"outbound_vol"`
		OutboundCount int     `json:"outbound_count"`
	}
	byChain := map[string]*chainFlow{}
	var totalIn, totalOut float64
	var countIn, countOut int

	for _, r := range records {
		amt := parseUSDC(r.GetString("amount_usdc"))
		src := r.GetInt("source_domain")
		dst := r.GetInt("destination_domain")

		if dst == 26 {
			totalIn += amt
			countIn++
			k := domainName(src)
			if byChain[k] == nil {
				byChain[k] = &chainFlow{}
			}
			byChain[k].InboundVol += amt
			byChain[k].InboundCount++
		} else if src == 26 {
			totalOut += amt
			countOut++
			k := domainName(dst)
			if byChain[k] == nil {
				byChain[k] = &chainFlow{}
			}
			byChain[k].OutboundVol += amt
			byChain[k].OutboundCount++
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"window":         window,
		"inbound_vol":    totalIn,
		"inbound_count":  countIn,
		"outbound_vol":   totalOut,
		"outbound_count": countOut,
		"net_flow":       totalIn - totalOut,
		"by_chain":       byChain,
	})
}

// API_DESC Agent leaderboard ranked by tx_count with job stats attached
// API_TAGS Agents
func analyticsAgentLeaderboardHandler(c *core.RequestEvent) error {
	limit, _ := limitOffset(c)
	if limit > 100 {
		limit = 100
	}

	agents, err := c.App.FindRecordsByFilter("agents", "", "-tx_count", limit, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	result := make([]map[string]any, 0, len(agents))
	for _, a := range agents {
		entry := a.PublicExport()
		addr := a.GetString("agent_address")

		jobs, _ := c.App.FindRecordsByFilter("agent_jobs",
			"employer_address = {:a} || worker_address = {:a}", "", 500, 0,
			map[string]any{"a": addr})

		var totalEscrow float64
		var settled, disputed int
		for _, j := range jobs {
			totalEscrow += parseUSDC(j.GetString("payment_usdc"))
			switch j.GetString("status") {
			case "settled":
				settled++
			case "disputed":
				disputed++
			}
		}

		entry["job_count"] = len(jobs)
		entry["total_escrow"] = totalEscrow
		entry["settled_jobs"] = settled
		entry["disputed_jobs"] = disputed
		result = append(result, entry)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"leaderboard": result,
		"count":       len(result),
	})
}
