# Arcadia Frontend Plan

## Vision

A fullscreen 3D visualizer for the Arc L1 chain. The centrepiece is a **living neural graph** — wallet nodes positioned by force-directed layout form a brain-like 3D structure, connected by transfer edges that trace the network's synaptic pathways. There are no static decorative meshes. Every sphere, line, and particle represents real on-chain data: blocks, transfers, agents, jobs, FX swaps. The only non-data element is the ARCADIA logotype as a brand anchor. Capital flows in from outside (cross-chain), circulates inside (transfers, agent jobs, FX swaps), and the chain spine runs through the Z axis as time. Everything is live via PocketBase websockets. No dashboards, no tables — just the chain breathing.

**Design principle:** If a mesh doesn't represent data, it doesn't belong in the scene.

---

## Stack

| Layer | Choice | Reason |
|---|---|---|
| Framework | SvelteKit | SSR shell, file-based routing |
| 3D renderer | **Threlte** (`@threlte/core`, `@threlte/extras`) | Svelte-native Three.js; scene graph = component tree; Svelte 5 runes-compatible |
| Underlying 3D | Three.js (via Threlte) | Full access when needed |
| Graph layout | `d3-force-3d` | Force-directed wallet node placement in 3D space |
| Real-time | PocketBase websockets (on the `pb` instance) | Subscribe to collection changes, push to stores |
| Charts (HUD) | Raw SVG `$derived` | Lightweight time-series for the 2D overlay |
| Types / state | `$lib/api/` + `$lib/stores/` | Done — all 12 endpoints typed and stored |

**Installed:**
```
@threlte/core  @threlte/extras  three  @types/three
```
**Still needed:**
```
d3-force-3d
```

---

## Routes

```
/          → main 3D scene (Scene.svelte as full viewport canvas)   ✅
/debug     → API explorer with filter controls for every endpoint   ✅
```

The root page is a full-viewport Threlte `<Canvas>` with a 2D HUD overlay rendered in a Svelte layer on top via CSS `position: absolute`.

---

## Visual Design Language

Derived from the wireframe SVG (`docs/Arcadia_Wireframe.html`).

| Element | Value |
|---|---|
| Background | `#0a0a0f` |
| Wallet nodes | `#ffffff` — form the neural graph structure via force layout |
| Transfer edges | `#9aa0b4` at 25–40% opacity — the "wireframe" IS the transfer graph |
| Agent nodes | `#7ee5a8` (green) — glow + pulse, sized by `usdc_spent_fees` |
| Cross-chain inflows | `#6be3ff` (cyan) |
| USDC flow | `#2775ca` (Circle blue) |
| EURC flow | `#e8b84b` (EUR gold) |
| USYC flow | `#7b61ff` (purple, yield) |
| FX swap arcs | gradient USDC→EURC |
| Block heat (cold) | hsl(210°, blue) |
| Block heat (hot) | hsl(25°, orange) |
| Job arcs | `#f0a500` (amber) |

---

## API & Stores

All 12 REST endpoints are implemented, typed, and stored. Types match `collections.go` exactly.

| Endpoint | Store | Notes |
|---|---|---|
| `GET /api/v1/stats` | `stats` | Latest block_stats row + rolling 10-block TPS avg |
| `GET /api/v1/block_stats` | `blockStats` | History for time-series charts ✅ added to backend |
| `GET /api/v1/blocks` | `blocks` | Full Block type inc. `utilization_pct`, `miner`, `size` |
| `GET /api/v1/transactions` | `transactions` | All 22 fields inc. `fee_usdc`, `sighash`, `is_contract_deploy` |
| `GET /api/v1/transfers` | `transfers` | `amount_raw` + `amount_human`, token union type |
| `GET /api/v1/traces` | `traces` | `trace_type`, `gas_used`, `error_msg` |
| `GET /api/v1/crosschain` | `crosschain` | `amount_usdc`, `nonce_val`, protocol/event union types |
| `GET /api/v1/fx` | `fx` | `trade_id`, `taker_fee`, `maker_fee`, `status_code` |
| `GET /api/v1/agents` | `agents` | `metadata_uri`, `tx_count`, `usdc_spent_fees` |
| `GET /api/v1/agents/{address}` | `agent` | Single agent + job history |
| `GET /api/v1/jobs` | `agentJobs` | Full job lifecycle fields |
| `GET /api/v1/edges` | `graph` | `total_usdc`, `last_seen_block`, `from_is_agent`, `to_is_agent` |
| `GET /api/v1/wallet/{address}` | `wallet` | Typed txs/transfers/edges/agent |

---

## Scene Architecture

```
<Canvas> (full viewport, #0a0a0f background)
│
├── <SceneCamera>          ✅  PerspectiveCamera at (0,6,18), OrbitControls, pan enabled
├── <SceneLighting>        ✅  Ambient + cyan point light + green point light
│
├── <ChainSpine>           ✅  Fetches 50 blocks on mount
│   └── <BlockNodes>       ✅  InstancedMesh, Z-axis spine, utilization heat colour
│
├── <WalletGraph>          ✅  THE CENTRAL VISUAL — force-directed neural graph of all wallet activity
│   ├── <TransferEdges>    ✅  LineSegments — the "wireframe" of the brain, brightness ∝ total_usdc
│   ├── <WalletNodes>      ✅  InstancedMesh — all active wallets, force-laid in 3D spherical cluster
│   └── <AgentNodes>       ✅  Green emissive subset, pulse animation, stronger repulsion → surface
│
├── <CrosschainArrows>         Bezier arcs entering the graph from outside (CCTP/Gateway)
│
├── <TokenParticles>         ✅  Ring buffer particles traveling along transfer edges, color by token
│
├── <FXSwapArcs>               Bidirectional USDC↔EURC arcs (StableFX)
│
└── <FeeHeatmap>               Fee burn encoded into block node colour intensity
```

---

## 2D HUD (Svelte overlay, `position: absolute` over the canvas)

```
+page.svelte
│
├── <Scene>  (Threlte Canvas, z-index 0)
│
└── <HUD>    (absolute overlay, pointer-events: none except interactive bits)
    ├── <StatsBar>          Top-left: TPS · Block time · Indexed block · Fee avg
    ├── <TokenFlowPanel>    Top-right: USDC / EURC / USYC volume this block
    ├── <AgentCounter>      Top-right below: Active agents · Open jobs
    ├── <LayerToggles>      Bottom-right: toggle each visual layer on/off
    ├── <MiniCharts>        Bottom-left: sparklines for TPS + USDC volume (last 50 blocks)
    └── <WalletInspector>   Right panel, slides in on wallet node click
```

---

## Data Flow

### Initial load
```
onMount
  → fetchStats() + fetchBlockStats() + fetchBlocks() + fetchAgents() + fetchEdges() (parallel)
  → stores populate
  → scene reads stores via $derived / reactive props
  → d3-force-3d runs on wallet edges → produces x/y/z positions with spherical boundary
  → WalletNodes + TransferEdges placed — THIS IS THE CENTRAL VISUAL (the "brain")
  → BlockNodes placed on Z spine through the graph
```

### Real-time (PocketBase websockets)
```
pb.collection('block_stats').subscribe('*', handler)
  → updates stats store → StatsBar + MiniCharts re-derive

pb.collection('transfers').subscribe('*', handler)
  → pushes to ring buffer (last 200 transfers)
  → spawns new particles in TokenParticles

pb.collection('crosschain_events').subscribe('*', handler)
  → triggers new CrosschainArrow animation

pb.collection('fx_swaps').subscribe('*', handler)
  → triggers new FXSwapArc animation

pb.collection('agent_jobs').subscribe('*', handler)
  → updates job arcs, pulses employer/worker nodes

pb.collection('agents').subscribe('*', handler)
  → adds new agent node to WalletGraph with green glow
```

### Interaction
```
click wallet node
  → raycast hit → get wallet address
  → fetchWallet(address)
  → WalletInspector slides in with sent/received/edges/agent status

scroll / drag → OrbitControls handles it natively
double-click → focus camera on node (Threlte useTween)
```

---

## Component File Structure

```
src/
├── routes/
│   ├── +page.svelte              ✅ full-viewport shell: <Scene> + <HUD> placeholder
│   └── debug/
│       └── +page.svelte          ✅ API explorer — all 12 endpoints, full filter controls
│
└── lib/
    ├── scene/
    │   ├── Scene.svelte           ✅ Threlte <Canvas> entry point
    │   ├── SceneCamera.svelte     ✅ PerspectiveCamera + OrbitControls (pan, no auto-rotate)
    │   ├── SceneLighting.svelte   ✅ Ambient + point lights (cyan + green)
    │   │
    │   ├── chain/
    │   │   ├── ChainSpine.svelte  ✅ Fetches blocks, mounts BlockNodes
    │   │   └── BlockNodes.svelte  ✅ Helix on X axis, heat colour, chain links, head pulse
    │   │
    │   ├── graph/                    ← THE CENTRAL VISUAL
    │   │   ├── WalletGraph.svelte    ✅ Fetches edges + agents + transfers, runs simulation
    │   │   ├── WalletNodes.svelte    ✅ Non-agent wallet InstancedMesh
    │   │   ├── AgentNodes.svelte     ✅ Green emissive agent InstancedMesh + pulse
    │   │   ├── TransferEdges.svelte  ✅ LineSegments, brightness ∝ total_usdc
    │   │   └── JobArcs.svelte
    │   │
    │   ├── particles/
    │   │   ├── TokenParticles.svelte   ✅ InstancedMesh, color-coded by token, travel along edges
    │   │   └── ParticleSpawner.svelte   ✅ Watches transfers + layout, spawns on load
    │   │
    │   └── fx/
    │       └── FXSwapArcs.svelte
    │
    ├── hud/
    │   ├── HUD.svelte
    │   ├── StatsBar.svelte
    │   ├── TokenFlowPanel.svelte
    │   ├── AgentCounter.svelte
    │   ├── LayerToggles.svelte
    │   ├── MiniCharts.svelte
    │   └── WalletInspector.svelte
    │
    ├── scene-state/
    │   ├── layout.svelte.ts       ✅ d3-force-3d simulation → node x/y/z positions (spherical boundary)
    │   ├── particles.svelte.ts    ✅ ring buffer of live transfers for particle spawning
    │   ├── layers.svelte.ts       boolean toggles for each visual layer
    │   └── selection.svelte.ts    currently selected wallet node
    │
    ├── api/                       ✅ all 12 endpoints — types match collections.go exactly
    │   ├── stats/
    │   ├── block_stats/           ✅ added (new backend endpoint)
    │   ├── chain/
    │   ├── transfers/
    │   ├── wallet/
    │   ├── crosschain/
    │   ├── fx/
    │   ├── agents/
    │   ├── graph/
    │   └── auth/                  (scaffolded, not used)
    │
    └── stores/                    ✅ one $state store per domain, all fetch functions
```

---

## Key Technical Decisions

### InstancedMesh for nodes
Wallet nodes will number in the thousands as the chain grows. `THREE.InstancedMesh` renders all of them in a single draw call. Each instance gets a matrix (position/scale) and a colour attribute updated per frame from the stores. Already used for BlockNodes.

### Ring buffer for particles
Live transfers come in continuously. A fixed-size ring buffer (500 slots) holds active particles. Each frame `useTask` advances particle `t` from 0→1 along its edge curve, then frees the slot. This bounds GPU memory regardless of chain activity.

### useTask not useFrame
Threlte v8 renamed `useFrame` → `useTask`. All animation loops use `useTask((delta: number) => {...})`.

### Suspense + Align avoided
Threlte's `Suspense` and `Align` components mutate internal `$state` from inside Promise callbacks, which Svelte 5 runes mode forbids. Text3DGeometry is used without Suspense; centering is done manually via `oncreate` callback + bounding box computation directly on the Three.js mesh.

### d3-force-3d for wallet layout — the neural graph
The `wallet_edges` data has `tx_count` as edge strength and `from_is_agent`/`to_is_agent` flags. A force simulation positions wallets in 3D with a **spherical boundary constraint** — nodes naturally cluster into a brain-like shape without any static sphere mesh. Transfer edges between nodes create the "wireframe" organically. Agent nodes get a separate charge modifier (stronger repulsion → they sit on the surface). Re-run debounced 5s when new edges arrive.

### Separate geometry per token type
USDC, EURC, and USYC particles use different `THREE.BufferGeometry` instances so each can have its own colour without shader branching.

### No static decorative meshes
Every mesh in the scene represents on-chain data. The old `SphereGrid` (static wireframe globe) and hex core (decorative cylinder) have been removed. The "globe" shape now emerges from the force-directed wallet layout itself. The only exception is the ARCADIA logotype, kept as a brand anchor.

### Raycasting for interaction
On click, cast a ray against the WalletNodes InstancedMesh, get the instance index, look up the wallet address from the layout store, trigger `fetchWallet`.

### PocketBase subscription cleanup
All `pb.collection(...).subscribe(...)` calls return an unsubscribe function. Clean up in `onDestroy` on Scene to prevent leaks on HMR.

---

## Performance Budget

| Component | Strategy |
|---|---|
| Wallet nodes (up to ~10k) | InstancedMesh, single draw call |
| Block nodes (50 max) | InstancedMesh, sliding window |
| Transfer edges | BufferGeometry LineSegments, rebuild debounced |
| Particles | Ring buffer, max 500 alive at once |
| Cross-chain arrows | Max 20 animated at once |
| FX arcs | Max 20 animated, fade out after 3s |
| HUD charts | Raw SVG `$derived` from blockStats store |

Target: 60fps on mid-range hardware with all layers active.

---

## Implementation Phases

### Phase 1 — Scene shell ✅ (updated — static meshes removed)
- Threlte + Three.js installed
- Full-viewport `<Canvas>` in `+page.svelte`
- `SceneCamera` — PerspectiveCamera + OrbitControls autoRotate
- `SceneLighting` — ambient + cyan + green point lights
- ~~`ArcSphere` — wireframe globe, hex core~~ → **Removed.** Replaced by data-driven `WalletGraph` (Phase 3)
- ~~`ArcLogotype` — Text3DGeometry~~ → **Removed.** No static meshes at all.

### Phase 2 — Chain spine ✅ (upgraded — helix layout + chain links)
- `ChainSpine` fetches 50 blocks on mount
- `BlockNodes` InstancedMesh in DNA **helix layout** (2 turns over Z +4 → -4, radius 0.65)
- IcosahedronGeometry (detail 1) with MeshStandardMaterial — lit by scene lights
- Utilization heat colour (blue→orange via HSL) with age-based dimming (newest bright, oldest dim)
- Node scale 0.08–0.26 proportional to tx_count
- **Chain links** — LineSegments connecting consecutive blocks, vertex-colored by heat
- Head block pulses ±15% ("heartbeat") via useTask
- ~~Faint axle cylinder~~ → **Removed.** Chain links ARE the spine now.

### Phase 3 — Wallet graph ✅ THE CENTRAL VISUAL — the "brain"
- `d3-force-3d` installed
- `scene-state/layout.svelte.ts` — force simulation on wallet edges with spherical boundary constraint (radius 3.0, radial force at 2.1)
- `WalletGraph` container — fetches 500 edges + 200 agents on mount, runs simulation
- `WalletNodes` InstancedMesh (pool 5000) — white IcosahedronGeometry, scale by txCount
- `TransferEdges` LineSegments — the "wireframe" of the brain, brightness ∝ total_usdc (HSL lightness 0.15–0.50)
- `AgentNodes` InstancedMesh (pool 500) — green emissive, scale by txCount, pulse ±12% at 1.8 rad/s
- Agent nodes get stronger repulsion (charge -20 vs -6) → float to graph surface
- Link strength ∝ tx_count (more active connections pull nodes closer)

### Phase 4 — Particles ✅
- `scene-state/particles.svelte.ts` ring buffer (max 500 live particles)
- `TokenParticles.svelte` — single InstancedMesh, particles lerp along edges, swell mid-journey
- Color-coded by token type: USDC `#2775ca`, EURC `#e8b84b`, USYC `#7b61ff`
- `ParticleSpawner.svelte` — watches transfers + layout, spawns on initial load
- Particles auto-die when `t >= 1`, ticked every frame via `useTask`
- WalletGraph fetches 200 transfers on mount as particle source data

### Phase 5 — Live data
- PocketBase websocket subscriptions (block_stats, transfers, agents, jobs, crosschain, fx)
- New blocks animate into spine
- New transfers spawn particles
- New agents add glowing node
- New jobs draw arc

### Phase 6 — Cross-chain + FX
- `CrosschainArrows` — Bezier arcs entering the sphere from outside
- `FXSwapArcs` — bidirectional USDC↔EURC curves

### Phase 7 — HUD
- `StatsBar` — TPS, block time, avg fee
- `TokenFlowPanel` — per-block USDC/EURC/USYC volumes
- `MiniCharts` — sparklines from `blockStats` store (last 50 blocks)
- `LayerToggles` — `scene-state/layers.svelte.ts`

### Phase 8 — Interaction
- Raycast on click → `WalletInspector` panel slides in
- Camera focus animation on selected node
- Double-click to zoom/reset

### Phase 9 — Polish
- Bloom on agent nodes + hot blocks (`@threlte/extras` EffectComposer)
- Fog for depth cueing
- Smooth idle auto-orbit
- Mobile fallback (no particles, simplified geometry)
