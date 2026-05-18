package jobs

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
	"arcadia/internal/rpc"
	"arcadia/internal/utils"
)

// RunTokenAnalytics computes per-token aggregated stats from the transfers table
// and enriches them with onchain metadata (name, symbol, decimals, totalSupply).
// Tokens that already have cached metadata skip the RPC calls entirely.
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

	// Single-pass preload: one query instead of N FindRecordsByFilter calls.
	existing, err := repo.AllTokenAnalytics(app)
	if err != nil {
		log.Printf("[token-analytics] preload failed: %v", err)
		return
	}
	byAddr := make(map[string]*core.Record, len(existing))
	for _, r := range existing {
		byAddr[r.GetString("token_address")] = r
	}

	// Worker pool for concurrent RPC enrichment.
	type workItem struct {
		agg tokenAgg
		r   *core.Record
	}
	type result struct {
		r *core.Record
	}

	const workers = 6
	workCh := make(chan workItem, len(aggResults))
	resultCh := make(chan result, len(aggResults))

	var wg sync.WaitGroup
	// Per-worker throttle: each tracks its own rpc call count so we don't
	// serialise a global counter; the 500 ms pause every 10 RPC calls
	// distributes naturally across workers.
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rpcCalls := 0
			for item := range workCh {
				r := item.r
				agg := item.agg
				addr := common.HexToAddress(agg.TokenAddress)
				hasMetadata := r.GetString("symbol") != "" &&
					r.GetInt("decimals") > 0 &&
					!r.GetBool("lookup_failed")

				if !hasMetadata {
					info := rpc.FetchFullTokenInfo(addr)
					r.Set("symbol", info.Symbol)
					r.Set("name", info.Name)
					r.Set("decimals", int(info.Decimals))
					r.Set("token_type", info.TokenType)
					r.Set("lookup_failed", info.LookupFailed)
					if info.TotalSupply != nil {
						r.Set("total_supply_raw", info.TotalSupply.String())
						if !info.LookupFailed && info.Decimals > 0 {
							r.Set("total_supply_human", utils.TokenAmountHuman(info.TotalSupply, info.Decimals))
						}
					}
					rpcCalls++
					if rpcCalls%10 == 0 {
						time.Sleep(500 * time.Millisecond)
					}
				}

				r.Set("transfer_count", agg.TransferCount)
				r.Set("first_seen_block", agg.FirstBlock)
				r.Set("last_seen_block", agg.LastBlock)
				r.Set("unique_senders", agg.UniqueSenders)
				r.Set("unique_receivers", agg.UniqueReceivers)
				resultCh <- result{r: r}
			}
		}()
	}

	// Feed workers.
	for _, agg := range aggResults {
		addr := agg.TokenAddress
		r, exists := byAddr[addr]
		if !exists {
			r = core.NewRecord(coll)
			r.Set("token_address", addr)
		}
		workCh <- workItem{agg: agg, r: r}
	}
	close(workCh)

	// Close resultCh once all workers finish.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Drain results into a slice, then save in a single transaction.
	records := make([]*core.Record, 0, len(aggResults))
	for res := range resultCh {
		records = append(records, res.r)
	}

	saved, failed := 0, 0
	err = app.RunInTransaction(func(txApp core.App) error {
		for _, r := range records {
			if err := txApp.Save(r); err != nil {
				log.Printf("[token-analytics] failed to save %s: %v", r.GetString("token_address"), err)
				failed++
			} else {
				saved++
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[token-analytics] transaction failed: %v", err)
		return
	}

	log.Printf("[token-analytics] completed: %d saved, %d failed", saved, failed)
}

// RegisterTokenAnalyticsJob registers the token analytics job with the
// pb-ext job manager (visible in the admin UI, logged execution history).
func RegisterTokenAnalyticsJob(app core.App) error {
	jm := jobs.GetManager()
	if jm == nil {
		return fmt.Errorf("job manager not initialized")
	}

	return jm.RegisterJob(
		"tokenAnalytics",
		"Token Analytics",
		"Aggregates per-token stats from transfers and enriches with on-chain metadata",
		"*/10 * * * *",
		func(el *jobs.ExecutionLogger) {
			el.Start("Token Analytics")

			var before struct {
				Count int `db:"cnt"`
			}
			_ = app.DB().NewQuery("SELECT COUNT(*) AS cnt FROM token_analytics").One(&before)

			RunTokenAnalytics(app)

			var after struct {
				Count int `db:"cnt"`
			}
			_ = app.DB().NewQuery("SELECT COUNT(*) AS cnt FROM token_analytics").One(&after)

			el.Statistics(map[string]interface{}{
				"tokens_before": before.Count,
				"tokens_after":  after.Count,
			})
			el.Complete(fmt.Sprintf("Token analytics done — %d tokens", after.Count))
		},
	)
}
