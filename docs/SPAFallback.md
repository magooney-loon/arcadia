# SPA Fallback for PocketBase

PocketBase serves static files from `pb_public/` via `apis.Static()`, but it has
no built-in SPA catch-all. When a user navigates directly to a client-side route
like `/wallet/0xabc.../`, the server looks for a physical file at that path,
doesn't find one, and returns a 404 — the JS bundle never loads.

This doc covers the two halves of the fix:

1. **SvelteKit** — generate a dedicated fallback HTML shell
2. **Go server** — intercept 404s and serve that shell instead

---

## SvelteKit side

In `svelte.config.js`, use `adapter-static` with a `fallback` filename:

```js
import adapter from '@sveltejs/adapter-static';

const config = {
  kit: {
    adapter: adapter({ fallback: '200.html' })
  }
};
```

`adapter-static` produces a single `200.html` in the build output containing the
full SPA shell (JS bundles, CSS, `<div id="app">`, etc.). The filename `200.html`
is intentionally separate from `index.html` so it doesn't overwrite the root page.

**Why `200.html` and not `index.html`?**

Some hosting platforms (Netlify, Vercel) recognise `200.html` as an implicit SPA
fallback. For PocketBase the name doesn't matter — the Go middleware picks the
file up explicitly — but `200.html` is the conventional name and avoids
confusion with a prerendered root `index.html`.

---

## Go server side

PocketBase's `apis.Static(fsys, false)` returns a `*router.ApiError` with
`Status: 404` when a file isn't found. It does **not** write an HTTP 404
response — it returns a Go error. A middleware that only checks
`c.Response.Status` or `c.Written()` will never catch it.

The fix is router-level middleware registered in `OnServe` that:

1. Calls `c.Next()` and inspects the returned error
2. Uses `errors.As()` to check if it's a `*router.ApiError` with status 404
3. Skips API routes (`/api/*`), admin routes (`/_*`), and static assets
4. Serves the `200.html` content inline with a 200 status

### Minimal implementation

```go
package main

import (
    "errors"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/pocketbase/tools/router"
)

type spaFallback struct {
    html    string
    modTime time.Time
}

func newSpaFallback() *spaFallback {
    candidates := []string{
        "./pb_public/200.html",
    }
    // Also try relative to the binary (production deployments).
    if exe, err := os.Executable(); err == nil {
        dir := filepath.Dir(exe)
        candidates = append(candidates,
            filepath.Join(dir, "pb_public", "200.html"),
            filepath.Join(dir, "../pb_public", "200.html"),
        )
    }
    for _, p := range candidates {
        data, err := os.ReadFile(p)
        if err != nil {
            continue
        }
        info, _ := os.Stat(p)
        return &spaFallback{html: string(data), modTime: info.ModTime()}
    }
    return &spaFallback{} // no fallback available
}

func isAssetPath(path string) bool {
    if strings.HasPrefix(path, "/_app/") {
        return true
    }
    switch strings.ToLower(filepath.Ext(path)) {
    case ".js", ".mjs", ".css", ".map", ".json",
        ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico",
        ".woff", ".woff2", ".ttf", ".eot", ".otf",
        ".wasm", ".webmanifest":
        return true
    }
    return false
}

func registerSpaFallback(e *core.ServeEvent) {
    spa := newSpaFallback()

    e.Router.BindFunc(func(c *core.RequestEvent) error {
        err := c.Next()

        // Only intercept on GET requests that resulted in an error.
        if err == nil || c.Request.Method != http.MethodGet {
            return err
        }

        // Must be a 404 ApiError from the static file handler.
        var apiErr *router.ApiError
        if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
            return err
        }

        // Don't shadow API routes, admin UI, or static assets.
        p := c.Request.URL.Path
        if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/_") || isAssetPath(p) {
            return err
        }

        // Serve the SPA shell.
        if spa.html != "" {
            c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
            c.Response.WriteHeader(http.StatusOK)
            http.ServeContent(c.Response, c.Request, "index.html", spa.modTime, strings.NewReader(spa.html))
            return nil
        }

        return err
    })
}
```

### Registration

Hook it into your `OnServe` handler:

```go
srv.App().OnServe().BindFunc(func(e *core.ServeEvent) error {
    registerSpaFallback(e)
    // ...other setup...
    return e.Next()
})
```

**Ordering note:** `e.Router.BindFunc` adds middleware to the router itself.
It runs for every request, wrapping the route handlers. Since it calls
`c.Next()` first and only acts on the result, it doesn't interfere with
API or static file routes that resolve successfully.

---

## How it works end-to-end

```
Browser → GET /wallet/0xabc.../
  → pb-ext static handler: no file at pb_public/wallet/0xabc.../
  → returns *router.ApiError{Status: 404}
  → SPA middleware: 404 on GET, not API, not asset → serve 200.html
  → browser receives SPA shell (200 OK, text/html)
  → SvelteKit JS boots, client-side router matches /wallet/[address]
  → page renders
```

```
Browser → GET /api/v1/stats
  → API route handler: matches, returns JSON
  → SPA middleware: err == nil → pass through
  → normal API response
```

```
Browser → GET /_app/immutable/assets/app-XYZ.css
  → pb-ext static handler: file exists in pb_public
  → returns CSS file (200 OK)
  → SPA middleware: err == nil → pass through
  → normal static response
```

---

## What to add to pb-ext

The `newSpaFallback()` / `isAssetPath()` / middleware registration should be
folded into pb-ext's `core/server/server.go`, right next to the existing
`e.Router.GET("/{path...}", apis.Static(...))` call. Possible API:

```go
// Opt-in via server option:
srv := app.New(app.WithSpaFallback(true))
```

Or detect `200.html` in `pb_public` automatically and enable the fallback
without any configuration — similar to how Netlify works.
