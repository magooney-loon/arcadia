package jobs

import (
	"fmt"
	"time"

	"github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
)

// snapshotRetention is how long analytics_snapshots rows are kept. At the
// current cadence (1h every 5 min + 24h every 10 min + 7d every 30 min) the
// table accrues ~17 rows/hour, so 30 days ≈ 12K rows — small enough to query
// quickly while still giving enough history for charts.
const snapshotRetention = 30 * 24 * time.Hour

func snapshotCleanupJob(app core.App) error {
	jm := jobs.GetManager()
	if jm == nil {
		return fmt.Errorf("job manager not initialized")
	}

	return jm.RegisterJob(
		"analyticsSnapshotsCleanup",
		"Analytics Snapshots Cleanup",
		"Deletes analytics_snapshots rows older than 30 days",
		"30 3 * * *",
		func(el *jobs.ExecutionLogger) {
			el.Start("Analytics Snapshots Cleanup")

			cutoff := time.Now().Add(-snapshotRetention).Unix()
			deleted, err := repo.DeleteSnapshotsBefore(app, cutoff)
			if err != nil {
				el.Info("Failed to delete old snapshots: %s", err)
				el.Complete("Cleanup failed")
				return
			}

			el.Statistics(map[string]interface{}{"deleted": deleted})
			el.Complete(fmt.Sprintf("Deleted %d snapshots older than %d days", deleted, int(snapshotRetention.Hours()/24)))
		},
	)
}
