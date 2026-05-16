package handlers

// API_SOURCE

import (
	"net/http"
	"sync"

	"arcadia/internal/repo"

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
		agentRecord          *core.Record
	)
	run := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}
	run(func() {
		sent, _ = repo.TransfersBySender(c.App, address, limit, offset)
	})
	run(func() {
		received, _ = repo.TransfersByReceiver(c.App, address, limit, offset)
	})
	run(func() {
		outEdges, _ = repo.EdgesByFromWallet(c.App, address, 20, 0)
	})
	run(func() {
		inEdges, _ = repo.EdgesByToWallet(c.App, address, 20, 0)
	})
	run(func() {
		txsSent, _ = repo.TransactionsBySender(c.App, address, limit, offset)
	})
	run(func() {
		txsReceived, _ = repo.TransactionsByReceiver(c.App, address, limit, offset)
	})
	run(func() {
		agentRecord, _ = repo.AgentByAddress(c.App, address)
	})
	wg.Wait()

	var agentData any
	if agentRecord != nil {
		agentData = enrichAgentRecord(agentRecord)
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
	wallet := qp(c, "wallet", "")

	records, err := repo.EdgesByWallet(c.App, wallet, limit, offset)
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
