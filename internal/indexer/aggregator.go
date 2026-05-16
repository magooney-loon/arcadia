package indexer

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/dbx"
)

// edgeUpsertChunk caps the number of edge rows per bulk INSERT. SQLite's
// default parameter limit is 32k; at 7 columns per row, 1024 is comfortable
// and keeps query strings short.
const edgeUpsertChunk = 1024

// blockAcc accumulates per-block statistics during a batch so that block_stats
// can be written in one pass after all transactions and logs are processed.
type blockAcc struct {
	txCount         int
	uniqueSenders   map[string]struct{}
	uniqueReceivers map[string]struct{}
	newContracts    int
	totalFee        *big.Int
	totalUSDC       *big.Int
	totalEURC       *big.Int
	totalUSYC       *big.Int
	largestUSDC     *big.Int
}

func newBlockAcc() *blockAcc {
	return &blockAcc{
		uniqueSenders:   make(map[string]struct{}),
		uniqueReceivers: make(map[string]struct{}),
		totalFee:        new(big.Int),
		totalUSDC:       new(big.Int),
		totalEURC:       new(big.Int),
		totalUSYC:       new(big.Int),
		largestUSDC:     new(big.Int),
	}
}

// agentDelta accumulates per-agent (wallet address) updates for a batch so
// that the agents table can be updated with one query per address.
type agentDelta struct {
	feeWei      *big.Int
	transferred *big.Int
	txCount     int
}

func newAgentDelta() *agentDelta {
	return &agentDelta{feeWei: new(big.Int), transferred: new(big.Int)}
}

// flushEdgeDeltas applies the per-batch edge aggregator to wallet_edges using
// one bulk INSERT … ON CONFLICT DO UPDATE per chunk. Big.Int arithmetic stays
// in Go so we don't depend on SQLite INTEGER bounds for total_usdc.
func flushEdgeDeltas(app core.App, deltas map[edgeKey]*edgeDelta) error {
	if len(deltas) == 0 {
		return nil
	}
	keys := make([]edgeKey, 0, len(deltas))
	for k := range deltas {
		keys = append(keys, k)
	}
	existing, err := loadEdgesFor(app, keys)
	if err != nil {
		return err
	}

	// Resolve final values per edge (existing + delta) ahead of the upsert.
	type row struct {
		from, to                 string
		total                    string
		txCount                  int
		lastSeen                 uint64
		fromIsAgent, toIsAgent   bool
	}
	rows := make([]row, 0, len(deltas))
	for key, d := range deltas {
		var total *big.Int
		var txCount int
		var lastSeen uint64
		if r, hit := existing[key]; hit {
			prev, _ := new(big.Int).SetString(r.TotalUsdc, 10)
			if prev == nil {
				prev = new(big.Int)
			}
			total = new(big.Int).Add(prev, d.total)
			txCount = r.TxCount + d.count
			lastSeen = r.LastSeenBlock
			if d.lastSeen > lastSeen {
				lastSeen = d.lastSeen
			}
		} else {
			total = new(big.Int).Set(d.total)
			txCount = d.count
			lastSeen = d.lastSeen
		}
		rows = append(rows, row{
			from: key.from, to: key.to,
			total: total.String(), txCount: txCount, lastSeen: lastSeen,
			fromIsAgent: d.fromIsAgent, toIsAgent: d.toIsAgent,
		})
	}

	for start := 0; start < len(rows); start += edgeUpsertChunk {
		end := start + edgeUpsertChunk
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[start:end]

		// Build "(?,?,?,?,?,?,?),(?,?,?,?,?,?,?),..."
		placeholders := make([]string, len(chunk))
		params := dbx.Params{}
		for i, r := range chunk {
			placeholders[i] = fmt.Sprintf("({:f%d},{:t%d},{:u%d},{:c%d},{:l%d},{:fa%d},{:ta%d})", i, i, i, i, i, i, i)
			params[fmt.Sprintf("f%d", i)] = r.from
			params[fmt.Sprintf("t%d", i)] = r.to
			params[fmt.Sprintf("u%d", i)] = r.total
			params[fmt.Sprintf("c%d", i)] = r.txCount
			params[fmt.Sprintf("l%d", i)] = r.lastSeen
			params[fmt.Sprintf("fa%d", i)] = r.fromIsAgent
			params[fmt.Sprintf("ta%d", i)] = r.toIsAgent
		}

		// ON CONFLICT only refreshes accumulating columns. from_is_agent /
		// to_is_agent stay at their first-insert values (current behaviour).
		sql := `INSERT INTO wallet_edges
			(from_wallet, to_wallet, total_usdc, tx_count, last_seen_block, from_is_agent, to_is_agent)
			VALUES ` + strings.Join(placeholders, ",") + `
			ON CONFLICT(from_wallet, to_wallet) DO UPDATE SET
				total_usdc = excluded.total_usdc,
				tx_count = excluded.tx_count,
				last_seen_block = excluded.last_seen_block`
		if _, err := app.DB().NewQuery(sql).Bind(params).Execute(); err != nil {
			return fmt.Errorf("bulk upsert wallet_edges (%d rows): %w", len(chunk), err)
		}
	}
	return nil
}
