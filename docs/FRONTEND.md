# Arcadia — City of Chains

## The City

Arcadia is a city you can fly through. Each chain block is a **city block** — a literal parcel of land in the urban grid. Inside every city block live the wallets that were active in it, connected by the transfers that moved between them during that block. The city extends in one direction through time: newest block at the front, history receding.

From the air you see the skyline — blocks raised to different heights by activity, glowing hot or cold by utilization. Fly down into any block and it opens up: buildings (wallet addresses) rise from the parcel floor, streets (transfer edges) connect them, agents tower above the rest in green.

The city is alive. New blocks land at the front every second. Couriers (particles) race between buildings. Cross-chain capital arrives at the city border as port traffic.

**No decorative structures. Every building, street, and courier is data.**

---

## Blueprint

| Layer | Choice | Reason |
|---|---|---|
| Framework | SvelteKit | SSR shell, file-based routing |
| 3D renderer | **Threlte** (`@threlte/core`, `@threlte/extras`) | Svelte-native Three.js; scene graph = component tree; Svelte 5 runes |
| Underlying 3D | Three.js (via Threlte) | Full access when needed |
| Block-level layout | `d3-force-3d` | Per-block mini force sim → positions wallets within a parcel's bounds |
| Real-time | PocketBase websockets | Subscribe to collection changes, push to stores |
| Dashboard charts | Raw SVG `$derived` | Lightweight time-series for the 2D overlay |
| Types / state | `$lib/api/` + `$lib/stores/` | Done — all endpoints typed and stored |

**Installed:**
```
@threlte/core  @threlte/extras  three  @types/three  d3-force-3d
```

---

## The Map (Routes)

```
/          → The City   (Scene.svelte as full viewport canvas)   ✅
/debug     → Control Room  (API explorer with filter controls)   ✅
```

---

## Urban Palette

| Element | Value |
|---|---|
| Night sky (background) | `#0a0a0f` |
| City block parcel (ground) | `#111118` — the floor of each block |
| Buildings (wallet nodes) | `#ffffff` — occupants of a block |
| Streets (transfer edges within block) | `#9aa0b4` at 30% opacity |
| Agent towers | `#7ee5a8` (green) — glow + pulse, taller by `usdc_spent_fees` |
| Resident paths (cross-block edges) | `#9aa0b4` at 10% opacity — faint lines connecting same wallet across blocks |
| Harbor traffic (cross-chain) | `#6be3ff` (cyan) |
| USDC couriers | `#2775ca` (Circle blue) |
| EURC couriers | `#e8b84b` (EUR gold) |
| USYC couriers | `#7b61ff` (purple, yield) |
| FX bridges | gradient USDC→EURC |
| Block heat (cold, low utilization) | hsl(210°, blue) |
| Block heat (hot, high utilization) | hsl(25°, orange) |
| Job arcs (work orders between agents) | `#f0a500` (amber) |

---

## City Data Lines (API & Stores)

| Endpoint | Store | What it feeds |
|---|---|---|
| `GET /api/v1/stats` | `stats` | City pulse: latest block_stats + rolling TPS avg |
| `GET /api/v1/block_stats` | `blockStats` | Block history for dashboard sparklines |
| `GET /api/v1/blocks` | `blocks` | City block parcels: `tx_count`, `utilization_pct`, `gas_used`, `timestamp` |
| `GET /api/v1/transactions?block=N` | `blockTxs` | All transactions inside one city block |
| `GET /api/v1/transfers?block=N` | `blockTransfers` | All transfers inside one block — **needs backend filter** (see below) |
| `GET /api/v1/transfers` | `transfers` | Global transfer feed for particle spawning |
| `GET /api/v1/traces` | `traces` | Internal call traces |
| `GET /api/v1/crosschain` | `crosschain` | Harbor arrivals: CCTP/Gateway events |
| `GET /api/v1/fx` | `fx` | Exchange floor: StableFX trades |
| `GET /api/v1/agents` | `agents` | Agent tower registry |
| `GET /api/v1/agents/{address}` | `agent` | Single tower profile + job history |
| `GET /api/v1/jobs` | `agentJobs` | Work orders: full job lifecycle |
| `GET /api/v1/edges` | `graph` | Resident paths: cumulative cross-block wallet connections |
| `GET /api/v1/wallet/{address}` | `wallet` | Wallet profile: history, edges, agent status |

**Needed backend addition:** `GET /api/v1/transfers?block=N` — `transfers` table has `block_number` field; the handler just needs a `block` query param filter added (same pattern as `transactionsHandler`).

---

## City Layout (Scene Architecture)

```
<Canvas>  — THE CITY  (#0a0a0f night sky)
│
├── <SceneCamera>          The Drone — two modes: CityView (high, wide) + BlockView (low, focused)
├── <SceneLighting>        City Lights — ambient glow + accent points
│
├── <CityGrid>             THE CITY GRID — sequence of chain blocks laid flat on the XZ plane
│   ├── <BlockParcel>      One per block — raised platform, heat-colored, sized by tx_count
│   ├── <BlockBuildings>   Shown when block is focused — wallet nodes inside the parcel bounds
│   ├── <BlockStreets>     Shown when block is focused — transfer edges between wallets in this block
│   └── <AgentTowers>      Agent wallets as taller green structures, visible in both modes
│
├── <ResidentPaths>        Cross-block wallet connections from wallet_edges — faint overlay on the grid
│
├── <CrosschainArrows>         Harbor — Bezier arcs arriving at the city border
│
├── <TokenParticles>         ✅  Couriers — racing between wallets along transfer routes, color by token
│
└── <FXSwapArcs>               Exchange bridges — bidirectional USDC↔EURC arcs
```

### Camera modes

**CityView (default):** Camera is high and angled, showing the full block strip. You orbit and pan over the grid. Block heights, heat, and scale make the skyline legible at a glance.

**BlockView (on click):** Camera glides down into the selected block. The parcel expands slightly, `BlockBuildings` and `BlockStreets` fade in. You see exactly who was in this block and what moved between them. Click away or press Escape → camera lifts back to CityView.

---

## City Operations Dashboard (2D HUD)

```
+page.svelte
│
├── <Scene>  (The City, z-index 0)
│
└── <HUD>    City Ops Dashboard (absolute overlay)
    ├── <StatsBar>          TPS · Block time · Indexed block · Avg fee
    ├── <TokenFlowPanel>    USDC / EURC / USYC volume this block
    ├── <AgentCounter>      Active agents · Open jobs
    ├── <LayerToggles>      Toggle: parcels / buildings / resident paths / couriers / harbor
    ├── <MiniCharts>        Sparklines: TPS + USDC volume (last 50 blocks)
    └── <BlockInspector>    Right panel — slides in when a block is focused
                            Shows: block number, timestamp, tx_count, utilization, fee total
                            + list of wallets + transfers that happened inside it
```

---

## City Logistics (Data Flow)

### City wakes up
```
onMount
  → Fetch last 50 blocks → lay out CityGrid parcels on XZ plane
  → Fetch wallet_edges (500) → draw ResidentPaths across the grid
  → Fetch agents → mark agent tower locations on parcels
  → Fetch transfers (200) → seed courier ring buffer
  → Scene is live
```

### City breathes (PocketBase websockets)
```
pb.collection('block_stats').subscribe('*', handler)
  → New block arrives → new parcel added at front of grid, oldest slides back

pb.collection('transfers').subscribe('*', handler)
  → New transfer → courier dispatched in ring buffer

pb.collection('crosschain_events').subscribe('*', handler)
  → Capital arrives at Harbor → arc animation fires

pb.collection('fx_swaps').subscribe('*', handler)
  → Trade executed → FX bridge arc fires

pb.collection('agent_jobs').subscribe('*', handler)
  → Job posted/filled → work order arc fires between agent towers

pb.collection('agents').subscribe('*', handler)
  → New agent registered → tower marked on its parcel
```

### Exploring the city (Interaction)
```
click city block (BlockParcel)
  → camera glides to BlockView
  → fetch transactions?block=N + transfers?block=N (parallel)
  → d3-force-3d mini sim positions wallets within parcel bounds
  → BlockBuildings + BlockStreets fade in
  → BlockInspector panel opens with block detail + wallet list

click wallet node (inside BlockView)
  → fetchWallet(address)
  → BlockInspector switches to wallet profile: history, cross-block edges, agent status

click away / Escape → camera lifts to CityView, parcel contents fade out

drag / scroll → Drone orbits in current mode
double-click parcel → same as click (focus)
```

---

## City Blueprints (Component File Structure)

```
src/
├── routes/
│   ├── +page.svelte              ✅ full-viewport shell: <Scene> + <HUD>
│   └── debug/
│       └── +page.svelte          ✅ Control Room — all endpoints, full filter controls
│
└── lib/
    ├── scene/
    │   ├── Scene.svelte           ✅ Threlte <Canvas>
    │   ├── SceneCamera.svelte     ✅ Drone — CityView + BlockView modes
    │   ├── SceneLighting.svelte   ✅ Ambient + accent lights
    │   │
    │   ├── city/                     ← THE CITY GRID
    │   │   ├── CityGrid.svelte       Lays out BlockParcels in a strip, manages focused block state
    │   │   ├── BlockParcel.svelte    One chain block: raised platform, heat color, tx_count height
    │   │   ├── BlockBuildings.svelte Wallets active in focused block — InstancedMesh, mini force layout
    │   │   ├── BlockStreets.svelte   Transfer edges within focused block — LineSegments
    │   │   └── AgentTowers.svelte    Agent wallet markers — green emissive, taller, always visible
    │   │
    │   ├── global/
    │   │   └── ResidentPaths.svelte  Cross-block wallet edges — faint LineSegments over the grid
    │   │
    │   ├── particles/
    │   │   ├── TokenParticles.svelte   ✅ Couriers — InstancedMesh, color by token, lerp along routes
    │   │   └── ParticleSpawner.svelte  ✅ Watches transfers + layout, dispatches couriers
    │   │
    │   └── fx/
    │       ├── CrosschainArrows.svelte   Harbor traffic — Bezier arcs from city border
    │       └── FXSwapArcs.svelte         Exchange bridges
    │
    ├── hud/
    │   ├── HUD.svelte
    │   ├── StatsBar.svelte
    │   ├── TokenFlowPanel.svelte
    │   ├── AgentCounter.svelte
    │   ├── LayerToggles.svelte
    │   ├── MiniCharts.svelte
    │   └── BlockInspector.svelte      (was WalletInspector — now shows block + wallet detail)
    │
    ├── scene-state/
    │   ├── city.svelte.ts         Block strip state: parcel positions, focused block, camera mode
    │   ├── layout.svelte.ts       ✅ d3-force-3d mini sim for wallets within a focused block
    │   ├── particles.svelte.ts    ✅ Courier ring buffer
    │   ├── layers.svelte.ts           Layer visibility toggles
    │   └── selection.svelte.ts        Currently focused block + selected wallet
    │
    ├── api/                       ✅ all endpoints typed — add block filter to transfers
    │   ├── stats/
    │   ├── block_stats/
    │   ├── chain/
    │   ├── transfers/
    │   ├── wallet/
    │   ├── crosschain/
    │   ├── fx/
    │   ├── agents/
    │   ├── graph/
    │   └── auth/
    │
    └── stores/                    ✅ one $state store per domain, all fetch functions
```

---

## Engineering Codes (Key Technical Decisions)

### City block grid layout
50 blocks laid flat on the XZ plane. Each parcel is a fixed-width square (e.g. 3×3 units) with a 0.5-unit street gap. Block height (Y) = `utilization_pct` scaled to 0.1–1.5 units. Block color = `utilization_pct` mapped through HSL (210° blue → 25° orange). Newest block at Z=0, history at Z=-N. New blocks shift everything back by one slot on websocket event.

### Per-block mini force layout
When a block is focused, fetch its transactions and transfers, collect the unique wallet addresses, run a constrained d3-force-3d sim with the parcel footprint as bounds (radial force keeps nodes within the parcel's XZ square). 50–100 nodes max per block, sim settles in <200ms. Tear down the sim on block unfocus.

### Two-mode camera
Camera has a `mode` reactive variable: `city` or `block`. In `city` mode: PerspectiveCamera at (0, 20, 10), OrbitControls for free orbit. In `block` mode: camera tweens to (parcel.x, 4, parcel.z + 6), OrbitControls constrained. Transition via Threlte `useTween`. Escape or outside-click returns to `city`.

### InstancedMesh for buildings
Wallets inside a focused block use InstancedMesh (pool 200 per block). Position and scale updated from the force sim result. Agent towers use a separate InstancedMesh with taller geometry, always visible across both camera modes.

### Bounded courier fleet
Ring buffer (500 slots) for live couriers. Each frame `useTask` advances courier `t` from 0→1 along its edge, frees the slot at completion. Couriers route between their source wallet's parcel position and destination wallet's parcel position — cross-block couriers travel visibly across the grid.

### Frame ticker
All animation loops use `useTask((delta: number) => {...})` — Threlte v8 convention.

### No scaffolding tricks
Threlte's `Suspense` and `Align` mutate internal `$state` from Promise callbacks, which Svelte 5 runes forbids. Text3DGeometry centering done manually via `oncreate` + bounding box on the Three.js mesh.

### Resident paths (cross-block edges)
`wallet_edges` records are cumulative — they represent the total relationship between two wallets across all blocks. Rendered as faint LineSegments connecting the parcel positions of their last-seen blocks. Not per-block, just a city-wide network overlay.

### Block filter for transfers
`transfers` has `block_number` in the DB schema but the handler doesn't expose a `?block=N` filter yet. Add it to `transfersHandler` — same pattern as the existing `?block=` param in `transactionsHandler`. Required for BlockStreets to work.

### Clean eviction
All `pb.collection(...).subscribe(...)` calls return an unsubscribe function. Cleaned up in `onDestroy` on Scene. Per-block force sim torn down when block is unfocused.

---

## City Capacity Plan (Performance Budget)

| Component | Strategy |
|---|---|
| City block parcels (50) | InstancedMesh, one draw call for all parcel platforms |
| Block buildings (≤200 wallets per focused block) | InstancedMesh, pool per block, torn down on unfocus |
| Block streets (transfer edges within block) | BufferGeometry LineSegments, rebuilt on focus |
| Resident paths (cross-block edges, ~500) | BufferGeometry LineSegments, built once on load |
| Agent towers | InstancedMesh, always rendered, small pool |
| Couriers (particles) | Ring buffer, max 500 alive at once |
| Harbor arcs | Max 20 animated at once |
| FX bridges | Max 20 animated, fade after 3s |
| Dashboard sparklines | Raw SVG `$derived` from blockStats store |

Target: 60fps on mid-range hardware in CityView; 60fps in BlockView with full block content.

---

## Construction Timeline (Implementation Phases)

### Phase 1 — Foundation ✅
- Threlte + Three.js installed, full-viewport `<Canvas>`
- `SceneCamera` — PerspectiveCamera + OrbitControls
- `SceneLighting` — ambient + accent lights

### Phase 2 — City Grid (replaces helix)
- `CityGrid.svelte` — lays 50 `BlockParcel` instances in a strip on XZ plane
- `BlockParcel.svelte` — InstancedMesh platform, height = `utilization_pct`, heat color
- Street gaps between parcels
- Newest block pulses ("city heartbeat")
- Live: new block arrives → grid shifts, new parcel added at front

### Phase 3 — Block Drill-Down (replaces global wallet graph)
- Add `?block=N` filter to `transfersHandler` in `handlers.go`
- `BlockBuildings.svelte` — fetches txs + transfers for focused block, runs mini force sim, renders wallets as InstancedMesh within parcel bounds
- `BlockStreets.svelte` — LineSegments between wallets for transfers within the block
- `AgentTowers.svelte` — agent wallets marked on their parcels, always visible
- Camera tween between CityView and BlockView
- `BlockInspector` HUD panel

### Phase 4 — City Traffic ✅
- `scene-state/particles.svelte.ts` ring buffer (max 500 live couriers)
- `TokenParticles.svelte` — couriers lerp between wallet parcel positions, color by token
- `ParticleSpawner.svelte` — watches transfers + layout, dispatches couriers
- Cross-block couriers travel visibly across the grid

### Phase 5 — Resident Paths
- `ResidentPaths.svelte` — renders `wallet_edges` as faint cross-block LineSegments
- Connects wallets' last-known parcel positions
- Fades with distance from newest block

### Phase 6 — Live City
- PocketBase websocket subscriptions
- New block → parcel added at grid front
- New transfer → courier dispatched
- New agent → tower marked
- New job → work order arc fired

### Phase 7 — Harbor + Exchange
- `CrosschainArrows` — Bezier arcs arriving at the city border (from outside the grid)
- `FXSwapArcs` — bidirectional USDC↔EURC exchange bridge arcs

### Phase 8 — City Ops Dashboard
- `StatsBar`, `TokenFlowPanel`, `MiniCharts`, `AgentCounter`
- `LayerToggles` — toggle parcels / buildings / paths / couriers / harbor
- `BlockInspector` — block detail + wallet list + transfer list

### Phase 9 — Exploration Polish
- Bloom on agent towers + hot parcels (`@threlte/extras` EffectComposer)
- Fog for depth — city recedes into haze
- Smooth idle auto-orbit in CityView
- Double-click parcel to focus, Escape to return
- Mobile fallback (no couriers, simplified geometry, orthographic camera)
