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

	// agent status
	agentRecords, _ := c.App.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": address})
	isAgent := len(agentRecords) > 0

	return c.JSON(http.StatusOK, map[string]any{
		"address":       address,
		"is_agent":      isAgent,
		"sent":          recordsToMaps(sent),
		"received":      recordsToMaps(received),
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
	if proto := qp(c, "protocol", ""); proto != "" {
		filter = "protocol = {:p}"
		params["p"] = proto
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

// API_DESC Recent StableFX USDC↔EURC swap events
// API_TAGS FX
func fxHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)
	records, err := c.App.FindRecordsByFilter("fx_swaps", "", "-block_number", limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"swaps": recordsToMaps(records),
		"count": len(records),
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

	records, err := c.App.FindRecordsByFilter("agent_jobs", filter, "-created_at_block", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"jobs":  recordsToMaps(records),
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
