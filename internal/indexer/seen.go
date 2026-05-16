package indexer

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// txLogKey identifies a single event log (transfer / crosschain) by its
// transaction hash and log index — matches the unique indexes on those tables.
type txLogKey struct {
	hash string
	idx  uint64
}

// edgeKey identifies a directed wallet→wallet edge in wallet_edges.
type edgeKey struct {
	from, to string
}

// batchSeen holds dedupe sets and existing-record lookups for one batch.
// All keys come from a single range query per table, replacing per-row
// SELECT-before-INSERT (the N+1 problem in the old hot path).
type batchSeen struct {
	blocks     map[uint64]struct{}
	txs        map[string]struct{}
	transfers  map[txLogKey]struct{}
	crosschain map[txLogKey]struct{}
	agents     map[string]struct{}
	// upsert collections — keep the record so we can mutate in place
	jobs  map[string]*core.Record
	fx    map[string]*core.Record
	edges map[edgeKey]*core.Record
	// blocks created in this batch, keyed by number, for in-memory backfill.
	newBlocks map[uint64]*core.Record
}

// edgeDelta accumulates per-edge updates for a batch so wallet_edges takes one
// upsert per (from,to) pair instead of one per transfer log.
type edgeDelta struct {
	total       *big.Int
	count       int
	lastSeen    uint64
	fromIsAgent bool
	toIsAgent   bool
}

func newBatchSeen() *batchSeen {
	return &batchSeen{
		blocks:     map[uint64]struct{}{},
		txs:        map[string]struct{}{},
		transfers:  map[txLogKey]struct{}{},
		crosschain: map[txLogKey]struct{}{},
		agents:     map[string]struct{}{},
		jobs:       map[string]*core.Record{},
		fx:         map[string]*core.Record{},
		edges:      map[edgeKey]*core.Record{},
		newBlocks:  map[uint64]*core.Record{},
	}
}

// loadBatchSeen scans the per-table block range once each and returns the
// dedupe sets the batch needs. Every table queried here is indexed on
// block_number so each lookup is an index range scan, not a table scan.
func loadBatchSeen(app core.App, res *types.QueryResponse) (*batchSeen, error) {
	bs := newBatchSeen()

	fromBlock, toBlock, ok := blockRangeOf(res)
	if !ok {
		return bs, nil
	}

	// ── blocks ─────────────────────────────────────────────────────────────
	type numRow struct {
		N uint64 `db:"number"`
	}
	var blkRows []numRow
	if err := app.DB().NewQuery(
		"SELECT number FROM blocks WHERE number BETWEEN {:from} AND {:to}").
		Bind(dbx.Params{"from": fromBlock, "to": toBlock}).All(&blkRows); err != nil {
		return nil, fmt.Errorf("load seen blocks: %w", err)
	}
	for _, r := range blkRows {
		bs.blocks[r.N] = struct{}{}
	}

	// ── transactions ───────────────────────────────────────────────────────
	type hashRow struct {
		H string `db:"hash"`
	}
	var txRows []hashRow
	if err := app.DB().NewQuery(
		"SELECT hash FROM transactions WHERE block_number BETWEEN {:from} AND {:to}").
		Bind(dbx.Params{"from": fromBlock, "to": toBlock}).All(&txRows); err != nil {
		return nil, fmt.Errorf("load seen txs: %w", err)
	}
	for _, r := range txRows {
		bs.txs[r.H] = struct{}{}
	}

	// ── transfers ──────────────────────────────────────────────────────────
	type txLogRow struct {
		H string `db:"tx_hash"`
		I uint64 `db:"log_index"`
	}
	var trRows []txLogRow
	if err := app.DB().NewQuery(
		"SELECT tx_hash, log_index FROM transfers WHERE block_number BETWEEN {:from} AND {:to}").
		Bind(dbx.Params{"from": fromBlock, "to": toBlock}).All(&trRows); err != nil {
		return nil, fmt.Errorf("load seen transfers: %w", err)
	}
	for _, r := range trRows {
		bs.transfers[txLogKey{r.H, r.I}] = struct{}{}
	}

	// ── crosschain ─────────────────────────────────────────────────────────
	var ccRows []txLogRow
	if err := app.DB().NewQuery(
		"SELECT tx_hash, log_index FROM crosschain_events WHERE block_number BETWEEN {:from} AND {:to}").
		Bind(dbx.Params{"from": fromBlock, "to": toBlock}).All(&ccRows); err != nil {
		return nil, fmt.Errorf("load seen crosschain: %w", err)
	}
	for _, r := range ccRows {
		bs.crosschain[txLogKey{r.H, r.I}] = struct{}{}
	}

	// ── agents prefetch (registrations are rare; the addresses we may insert
	//    come from Transfer logs from address(0) on the registry contract) ──
	// Cheap enough to load every known agent address: only 381 today.
	type addrRow struct {
		A string `db:"agent_address"`
	}
	var agentRows []addrRow
	if err := app.DB().NewQuery("SELECT agent_address FROM agents").All(&agentRows); err != nil {
		return nil, fmt.Errorf("load seen agents: %w", err)
	}
	for _, r := range agentRows {
		bs.agents[r.A] = struct{}{}
	}

	return bs, nil
}

// blockRangeOf returns the inclusive [min, max] block_number span of a batch.
func blockRangeOf(res *types.QueryResponse) (uint64, uint64, bool) {
	var lo, hi uint64
	has := false
	consider := func(n *big.Int) {
		if n == nil {
			return
		}
		v := n.Uint64()
		if !has {
			lo, hi = v, v
			has = true
			return
		}
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	for _, b := range res.Data.Blocks {
		consider(b.Number)
	}
	for _, t := range res.Data.Transactions {
		consider(t.BlockNumber)
	}
	for _, l := range res.Data.Logs {
		consider(l.BlockNumber)
	}
	return lo, hi, has
}

// existingEdge holds just the columns flushEdgeDeltas needs to accumulate
// against — avoids hydrating full *core.Records.
type existingEdge struct {
	TotalUsdc     string
	TxCount       int
	LastSeenBlock uint64
}

// loadEdgesFor prefetches wallet_edges rows for the (from,to) pairs we will
// touch. Used by the deferred edge-flush in processBatch.
//
// Previously this loaded every edge for every distinct from_wallet, which
// blew up for hot addresses (DEXes, exchanges) once wallet_edges grew. Now
// we match on the exact (from, to) tuples via row-value IN, and select only
// the three accumulating columns directly — no record hydration.
func loadEdgesFor(app core.App, keys []edgeKey) (map[edgeKey]existingEdge, error) {
	out := map[edgeKey]existingEdge{}
	if len(keys) == 0 {
		return out, nil
	}

	// Chunk to stay well under SQLite's parameter limit. 2 params per pair,
	// 1024 pairs = 2048 params per query.
	const chunk = 1024
	for start := 0; start < len(keys); start += chunk {
		end := start + chunk
		if end > len(keys) {
			end = len(keys)
		}
		slice := keys[start:end]

		params := dbx.Params{}
		tuples := make([]string, len(slice))
		for i, k := range slice {
			pf := fmt.Sprintf("ef%d", i)
			pt := fmt.Sprintf("et%d", i)
			params[pf] = k.from
			params[pt] = k.to
			tuples[i] = fmt.Sprintf("({:%s},{:%s})", pf, pt)
		}

		type row struct {
			From    string `db:"from_wallet"`
			To      string `db:"to_wallet"`
			Total   string `db:"total_usdc"`
			TxCount int    `db:"tx_count"`
			LastSee uint64 `db:"last_seen_block"`
		}
		var rows []row
		sql := "SELECT from_wallet, to_wallet, total_usdc, tx_count, last_seen_block " +
			"FROM wallet_edges WHERE (from_wallet, to_wallet) IN (" +
			strings.Join(tuples, ",") + ")"
		if err := app.DB().NewQuery(sql).Bind(params).All(&rows); err != nil {
			return nil, fmt.Errorf("load edges: %w", err)
		}
		for _, r := range rows {
			out[edgeKey{r.From, r.To}] = existingEdge{
				TotalUsdc: r.Total, TxCount: r.TxCount, LastSeenBlock: r.LastSee,
			}
		}
	}
	return out, nil
}
