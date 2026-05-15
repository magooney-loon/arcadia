package server

// API_SOURCE

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

// ── handlers ──────────────────────────────────────────────────────────────────

// API_DESC Latest live chain stats (TPS, fees, transfer volumes, agent activity)
// API_TAGS Stats
func statsHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 2)
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
		params["sym"] = strings.ToUpper(token)
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

	// Fan out the 7 independent SELECTs concurrently. SQLite WAL allows many
	// readers in parallel; on the previous sequential path tail latency was
	// the sum of all 7 round-trips.
	var (
		wg                                                          sync.WaitGroup
		sent, received                                              []*core.Record
		outEdges, inEdges                                           []*core.Record
		txsSent, txsReceived                                        []*core.Record
		agentRecords                                                []*core.Record
	)
	params := map[string]any{"a": address}
	run := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}
	run(func() {
		sent, _ = c.App.FindRecordsByFilter("transfers", "from_addr = {:a}", "-block_number", limit, offset, params)
	})
	run(func() {
		received, _ = c.App.FindRecordsByFilter("transfers", "to_addr = {:a}", "-block_number", limit, offset, params)
	})
	run(func() {
		outEdges, _ = c.App.FindRecordsByFilter("wallet_edges", "from_wallet = {:a}", "-tx_count", 20, 0, params)
	})
	run(func() {
		inEdges, _ = c.App.FindRecordsByFilter("wallet_edges", "to_wallet = {:a}", "-tx_count", 20, 0, params)
	})
	run(func() {
		txsSent, _ = c.App.FindRecordsByFilter("transactions", "from_addr = {:a}", "-block_number", limit, offset, params)
	})
	run(func() {
		txsReceived, _ = c.App.FindRecordsByFilter("transactions", "to_addr = {:a}", "-block_number", limit, offset, params)
	})
	run(func() {
		agentRecords, _ = c.App.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, params)
	})
	wg.Wait()

	var agentData any
	if len(agentRecords) > 0 {
		agentData = enrichAgentRecord(agentRecords[0])
	}

	enrichEdges := func(recs []*core.Record) []map[string]any {
		out := make([]map[string]any, len(recs))
		for i, r := range recs {
			out[i] = enrichEdgeRecord(r)
		}
		return out
	}

	return c.JSON(http.StatusOK, map[string]any{
		"address":        address,
		"is_agent":       agentData != nil,
		"agent":          agentData,
		"txs_sent":       recordsToMaps(txsSent),
		"txs_received":   recordsToMaps(txsReceived),
		"sent":           recordsToMaps(sent),
		"received":       recordsToMaps(received),
		"outgoing_edges": enrichEdges(outEdges),
		"incoming_edges": enrichEdges(inEdges),
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
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = enrichAgentRecord(r)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"agents": out,
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
		"agent": enrichAgentRecord(agentRows[0]),
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
	if tx := qp(c, "tx_hash", ""); tx != "" {
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
	cacheHeaders(c, 2)
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
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = enrichEdgeRecord(r)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"edges": out,
		"count": len(records),
	})
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

// ── new endpoints ─────────────────────────────────────────────────────────────

// API_DESC Indexer health: lag, error rate, last indexed block
// API_TAGS Stats
func healthHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 2)
	// Single query for all meta keys instead of 3 separate lookups.
	metaRows, _ := c.App.FindRecordsByFilter("indexer_meta", "key != ''", "", 10, 0)
	metaMap := make(map[string]string, len(metaRows))
	for _, r := range metaRows {
		metaMap[r.GetString("key")] = r.GetString("value")
	}
	lastBlock, _ := strconv.Atoi(metaMap["lastBlock"])
	tip, _ := strconv.Atoi(metaMap["chainTip"])
	lag, _ := strconv.Atoi(metaMap["lagBlocks"])

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

// API_DESC All discovered ERC-20 tokens with aggregated analytics
// API_TAGS Tokens
func tokensHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	if search := qp(c, "search", ""); search != "" {
		filter = "(LOWER(symbol) LIKE {:s} OR LOWER(name) LIKE {:s} OR LOWER(token_address) LIKE {:s})"
		params["s"] = "%" + strings.ToLower(search) + "%"
	}

	records, err := c.App.FindRecordsByFilter("token_analytics", filter, "-transfer_count", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"tokens": recordsToMaps(records),
		"count":  len(records),
	})
}

// API_DESC Token detail with recent transfers
// API_TAGS Tokens
func tokenDetailHandler(c *core.RequestEvent) error {
	address := c.Request.PathValue("address")
	if address == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "address required"})
	}

	lower := strings.ToLower(address)

	tokenRows, err := c.App.FindRecordsByFilter("token_analytics",
		"token_address = {:a}", "", 1, 0,
		map[string]any{"a": lower})
	if err != nil || len(tokenRows) == 0 {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "token not found"})
	}

	tokenAddr := strings.ToLower(tokenRows[0].GetString("token_address"))

	// Fetch recent transfers for this token
	transfers, err := c.App.FindRecordsByFilter("transfers",
		"token_address = {:a}", "-block_number", 50, 0,
		map[string]any{"a": tokenAddr})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"token":     tokenRows[0].PublicExport(),
		"transfers": recordsToMaps(transfers),
	})
}

// API_DESC Agent leaderboard ranked by stablecoin volume transferred
// API_TAGS Agents
func analyticsAgentLeaderboardHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 60)
	limit, _ := limitOffset(c)
	if limit > 100 {
		limit = 100
	}

	// Batch query: get job stats for ALL agents in a single SQL aggregation
	// instead of N+1 queries per agent.
	type jobAgg struct {
		Addr         string  `db:"addr"`
		JobCount     int     `db:"job_count"`
		TotalEscrow  float64 `db:"total_escrow"`
		PaidJobs     int     `db:"paid_jobs"`
		RejectedJobs int     `db:"rejected_jobs"`
	}
	var jobStats []jobAgg
	_ = c.App.DB().NewQuery(`
		SELECT addr, SUM(job_count) as job_count, SUM(total_escrow) as total_escrow,
		       SUM(paid_jobs) as paid_jobs, SUM(rejected_jobs) as rejected_jobs
		FROM (
		    SELECT employer_address as addr, COUNT(*) as job_count,
		           COALESCE(SUM(CAST(payment_usdc AS REAL)), 0) as total_escrow,
		           SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) as paid_jobs,
		           SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) as rejected_jobs
		    FROM agent_jobs GROUP BY employer_address
		    UNION ALL
		    SELECT worker_address as addr, COUNT(*) as job_count,
		           COALESCE(SUM(CAST(payment_usdc AS REAL)), 0) as total_escrow,
		           SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) as paid_jobs,
		           SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) as rejected_jobs
		    FROM agent_jobs GROUP BY worker_address
		) GROUP BY addr`).All(&jobStats)

	statsMap := make(map[string]*jobAgg, len(jobStats))
	for i := range jobStats {
		statsMap[jobStats[i].Addr] = &jobStats[i]
	}

	// Index-backed sort on the numeric mirror column avoids the old in-memory
	// sprintf+ParseFloat-per-agent pass and lets us cap the result set in SQL.
	agents, err := c.App.FindRecordsByFilter("agents", "", "-usdc_transferred_num", limit, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	result := make([]map[string]any, 0, len(agents))
	for _, a := range agents {
		entry := enrichAgentRecord(a)
		addr := a.GetString("agent_address")
		if s, ok := statsMap[addr]; ok {
			entry["job_count"] = s.JobCount
			entry["total_escrow"] = s.TotalEscrow
			entry["paid_jobs"] = s.PaidJobs
			entry["rejected_jobs"] = s.RejectedJobs
		} else {
			entry["job_count"] = 0
			entry["total_escrow"] = 0.0
			entry["paid_jobs"] = 0
			entry["rejected_jobs"] = 0
		}
		result = append(result, entry)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"leaderboard": result,
		"count":       len(result),
	})
}
