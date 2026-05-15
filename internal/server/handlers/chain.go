package handlers

// API_SOURCE

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

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
