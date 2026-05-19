# Arcadia Frontend

SvelteKit dashboard for the Arcadia blockchain indexer. Displays live Arc L1 data — blocks, transactions, stablecoin transfers, cross-chain flows, FX swaps, AI agent activity, and a 3D wallet graph — powered by REST and SSE from the Go backend.

## Stack

| | |
|---|---|
| **Framework** | SvelteKit 2 + Svelte 5 (runes mode) |
| **Language** | TypeScript (strict) |
| **Charting** | uPlot (time-series) |
| **Graph** | d3-force (wallet relationship canvas) |
| **Backend client** | PocketBase JS SDK 0.26 (REST + SSE) |
| **Adapter** | Static (`@sveltejs/adapter-static`) with SPA fallback (`200.html`) |
| **Styling** | Plain CSS with CSS custom properties — no Tailwind, no UI framework |
| **Fonts** | Inter + JetBrains Mono (Google Fonts CDN) |

## Develop

```bash
npm install
npm run dev        # starts Vite dev server on :5173, expects PocketBase at :8090
npm run build      # production build → build/
npm run preview    # preview the production build locally
npm run check      # svelte-check type validation
npm run lint       # ESLint
npm run format     # Prettier
```

From the repo root you can also use `pb-cli` which builds the frontend and copies it to `pb_public/` before starting the Go server — this is the normal dev workflow.

## Directory structure

```
src/
  app.html                 HTML shell
  lib/
    api/                   One module per data domain — each wraps apiFetch()
      utils.ts             apiFetch(): 5s timeout, abort-on-navigate, error normalization
      stats.ts             /api/v1/stats
      chain.ts             /api/v1/blocks, /txs, /traces, /search
      analytics.ts         /api/v1/analytics/*
      agents.ts            /api/v1/agents, /jobs
      wallet.ts            /api/v1/wallet/:address
      tokens.ts            /api/v1/tokens
      crosschain.ts        /api/v1/crosschain, /fx
      graph.ts             /api/v1/graph/edges
      health.ts            /api/v1/health
      ...
    stores/                Svelte 5 $state stores — one per domain, same shape throughout
      config.svelte.ts     PocketBase singleton + API URL detection (dev vs prod)
      stats.svelte.ts      Live chain metrics (block, tps, block_time_ms, lag)
      chain.svelte.ts      Blocks, txs, traces + live feeds seeded by SSE
      analytics.svelte.ts  Overview, history, fees, volume, bridge flow, leaderboard
      wallet.svelte.ts     Wallet detail (transfers, edges)
      ...
    components/            Reusable UI
      Chart.svelte         uPlot time-series with hover tooltip
      ForceGraph.svelte    d3-force canvas wallet graph
      DataState.svelte     Unified loading / error / empty renderer
      SyncOverlay.svelte   Catch-up progress bar (shown when indexer is behind)
      Pagination.svelte    Offset-based pagination
      AddrLink.svelte      Truncated address + arcscan link
      TxLink.svelte        Truncated tx hash + arcscan link
      ...
    realtime.ts            SSE connection manager (connectRealtime / disconnectRealtime)
    fmt.ts                 Formatting helpers (usdc, num, pct, tsAge, domainName, …)
    sort.svelte.ts         Reactive bidirectional sort state
  routes/
    +layout.svelte         App shell: sidebar, top bar (live metrics), SSE init
    +layout.ts             Global load
    overview/              Live dashboard
    blocks/                Block list + [number]/ detail
    txs/                   Transaction list + [hash]/ detail
    transfers/             Token transfer history
    traces/                Internal call traces
    tokens/                Token analytics + [address]/ detail
    crosschain/            CCTP + Gateway events + [chain]/
    fx/                    StableFX swap analytics
    agents/                Agent registry
    jobs/                  Job market (ERC-8183)
    graph/                 Wallet relationship force graph
    wallet/[address]/      Wallet / agent detail
    readme/                Protocol docs
    debug/                 Indexer internals debug view
```

## Data flow

The dashboard uses two transports in parallel:

### REST — initial load + user actions

All fetch calls go through `apiFetch()` in `src/lib/api/utils.ts`:
- 5-second timeout (prevents connection pool starvation during SvelteKit lazy chunk loads)
- All in-flight requests are tracked and aborted on navigation (`abortAll()`)
- Errors are normalized to strings; cancelled requests are silently ignored

Route `+page.ts` files trigger fetches non-blocking and return `{}` — the page renders with a loading state immediately and populates as data arrives.

### SSE — live updates

`realtime.ts` manages a single persistent PocketBase subscription. `connectRealtime()` is called once in `+layout.svelte`. The PocketBase SDK handles the SSE lifecycle: auto-reconnect, clientId handshake, re-subscribe on reconnect.

Three topics are active for every connected client:

| Topic | Payload | When |
|---|---|---|
| `indexer` | `{stats, health, blocks[], transactions[]}` | After each batch commit (~1 Hz, throttled) |
| `analytics` | `{window, overview, bridge_flow, volume}` | After each 5-min snapshot job |
| `charts` | `{block_stats[]}` (50 rows) | After each batch commit, overview page only |

SSE handlers mutate store state directly (`stats.data = payload.stats`). The charts topic is connected and disconnected by the overview page itself to avoid redundant data for other routes.

## Store pattern

Every store in `src/lib/stores/` follows the same shape:

```ts
export const thing = $state<{ data: Thing | null; loading: boolean; error: string | null }>({
  data: null, loading: false, error: null,
});

export async function fetchThing(params: Params) {
  thing.loading = true;
  thing.error = null;
  try {
    thing.data = await thingClient.get(params);
  } catch (e) {
    if (!String(e).includes('cancelled')) thing.error = String(e);
  } finally {
    thing.loading = false;
  }
}
```

Cancelled requests (from navigation abort) are silently swallowed. Everything else surfaces as `thing.error`.

## Routing

Static routes are prerendered (fast HTML shell). Dynamic routes (`/wallet/[address]/`, `/blocks/[number]/`, etc.) set `ssr = false` and `prerender = false` — they render client-side from the SPA fallback (`200.html`).

The static adapter outputs to `build/`. `pb-cli` copies the build output into the Go server's `pb_public/` directory so the backend serves the dashboard as a static site.

## Adding a new page

1. Create `src/routes/mypage/+page.svelte` and optionally `+page.ts`
2. Add a fetch function to an existing API module or create `src/lib/api/mypage.ts`
3. Add a store in `src/lib/stores/mypage.svelte.ts` following the store pattern above
4. Add the nav link in `+layout.svelte`

## Customization

**Backend URL** — `src/lib/stores/config.svelte.ts`, function `getApiUrl()`:
- Dev: hardcoded `http://127.0.0.1:8090`
- Prod: `window.location.origin` (assumes same-origin reverse proxy)

**Chain/domain names** — `src/lib/fmt.ts`, constant `DOMAIN_NAMES` (maps CCTP domain IDs to chain names).

**SSE topics** — `src/lib/realtime.ts`. If the Go backend publishes different topic names, update the `pb.realtime.subscribe()` calls here.

**New API endpoints** — add a function to the appropriate `src/lib/api/*.ts` module using `apiFetch()`.
