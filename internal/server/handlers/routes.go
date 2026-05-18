package handlers

// API_SOURCE

import (
	"github.com/magooney-loon/pb-ext/core/server/api"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterRoutes(app core.App) {
	versionManager := InitVersionedSystem()
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := versionManager.RegisterAllVersionRoutes(e); err != nil {
			return err
		}
		return e.Next()
	})
	versionManager.RegisterWithServer(app)
}

func InitVersionedSystem() *api.APIVersionManager {
	baseConfig := &api.APIDocsConfig{
		Title:       "Arcadia API",
		Description: "Arc L1 blockchain indexer — blocks, transfers, agents, cross-chain flows",
		BaseURL:     "http://127.0.0.1:8090/",
		Enabled:     true,

		ContactName:  "Arcadia",
		ContactEmail: "contact@magooney.org",
		ContactURL:   "https://github.com/magooney-loon/arcadia",

		LicenseName: "MIT",
		LicenseURL:  "https://opensource.org/licenses/MIT",
	}

	v1Config := *baseConfig
	v1Config.Version = "1.0.0"
	v1Config.Status = "stable"
	v1Config.PublicSwagger = true

	return api.InitializeVersionedSystemWithRoutes(map[string]*api.VersionSetup{
		"v1": {
			Config: &v1Config,
			Routes: registerV1Routes,
		},
	}, "v1")
}

func registerV1Routes(router *api.VersionedAPIRouter) {
	v1 := router.SetPrefix("/api/v1")

	v1.GET("/stats", statsHandler)
	v1.GET("/block_stats", blockStatsHandler)

	v1.GET("/blocks", blocksHandler)
	v1.GET("/transactions", transactionsHandler)
	v1.GET("/traces", tracesHandler)

	v1.GET("/transfers", transfersHandler)

	v1.GET("/tokens", tokensHandler)
	v1.GET("/tokens/{address}", tokenDetailHandler)

	v1.GET("/wallet/{address}", walletHandler)

	v1.GET("/crosschain", crosschainHandler)
	v1.GET("/fx", fxHandler)

	v1.GET("/agents", agentsHandler)
	v1.GET("/agents/{address}", agentHandler)
	v1.GET("/jobs", agentJobsHandler)

	v1.GET("/edges", edgesHandler)

	v1.GET("/health", healthHandler)

	v1.GET("/search", searchHandler)

	v1.GET("/tx/{hash}", txDetailHandler)
	v1.GET("/block/{number}", blockDetailHandler)

	v1.GET("/analytics/overview", analyticsOverviewHandler)
	v1.GET("/analytics/fees", analyticsFeesHandler)
	v1.GET("/analytics/volume", analyticsVolumeHandler)
	v1.GET("/analytics/bridge_flow", analyticsBridgeFlowHandler)
	v1.GET("/analytics/agent_leaderboard", analyticsAgentLeaderboardHandler)
	v1.GET("/analytics/history", analyticsHistoryHandler)
}
