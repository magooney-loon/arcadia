package main

import (
	"flag"
	"log"

	"github.com/joho/godotenv"
	app "github.com/magooney-loon/pb-ext/core"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/indexer"
	"arcadia/internal/jobs"
	"arcadia/internal/server"
)

func main() {
	// Load .env if present — silently ignored if the file doesn't exist
	// so production envs that inject vars directly are unaffected.
	_ = godotenv.Load()

	devMode := flag.Bool("dev", false, "Run in developer mode")
	generateSpecsDir := flag.String("generate-specs-dir", "", "Generate OpenAPI specs into the provided directory and exit")
	generateSpecVersion := flag.String("generate-spec-version", "", "Optional API version to generate (requires --generate-specs-dir)")
	validateSpecsDir := flag.String("validate-specs-dir", "", "Validate OpenAPI specs from the provided directory and exit")
	flag.Parse()

	if *generateSpecsDir != "" {
		gen := app.NewSpecGeneratorWithInitializer(func() (*app.APIVersionManager, error) {
			return server.InitVersionedSystem(), nil
		})
		if err := gen.Generate(*generateSpecsDir, *generateSpecVersion); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *validateSpecsDir != "" {
		gen := app.NewSpecGeneratorWithInitializer(func() (*app.APIVersionManager, error) {
			return server.InitVersionedSystem(), nil
		})
		if err := gen.Validate(*validateSpecsDir); err != nil {
			log.Fatal(err)
		}
		return
	}

	initApp(*devMode)
}

func initApp(devMode bool) {
	var opts []app.Option

	if devMode {
		opts = append(opts, app.InDeveloperMode())
	} else {
		opts = append(opts, app.InNormalMode())
	}

	// Option 1: Use a custom PocketBase config
	// pbConfig := &pocketbase.Config{
	// 	DefaultDev:     true,
	// 	DefaultDataDir: "./custom_pb_data",
	// }
	// opts = append(opts, app.WithConfig(pbConfig))

	// Option 2: Use an existing PocketBase instance
	// pb := pocketbase.New()
	// opts = append(opts, app.WithPocketbase(pb))

	// Set custom port programmatically
	// os.Args = []string{"app", "serve", "--http=127.0.0.1:9090"}

	// Note: WithConfig and WithPocketbase cannot be used together

	srv := app.New(opts...)

	app.SetupLogging(srv)

	server.RegisterCollections(srv.App())
	server.RegisterRoutes(srv.App())
	jobs.RegisterJobs(srv.App())

	// Apply SQLite PRAGMA tuning before the server starts serving requests.
	// WAL + NORMAL synchronous is crash-safe and gives better write throughput.
	// busy_timeout prevents SQLITE_BUSY errors under concurrent writer pressure.
	// cache_size and temp_store improve sort/query performance at the cost of RAM.
	srv.App().OnServe().BindFunc(func(e *core.ServeEvent) error {
		db := e.App.DB()
		for _, pragma := range []string{
			"PRAGMA synchronous=NORMAL",
			"PRAGMA busy_timeout=5000",
			"PRAGMA cache_size=-8000",
			"PRAGMA temp_store=2",
			"PRAGMA mmap_size=268435456",
		} {
			if _, err := db.NewQuery(pragma).Execute(); err != nil {
				e.App.Logger().Warn("SQLite PRAGMA failed", "pragma", pragma, "error", err)
			}
		}
		return e.Next()
	})

	srv.App().OnServe().BindFunc(func(e *core.ServeEvent) error {
		app.SetupRecovery(srv.App(), e)
		indexer.StartIndexer(srv.App())
		jobs.StartTokenAnalyticsScheduler(srv.App())
		return e.Next()
	})

	if err := srv.Start(); err != nil {
		srv.App().Logger().Error("Fatal application error",
			"error", err,
			"uptime", srv.Stats().StartTime,
			"total_requests", srv.Stats().TotalRequests.Load(),
			"active_connections", srv.Stats().ActiveConnections.Load(),
			"last_request_time", srv.Stats().LastRequestTime.Load(),
		)
		log.Fatal(err)
	}
}

// Build toolchain (pb-cli):
// go install github.com/magooney-loon/pb-ext/cmd/pb-cli@latest
//
// Ready for a production build deployment?
// https://github.com/magooney-loon/pb-deployer
