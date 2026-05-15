package handlers

// API_SOURCE

import (
	"net/http"
	"sync"

	"github.com/pocketbase/pocketbase/core"
)

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
		wg                   sync.WaitGroup
		sent, received       []*core.Record
		outEdges, inEdges    []*core.Record
		txsSent, txsReceived []*core.Record
		agentRecords         []*core.Record
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
