package jobs

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	arc "arcadia/internal/chain/arc"
	"arcadia/internal/repo"
	"arcadia/internal/utils"
)

const (
	metaTokenAnalyticsCursor = "tokenAnalytics:lastBlock"
	metaTokenAnalyticsUnique = "tokenAnalytics:lastUniqueRecompute"
)

// uniqueRecomputeInterval is how often the expensive per-token
// COUNT(DISTINCT from_addr/to_addr) scan runs. Between recomputes the previously
// stored unique_senders/unique_receivers values are kept verbatim.
const uniqueRecomputeInterval = 24 * time.Hour

type tokenDelta struct {
	TokenAddress  string `db:"token_address"`
	TransferCount int    `db:"transfer_count"`
	FirstBlock    int    `db:"first_block"`
	LastBlock     int    `db:"last_block"`
}

// RunTokenAnalytics aggregates per-token stats incrementally. Only transfers
// newer than the stored cursor are scanned each invocation; previous totals are
// added to the deltas. New token addresses surfaced by the delta scan are
// inserted and enriched via RPC. The unique_senders / unique_receivers fields
// are only refreshed once per uniqueRecomputeInterval since they cannot be
// computed incrementally without a per-address side table.
func RunTokenAnalytics(app core.App) {
	if !heavyJobMu.TryLock() {
		log.Println("[token-analytics] another heavy analytics job is running, skipping")
		return
	}
	defer heavyJobMu.Unlock()

	started := time.Now()
	cursor := readTokenCursor(app)

	var deltas []tokenDelta
	err := app.DB().NewQuery(
		`SELECT token_address,
		        COUNT(*)          AS transfer_count,
		        MIN(block_number) AS first_block,
		        MAX(block_number) AS last_block
		 FROM transfers
		 WHERE block_number > {:cursor}
		 GROUP BY token_address`).
		Bind(dbx.Params{"cursor": cursor}).
		All(&deltas)
	if err != nil {
		log.Printf("[token-analytics] delta aggregation failed: %v", err)
		return
	}
	log.Printf("[token-analytics] cursor=%d deltas=%d", cursor, len(deltas))

	coll, err := app.FindCollectionByNameOrId("token_analytics")
	if err != nil {
		log.Printf("[token-analytics] token_analytics collection not found: %v", err)
		return
	}

	existing, err := repo.AllTokenAnalytics(app)
	if err != nil {
		log.Printf("[token-analytics] preload failed: %v", err)
		return
	}
	byAddr := make(map[string]*core.Record, len(existing))
	for _, r := range existing {
		byAddr[r.GetString("token_address")] = r
	}

	enrichAddrs := make([]string, 0, len(deltas))
	updated := make([]*core.Record, 0, len(deltas))
	highestBlock := cursor
	for _, d := range deltas {
		if d.LastBlock > highestBlock {
			highestBlock = d.LastBlock
		}
		r, exists := byAddr[d.TokenAddress]
		if !exists {
			r = core.NewRecord(coll)
			r.Set("token_address", d.TokenAddress)
			r.Set("transfer_count", d.TransferCount)
			r.Set("first_seen_block", d.FirstBlock)
			r.Set("last_seen_block", d.LastBlock)
			byAddr[d.TokenAddress] = r
		} else {
			r.Set("transfer_count", r.GetInt("transfer_count")+d.TransferCount)
			if prev := r.GetInt("first_seen_block"); prev == 0 || d.FirstBlock < prev {
				r.Set("first_seen_block", d.FirstBlock)
			}
			if d.LastBlock > r.GetInt("last_seen_block") {
				r.Set("last_seen_block", d.LastBlock)
			}
		}
		updated = append(updated, r)
		enrichAddrs = append(enrichAddrs, d.TokenAddress)
	}

	if len(enrichAddrs) > 0 {
		enrichTokenMetadata(byAddr, enrichAddrs)
	}

	if shouldRecomputeUnique(app) {
		recomputeUniqueCounts(app, byAddr)
		updated = updated[:0]
		for _, r := range byAddr {
			updated = append(updated, r)
		}
		_ = utils.SetMetaValue(app, metaTokenAnalyticsUnique, strconv.FormatInt(time.Now().Unix(), 10))
	}

	saved, failed := persistTokenRecords(app, updated)
	if highestBlock > cursor {
		if err := utils.SetMetaValue(app, metaTokenAnalyticsCursor, strconv.Itoa(highestBlock)); err != nil {
			log.Printf("[token-analytics] cursor save failed: %v", err)
		}
	}
	log.Printf("[token-analytics] done in %s: %d saved, %d failed", time.Since(started).Round(time.Millisecond), saved, failed)
}

func readTokenCursor(app core.App) int {
	raw := utils.GetMetaValue(app, metaTokenAnalyticsCursor)
	if raw == "" {
		return 0
	}
	v, _ := strconv.Atoi(raw)
	return v
}

func shouldRecomputeUnique(app core.App) bool {
	raw := utils.GetMetaValue(app, metaTokenAnalyticsUnique)
	if raw == "" {
		return true
	}
	ts, _ := strconv.ParseInt(raw, 10, 64)
	return time.Since(time.Unix(ts, 0)) >= uniqueRecomputeInterval
}

// recomputeUniqueCounts runs the expensive per-token COUNT(DISTINCT) scan and
// writes the result back to byAddr. This is the only path that touches the
// entire transfers table; called at most once per uniqueRecomputeInterval.
func recomputeUniqueCounts(app core.App, byAddr map[string]*core.Record) {
	log.Println("[token-analytics] running daily unique-address recompute")
	type uniqRow struct {
		TokenAddress    string `db:"token_address"`
		UniqueSenders   int    `db:"unique_senders"`
		UniqueReceivers int    `db:"unique_receivers"`
	}
	var rows []uniqRow
	err := app.DB().NewQuery(
		`SELECT token_address,
		        COUNT(DISTINCT from_addr) AS unique_senders,
		        COUNT(DISTINCT to_addr)   AS unique_receivers
		 FROM transfers
		 GROUP BY token_address`).All(&rows)
	if err != nil {
		log.Printf("[token-analytics] unique recompute failed: %v", err)
		return
	}
	for _, u := range rows {
		r, ok := byAddr[u.TokenAddress]
		if !ok {
			continue
		}
		r.Set("unique_senders", u.UniqueSenders)
		r.Set("unique_receivers", u.UniqueReceivers)
	}
}

func enrichTokenMetadata(byAddr map[string]*core.Record, addrs []string) {
	const workers = 6
	workCh := make(chan string, len(addrs))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rpcCalls := 0
			for addr := range workCh {
				r := byAddr[addr]
				hasMetadata := r.GetString("symbol") != "" &&
					r.GetInt("decimals") > 0 &&
					!r.GetBool("lookup_failed")
				if hasMetadata {
					continue
				}
				info := arc.FetchFullTokenInfo(common.HexToAddress(addr))
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
		}()
	}
	for _, a := range addrs {
		workCh <- a
	}
	close(workCh)
	wg.Wait()
}

func persistTokenRecords(app core.App, records []*core.Record) (saved, failed int) {
	err := app.RunInTransaction(func(txApp core.App) error {
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
	}
	return saved, failed
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
		"Incrementally aggregates per-token stats from transfers and enriches with on-chain metadata",
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
