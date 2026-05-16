package jobs

import (
	"fmt"
	"time"

	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
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
			deleted, err := repo.DeleteEventsBefore(app, cutoff)
			if err != nil {
				el.Info("Failed to delete old events: %s", err)
				el.Complete("Cleanup failed")
				return
			}

			el.Statistics(map[string]interface{}{"deleted": deleted})
			el.Complete(fmt.Sprintf("Deleted %d events older than 2 hours", deleted))
		},
	)
}
