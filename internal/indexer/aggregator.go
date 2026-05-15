package indexer

import (
	"fmt"
	"math/big"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

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

// flushEdgeDeltas applies the per-batch edge aggregator to wallet_edges in one
// prefetch + bulk save pass instead of SELECT+SAVE per Transfer log.
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

	coll, err := utils.FindCollection(app, "wallet_edges")
	if err != nil {
		return err
	}
	for key, d := range deltas {
		r, hit := existing[key]
		if hit {
			prev, _ := new(big.Int).SetString(r.GetString("total_usdc"), 10)
			if prev == nil {
				prev = new(big.Int)
			}
			r.Set("total_usdc", new(big.Int).Add(prev, d.total).String())
			r.Set("tx_count", r.GetInt("tx_count")+d.count)
			if d.lastSeen > 0 {
				r.Set("last_seen_block", d.lastSeen)
			}
		} else {
			r = core.NewRecord(coll)
			r.Set("from_wallet", key.from)
			r.Set("to_wallet", key.to)
			r.Set("total_usdc", d.total.String())
			r.Set("tx_count", d.count)
			if d.lastSeen > 0 {
				r.Set("last_seen_block", d.lastSeen)
			}
			if d.fromIsAgent {
				r.Set("from_is_agent", true)
			}
			if d.toIsAgent {
				r.Set("to_is_agent", true)
			}
		}
		if err := app.Save(r); err != nil {
			return fmt.Errorf("save wallet edge %s -> %s: %w", key.from, key.to, err)
		}
	}
	return nil
}
