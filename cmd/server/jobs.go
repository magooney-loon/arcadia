package main

import (
	"fmt"
	"time"

	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/pocketbase/core"
)

func registerJobs(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := indexerHealthJob(app); err != nil {
			app.Logger().Error("Failed to register indexer health job", "error", err)
			return err
		}
		app.Logger().Info("All jobs registered")
		return e.Next()
	})
}

// indexerHealthJob runs every minute and logs indexer cursor + collection row counts.
func indexerHealthJob(app core.App) error {
	jm := jobs.GetManager()
	if jm == nil {
		return fmt.Errorf("job manager not initialized")
	}

	return jm.RegisterJob(
		"indexerHealth",
		"Indexer Health Check",
		"Logs the current indexer cursor and row counts for all arcadia collections every minute",
		"* * * * *",
		func(el *jobs.ExecutionLogger) {
			el.Start("Indexer Health Check")
			el.Info("Checked at: %s", time.Now().Format(time.RFC3339))

			collections := []string{
				"blocks", "transactions", "transfers", "traces",
				"crosschain_events", "fx_swaps", "agents", "agent_jobs",
				"block_stats", "wallet_edges",
			}

			counts := make(map[string]interface{})
			for _, name := range collections {
				records, err := app.FindRecordsByFilter(name, "", "", 0, 0)
				if err != nil {
					counts[name] = "error"
				} else {
					counts[name] = len(records)
				}
			}

			cursor, _ := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
			lastBlock := "unknown"
			if len(cursor) > 0 {
				lastBlock = cursor[0].GetString("value")
			}

			el.Info("Last indexed block: %s", lastBlock)
			el.Statistics(counts)
			el.Complete(fmt.Sprintf("Health check done — last block: %s", lastBlock))
		},
	)
}
