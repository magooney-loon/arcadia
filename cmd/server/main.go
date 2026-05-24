package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	app "github.com/magooney-loon/pb-ext/core"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	_ "modernc.org/sqlite"

	"arcadia/internal/indexer"
	"arcadia/internal/jobs"
	"arcadia/internal/server"
)

// arcadiaPragmas are layered onto PocketBase's DSN-applied defaults via a
// custom DBConnect so they take effect on every connection in both the
// concurrent (reader) and nonconcurrent (writer) pools.
//
// Why each pragma:
//   - busy_timeout(10000):       PocketBase default; restated because it must
//     come before journal_mode(WAL).
//   - journal_mode(WAL):         required for concurrent readers + writer.
//   - synchronous(NORMAL):       safe under WAL; faster than FULL.
//   - foreign_keys(ON):          parity with PB default.
//   - temp_store(MEMORY):        keep sort/group spills out of disk.
//   - cache_size(-32000):        32 MB page cache per connection (PB default).
//   - journal_size_limit(...):   cap WAL+journal disk usage at 200 MB.
//   - wal_autocheckpoint(256):   trigger passive checkpoints every 256 pages
//     (~1 MB) instead of the 1000-page default so
//     WAL stays small under sustained writer load.
//     The idle TRUNCATE checkpoint in the indexer
//     handles the case where readers block passive
//     checkpoints from making progress.
//   - mmap_size(268435456):      256 MB memory-mapped read window; cuts
//     read-side syscalls during sprint indexing.
const arcadiaPragmas = "?_pragma=busy_timeout(10000)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(ON)" +
	"&_pragma=temp_store(MEMORY)" +
	"&_pragma=cache_size(-32000)" +
	"&_pragma=journal_size_limit(200000000)" +
	"&_pragma=wal_autocheckpoint(256)" +
	"&_pragma=mmap_size(268435456)"

// spaFallback holds the preloaded 200.html content for SPA routing.
type spaFallback struct {
	indexHTML string
	modTime   time.Time
}

func newSpaFallback() *spaFallback {
	path := "./pb_public/200.html"
	data, err := os.ReadFile(path)
	if err != nil {
		exePath, exeErr := os.Executable()
		if exeErr == nil {
			exeDir := filepath.Dir(exePath)
			for _, p := range []string{
				filepath.Join(exeDir, "pb_public", "200.html"),
				filepath.Join(exeDir, "../pb_public", "200.html"),
			} {
				if data, err = os.ReadFile(p); err == nil {
					break
				}
			}
		}
	}
	if err != nil {
		return &spaFallback{}
	}
	info, _ := os.Stat(path)
	var modTime time.Time
	if info != nil {
		modTime = info.ModTime()
	}
	return &spaFallback{indexHTML: string(data), modTime: modTime}
}

// isAssetPath returns true for paths that look like static assets
// so we never serve the SPA shell for those.
func isAssetPath(path string) bool {
	if strings.HasPrefix(path, "/_app/") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".mjs", ".css", ".map", ".json",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico",
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		".wasm", ".webmanifest":
		return true
	}
	return false
}

func arcadiaDBConnect(dbPath string) (*dbx.DB, error) {
	return dbx.Open("sqlite", dbPath+arcadiaPragmas)
}

func main() {
	if err := godotenv.Load(); err != nil {
		// No .env file — only fatal if required vars are also missing from
		// the process environment (e.g. Docker/systemd injects them).
		if os.Getenv("ENVIO_API_TOKEN") == "" {
			log.Fatal("missing .env file and ENVIO_API_TOKEN not set — copy .env.example and configure it")
		}
	}

	if os.Getenv("ENVIO_API_TOKEN") == "" {
		log.Fatal("ENVIO_API_TOKEN is required — get one at https://envio.dev")
	}

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

	// Inject tuned SQLite PRAGMAs via a custom DBConnect. This is the
	// only hook that runs *before* the connection pools open, so the
	// values apply to every connection in both the concurrent (reader)
	// and nonconcurrent (writer) pools. Earlier attempts via
	// app.DB().NewQuery("PRAGMA …") only hit the writer connection.
	opts = append(opts, app.WithConfig(&pocketbase.Config{
		DBConnect: arcadiaDBConnect,
	}))

	srv := app.New(opts...)

	app.SetupLogging(srv)

	server.RegisterCollections(srv.App())
	server.RegisterRoutes(srv.App())
	jobs.RegisterJobs(srv.App())

	srv.App().OnServe().BindFunc(func(e *core.ServeEvent) error {
		app.SetupRecovery(srv.App(), e)
		indexer.StartIndexer(srv.App())

		// SPA fallback: when the static file handler returns a 404 ApiError
		// for a non-API GET request, serve pb_public/200.html instead so
		// SvelteKit's client-side router handles dynamic routes.
		spa := newSpaFallback()
		e.Router.BindFunc(func(c *core.RequestEvent) error {
			err := c.Next()
			// Only intercept 404 ApiErrors on GET requests.
			if err == nil || c.Request.Method != http.MethodGet {
				return err
			}
			var apiErr *router.ApiError
			if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
				return err
			}
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/_") || isAssetPath(path) {
				return err
			}
			if spa.indexHTML != "" {
				c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
				c.Response.WriteHeader(http.StatusOK)
				http.ServeContent(c.Response, c.Request, "index.html", spa.modTime, strings.NewReader(spa.indexHTML))
				return nil
			}
			return err
		})

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
// go run ./cmd/pb-cli
//
// What is pb-ext? Learn more:
// https://github.com/magooney-loon/pb-ext
