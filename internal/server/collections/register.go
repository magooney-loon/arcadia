package collections

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterCollections(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		for _, fn := range []func(core.App) error{
			metaCollection,
			indexerEventsCollection,
			tokenAnalyticsCollection,
			blocksCollection,
			transactionsCollection,
			transfersCollection,
			tracesCollection,
			crosschainEventsCollection,
			fxSwapsCollection,
			agentsCollection,
			agentJobsCollection,
			blockStatsCollection,
			walletEdgesCollection,
			analyticsSnapshotsCollection,
		} {
			if err := fn(e.App); err != nil {
				app.Logger().Error("Collection setup error", "error", err)
			}
		}
		return e.Next()
	})
}

func collectionExists(app core.App, name string) bool {
	c, _ := app.FindCollectionByNameOrId(name)
	return c != nil
}
