package jobs

import (
	"fmt"
	"time"

	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/pocketbase/core"
)

func indexerEventsCleanupJob(app core.App) error {
	jm := jobs.GetManager()
	if jm == nil {
		return fmt.Errorf("job manager not initialized")
	}

	return jm.RegisterJob(
		"indexerEventsCleanup",
		"Indexer Events Cleanup",
		"Deletes indexer_events records older than 2 hours",
		"0 * * * *",
		func(el *jobs.ExecutionLogger) {
			el.Start("Indexer Events Cleanup")

			cutoff := time.Now().Add(-2 * time.Hour).Unix()
			records, err := app.FindRecordsByFilter(
				"indexer_events",
				fmt.Sprintf("timestamp < %d", cutoff),
				"", 0, 0,
			)
			if err != nil {
				el.Info("Failed to fetch old events: %s", err)
				el.Complete("Cleanup failed")
				return
			}

			deleted := 0
			for _, r := range records {
				if err := app.Delete(r); err == nil {
					deleted++
				}
			}

			el.Statistics(map[string]interface{}{"deleted": deleted})
			el.Complete(fmt.Sprintf("Deleted %d events older than 2 hours", deleted))
		},
	)
}
