package jobs

import (
	"log"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

// RunTokenAnalytics computes per-token aggregated stats from the transfers table
// and enriches them with onchain metadata (name, symbol, decimals, totalSupply).
func RunTokenAnalytics(app core.App) {
	log.Println("[token-analytics] starting token analytics job")

	type tokenAgg struct {
		TokenAddress    string `db:"token_address"`
		TransferCount   int    `db:"transfer_count"`
		FirstBlock      int    `db:"first_block"`
		LastBlock       int    `db:"last_block"`
		UniqueSenders   int    `db:"unique_senders"`
		UniqueReceivers int    `db:"unique_receivers"`
	}

	var aggResults []tokenAgg
	err := app.DB().Select(
		"token_address",
		"count(*) as transfer_count",
		"min(block_number) as first_block",
		"max(block_number) as last_block",
		"count(distinct from_addr) as unique_senders",
		"count(distinct to_addr) as unique_receivers",
	).From("transfers").
		GroupBy("token_address").
		OrderBy("transfer_count DESC").
		All(&aggResults)

	if err != nil {
		log.Printf("[token-analytics] aggregation query failed: %v", err)
		return
	}

	log.Printf("[token-analytics] found %d unique tokens in transfers", len(aggResults))

	coll, err := app.FindCollectionByNameOrId("token_analytics")
	if err != nil {
		log.Printf("[token-analytics] token_analytics collection not found: %v", err)
		return
	}

	for i, agg := range aggResults {
		addr := strings.ToLower(agg.TokenAddress)

		existing, _ := app.FindRecordsByFilter("token_analytics",
			"LOWER(token_address) = {:a}", "", 1, 0,
			map[string]any{"a": addr})

		info := utils.FetchFullTokenInfo(parseAddr(agg.TokenAddress))

		var r *core.Record
		if len(existing) > 0 {
			r = existing[0]
		} else {
			r = core.NewRecord(coll)
			r.Set("token_address", strings.ToLower(agg.TokenAddress))
		}

		r.Set("symbol", info.Symbol)
		r.Set("name", info.Name)
		r.Set("decimals", int(info.Decimals))
		r.Set("lookup_failed", info.LookupFailed)
		r.Set("transfer_count", agg.TransferCount)
		r.Set("first_seen_block", agg.FirstBlock)
		r.Set("last_seen_block", agg.LastBlock)
		r.Set("unique_senders", agg.UniqueSenders)
		r.Set("unique_receivers", agg.UniqueReceivers)

		if info.TotalSupply != nil {
			r.Set("total_supply_raw", info.TotalSupply.String())
			if !info.LookupFailed && info.Decimals > 0 {
				r.Set("total_supply_human", utils.TokenAmountHuman(info.TotalSupply, info.Decimals))
			}
		}

		if err := app.Save(r); err != nil {
			log.Printf("[token-analytics] failed to save %s: %v", addr, err)
			continue
		}

		if i > 0 && i%10 == 0 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	log.Printf("[token-analytics] completed: processed %d tokens", len(aggResults))
}

func parseAddr(hex string) common.Address {
	return common.HexToAddress(hex)
}

// StartTokenAnalyticsScheduler runs the analytics job periodically.
func StartTokenAnalyticsScheduler(app core.App) {
	const initialDelay = 30 * time.Second
	const interval = 10 * time.Minute

	go func() {
		time.Sleep(initialDelay)
		for {
			RunTokenAnalytics(app)
			time.Sleep(interval)
		}
	}()
}
