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

	v1.GET("/stats", timed("stats", statsHandler))
	v1.GET("/block_stats", timed("block_stats", blockStatsHandler))

	v1.GET("/blocks", timed("blocks", blocksHandler))
	v1.GET("/transactions", timed("transactions", transactionsHandler))
	v1.GET("/traces", timed("traces", tracesHandler))

	v1.GET("/transfers", timed("transfers", transfersHandler))

	v1.GET("/tokens", timed("tokens", tokensHandler))
	v1.GET("/tokens/{address}", timed("token_detail", tokenDetailHandler))

	v1.GET("/wallet/{address}", timed("wallet", walletHandler))

	v1.GET("/crosschain", timed("crosschain", crosschainHandler))
	v1.GET("/fx", timed("fx", fxHandler))

	v1.GET("/agents", timed("agents", agentsHandler))
	v1.GET("/agents/{address}", timed("agent", agentHandler))
	v1.GET("/jobs", timed("jobs", agentJobsHandler))

	v1.GET("/edges", timed("edges", edgesHandler))

	v1.GET("/health", timed("health", healthHandler))

	v1.GET("/search", timed("search", searchHandler))

	v1.GET("/tx/{hash}", timed("tx_detail", txDetailHandler))
	v1.GET("/block/{number}", timed("block_detail", blockDetailHandler))

	v1.GET("/analytics/overview", timed("analytics_overview", analyticsOverviewHandler))
	v1.GET("/analytics/fees", timed("analytics_fees", analyticsFeesHandler))
	v1.GET("/analytics/volume", timed("analytics_volume", analyticsVolumeHandler))
	v1.GET("/analytics/bridge_flow", timed("analytics_bridge_flow", analyticsBridgeFlowHandler))
	v1.GET("/analytics/agent_leaderboard", timed("analytics_agent_leaderboard", analyticsAgentLeaderboardHandler))
	v1.GET("/analytics/history", timed("analytics_history", analyticsHistoryHandler))
}
