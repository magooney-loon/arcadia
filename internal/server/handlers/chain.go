package handlers

// API_SOURCE

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"

	"arcadia/internal/server/cache"
)

// API_DESC Recent blocks with derived stats
// API_TAGS Chain
func blocksHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)
	if cached, ok := cache.Default.Get("blocks:" + strconv.Itoa(limit)); ok {
		return c.JSON(http.StatusOK, cached)
	}
	records, err := repo.ListBlocks(c.App, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	total, _ := repo.CountBlocks(c.App)
	return c.JSON(http.StatusOK, map[string]any{
		"blocks": recordsToMaps(records),
		"count":  len(records),
		"total":  total,
	})
}

// API_DESC Recent transactions
// API_TAGS Chain
func transactionsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)
	if cached, ok := cache.Default.Get("transactions:" + strconv.Itoa(limit)); ok {
		return c.JSON(http.StatusOK, cached)
	}

	f := repo.TransactionFilter{}
	if block := qp(c, "block", ""); block != "" {
		if n, err := strconv.ParseInt(block, 10, 64); err == nil {
			f.BlockNumber = n
		}
	}
	if from := qp(c, "from", ""); from != "" {
		f.FromAddr = from
	}
	if to := qp(c, "to", ""); to != "" {
		f.ToAddr = to
	}

	records, err := repo.ListTransactions(c.App, f, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	total, _ := repo.CountTransactions(c.App, f)
	return c.JSON(http.StatusOK, map[string]any{
		"transactions": recordsToMaps(records),
		"count":        len(records),
		"total":        total,
	})
}

// API_DESC Internal contract-to-contract calls — filterable by tx hash or address
// API_TAGS Chain
func tracesHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	f := repo.TraceFilter{}
	if tx := qp(c, "tx_hash", ""); tx != "" {
		f.TxHash = tx
	}
	if from := qp(c, "from", ""); from != "" {
		f.FromAddr = from
	}
	if to := qp(c, "to", ""); to != "" {
		f.ToAddr = to
	}

	records, err := repo.ListTraces(c.App, f, limit, offset)
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
		tx, _ := repo.TransactionByHash(c.App, q)
		if tx != nil {
			return c.JSON(http.StatusOK, map[string]any{"type": "tx", "result": tx.PublicExport()})
		}
		return c.JSON(http.StatusOK, map[string]any{"type": "not_found"})
	}

	if len(q) == 42 && strings.HasPrefix(q, "0x") {
		agent, _ := repo.AgentByAddress(c.App, q)
		if agent != nil {
			return c.JSON(http.StatusOK, map[string]any{
				"type":   "agent",
				"result": map[string]any{"address": q, "is_agent": true, "agent": agent.PublicExport()},
			})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"type":   "wallet",
			"result": map[string]any{"address": q, "is_agent": false},
		})
	}

	if isNumeric(q) {
		num, _ := strconv.Atoi(q)
		block, _ := repo.BlockByNumber(c.App, int64(num))
		if block != nil {
			return c.JSON(http.StatusOK, map[string]any{"type": "block", "result": block.PublicExport()})
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

	tx, err := repo.TransactionByHash(c.App, hash)
	if err != nil || tx == nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "transaction not found"})
	}

	transfers, _ := repo.TransfersByTxHash(c.App, hash)
	traces, _ := repo.TracesByTxHash(c.App, hash)

	return c.JSON(http.StatusOK, map[string]any{
		"transaction": tx.PublicExport(),
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

	block, err := repo.BlockByNumber(c.App, int64(num))
	if err != nil || block == nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "block not found"})
	}

	txs, _ := repo.TransactionsByBlock(c.App, int64(num))
	stat, _ := repo.BlockStatsByNumber(c.App, int64(num))

	result := map[string]any{
		"block":        block.PublicExport(),
		"transactions": recordsToMaps(txs),
	}
	if stat != nil {
		result["stats"] = stat.PublicExport()
	}
	return c.JSON(http.StatusOK, result)
}
