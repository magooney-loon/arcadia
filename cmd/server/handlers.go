package main

// API_SOURCE

import (
	"net/http"
	"strconv"

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

// API_DESC Token transfers — filterable by token symbol, sender, or receiver
// API_TAGS Transfers
func transfersHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	if token := qp(c, "token", ""); token != "" {
		filter = "token_symbol = {:sym}"
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
