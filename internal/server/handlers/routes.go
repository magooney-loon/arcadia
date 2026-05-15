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

	// live dashboard stats (latest block_stats row)
	v1.GET("/stats", statsHandler)
	// historical block stats for time-series charts
	v1.GET("/block_stats", blockStatsHandler)

	// chain data
	v1.GET("/blocks", blocksHandler)
	v1.GET("/transactions", transactionsHandler)
	v1.GET("/traces", tracesHandler)

	// token flows  — ?token=USDC|EURC|USYC, ?from=addr, ?to=addr
	v1.GET("/transfers", transfersHandler)

	// tokens
	v1.GET("/tokens", tokensHandler)
	v1.GET("/tokens/{address}", tokenDetailHandler)

	// wallet profile + edges
	v1.GET("/wallet/{address}", walletHandler)

	// cross-chain + FX
	v1.GET("/crosschain", crosschainHandler)
	v1.GET("/fx", fxHandler)

	// agent economy
	v1.GET("/agents", agentsHandler)
	v1.GET("/agents/{address}", agentHandler)
	v1.GET("/jobs", agentJobsHandler)

	// wallet graph edges (for 3D renderer)
	v1.GET("/edges", edgesHandler)

	// indexer health
	v1.GET("/health", healthHandler)

	// unified search
	v1.GET("/search", searchHandler)

	// single-record detail pages
	v1.GET("/tx/{hash}", txDetailHandler)
	v1.GET("/block/{number}", blockDetailHandler)

	// analytics (snapshot-backed, window-scoped)
	v1.GET("/analytics/overview", analyticsOverviewHandler)
	v1.GET("/analytics/fees", analyticsFeesHandler)
	v1.GET("/analytics/volume", analyticsVolumeHandler)
	v1.GET("/analytics/bridge_flow", analyticsBridgeFlowHandler)
	v1.GET("/analytics/agent_leaderboard", analyticsAgentLeaderboardHandler)
	v1.GET("/analytics/history", analyticsHistoryHandler)
}
