package indexer

import (
	"fmt"
	"math/big"

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

// loadEdgesFor prefetches wallet_edges rows for the (from,to) pairs we will
// touch. Used by the deferred edge-flush in processBatch.
func loadEdgesFor(app core.App, keys []edgeKey) (map[edgeKey]*core.Record, error) {
	out := map[edgeKey]*core.Record{}
	if len(keys) == 0 {
		return out, nil
	}
	// distinct from-wallets — query in one filter using IN-style OR.
	froms := map[string]struct{}{}
	for _, k := range keys {
		froms[k.from] = struct{}{}
	}
	want := map[edgeKey]struct{}{}
	for _, k := range keys {
		want[k] = struct{}{}
	}
	fromList := make([]any, 0, len(froms))
	for f := range froms {
		fromList = append(fromList, f)
	}
	// PocketBase's filter DSL doesn't speak `IN`; the dbx layer underneath
	// does. Use raw SQL with `IN` and quote-safe placeholders via dbx.
	params := dbx.Params{}
	in := ""
	for i, f := range fromList {
		key := fmt.Sprintf("f%d", i)
		params[key] = f
		if i > 0 {
			in += ","
		}
		in += "{:" + key + "}"
	}
	rows, err := app.FindRecordsByFilter(
		"wallet_edges",
		"from_wallet IN ("+in+")",
		"", 0, 0, params)
	if err != nil {
		return nil, fmt.Errorf("load edges: %w", err)
	}
	for _, r := range rows {
		k := edgeKey{r.GetString("from_wallet"), r.GetString("to_wallet")}
		if _, hit := want[k]; hit {
			out[k] = r
		}
	}
	return out, nil
}

