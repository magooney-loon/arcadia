package jobs

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterJobs(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := indexerHealthJob(app); err != nil {
			app.Logger().Error("Failed to register indexer health job", "error", err)
			return err
		}
		if err := indexerEventsCleanupJob(app); err != nil {
			app.Logger().Error("Failed to register indexer events cleanup job", "error", err)
			return err
		}
		if err := analyticsSnapshotJob(app); err != nil {
			app.Logger().Error("Failed to register analytics snapshot job", "error", err)
			return err
		}
		if err := RegisterTokenAnalyticsJob(app); err != nil {
			app.Logger().Error("Failed to register token analytics job", "error", err)
			return err
		}
		app.Logger().Info("All jobs registered")

		go func() {
			time.Sleep(5 * time.Second)
			for _, win := range []string{"1h", "24h", "7d"} {
				if err := takeAnalyticsSnapshot(app, win); err != nil {
					app.Logger().Warn("Initial analytics snapshot failed", "window", win, "error", err)
				}
			}
		}()

		go func() {
			time.Sleep(30 * time.Second)
			RunTokenAnalytics(app)
		}()

		return e.Next()
	})
}
