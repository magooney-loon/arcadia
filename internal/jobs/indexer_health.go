package jobs

import (
	"fmt"
	"time"

	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
	"arcadia/internal/utils"
)

func indexerHealthJob(app core.App) error {
	jm := jobs.GetManager()
	if jm == nil {
		return fmt.Errorf("job manager not initialized")
	}

	return jm.RegisterJob(
		"indexerHealth",
		"Indexer Health Check",
		"Logs the current indexer cursor and row counts for all arcadia collections",
		"0 * * * *",
		func(el *jobs.ExecutionLogger) {
			el.Start("Indexer Health Check")
			el.Info("Checked at: %s", time.Now().Format(time.RFC3339))

			collections := []string{
				"indexer_events", "blocks", "transactions", "transfers", "traces",
				"crosschain_events", "fx_swaps", "agents", "agent_jobs",
				"block_stats", "wallet_edges",
			}

			counts := make(map[string]interface{})
			for _, name := range collections {
				n, err := repo.RowCount(app, name)
				if err != nil {
					counts[name] = "error"
				} else {
					counts[name] = n
				}
			}

			lastBlock := "unknown"
			if last := utils.GetLastIndexedBlock(app); last > 0 {
				lastBlock = fmt.Sprintf("%d", last)
			}

			el.Info("Last indexed block: %s", lastBlock)
			el.Statistics(counts)
			el.Complete(fmt.Sprintf("Health check done — last block: %s", lastBlock))
		},
	)
}
