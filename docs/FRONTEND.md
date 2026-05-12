# Arcadia Frontend Plan

## Vision

A fullscreen 3D visualizer for the Arc L1 chain. The centrepiece is a wireframe globe representing Arc — capital flows in from outside (cross-chain), circulates inside (transfers, agent jobs, FX swaps), and the chain spine runs through the Z axis as time. Everything is live via PocketBase websockets. No dashboards, no tables — just the chain breathing.

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
| Sphere grid / skeleton | `#9aa0b4` at 25–40% opacity |
| ARCADIA logotype | `#e0e0ee`, extruded 3D, `metalness=0.55` |
| Wallet nodes | `#ffffff` |
| Agent nodes | `#7ee5a8` (green) |
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
├── <SceneCamera>          ✅  PerspectiveCamera at (0,1.5,10), OrbitControls autoRotate
├── <SceneLighting>        ✅  Ambient + cyan point light + green point light
│
├── <ArcSphere>            ✅  Wireframe globe — visual anchor
│   ├── <SphereGrid>       ✅  SphereGeometry wireframe, #9aa0b4 25% opacity
│   ├── hex core           ✅  Flat CylinderGeometry(6 sides) wireframe #7ee5a8, subtle pulse
│   ├── ARCADIA logotype   ✅  Text3DGeometry extruded, MeshStandardMaterial metalness=0.55
│   └── <CrosschainArrows>     Bezier arcs entering the sphere from outside (CCTP/Gateway)
│
├── <ChainSpine>           ✅  Fetches 50 blocks on mount
│   └── <BlockNodes>       ✅  InstancedMesh, Z-axis spine, utilization heat colour
│
├── <WalletGraph>              Force-directed graph of wallet activity
│   ├── <WalletNodes>          InstancedMesh — large pool, visible nodes = active wallets
│   ├── <AgentNodes>           Subset with green glow + pulse
│   ├── <TransferEdges>        LineSegments, width ∝ transfer amount
│   └── <JobArcs>              Curved lines employer↔worker (amber)
│
├── <TokenParticles>           Ring buffer, particles travel along edges
│   ├── <USDCParticles>        Blue
│   ├── <EURCParticles>        Gold
│   └── <USYCParticles>        Purple
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
  → d3-force-3d runs on wallet edges → produces x/y/z positions
  → WalletNodes + TransferEdges placed
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
    │   ├── SceneCamera.svelte     ✅ PerspectiveCamera + OrbitControls autoRotate
    │   ├── SceneLighting.svelte   ✅ Ambient + point lights (cyan + green)
    │   │
    │   ├── sphere/
    │   │   ├── ArcSphere.svelte   ✅ Hex core + Text3DGeometry logotype
    │   │   ├── SphereGrid.svelte  ✅ Wireframe globe
    │   │   └── CrosschainArrows.svelte
    │   │
    │   ├── chain/
    │   │   ├── ChainSpine.svelte  ✅ Fetches blocks, mounts BlockNodes
    │   │   └── BlockNodes.svelte  ✅ InstancedMesh Z-spine, heat colour, axle
    │   │
    │   ├── graph/
    │   │   ├── WalletGraph.svelte
    │   │   ├── WalletNodes.svelte
    │   │   ├── AgentNodes.svelte
    │   │   ├── TransferEdges.svelte
    │   │   └── JobArcs.svelte
    │   │
    │   ├── particles/
    │   │   ├── TokenParticles.svelte
    │   │   ├── USDCParticles.svelte
    │   │   ├── EURCParticles.svelte
    │   │   └── USYCParticles.svelte
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
    │   ├── layout.svelte.ts       d3-force-3d simulation → node x/y/z positions
    │   ├── particles.svelte.ts    ring buffer of live transfers for particle spawning
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

### d3-force-3d for wallet layout
The `wallet_edges` data has `tx_count` as edge strength and `from_is_agent`/`to_is_agent` flags. Running a force simulation on load gives organic, stable 3D positioning. Agent nodes get a separate charge modifier. Re-run debounced 5s when new edges arrive.

### Separate geometry per token type
USDC, EURC, and USYC particles use different `THREE.BufferGeometry` instances so each can have its own colour without shader branching.

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

### Phase 1 — Scene shell ✅
- Threlte + Three.js installed
- Full-viewport `<Canvas>` in `+page.svelte`
- `SceneCamera` — PerspectiveCamera + OrbitControls autoRotate
- `SceneLighting` — ambient + cyan + green point lights
- `ArcSphere` — wireframe globe, hex core with pulse, ARCADIA Text3DGeometry

### Phase 2 — Chain spine ✅
- `ChainSpine` fetches 50 blocks on mount
- `BlockNodes` InstancedMesh along Z axis (z=+3 most recent → z=-3 oldest)
- Utilization heat colour (blue→orange via HSL)
- Node scale proportional to tx_count
- Faint axle cylinder threading the sphere

### Phase 3 — Wallet graph
- Install `d3-force-3d`
- `scene-state/layout.svelte.ts` — force simulation on wallet edges
- `WalletGraph` / `WalletNodes` InstancedMesh
- `TransferEdges` LineSegments
- `AgentNodes` green glow + pulse

### Phase 4 — Particles
- `scene-state/particles.svelte.ts` ring buffer
- `USDCParticles` travelling along edges
- Extend to EURC + USYC

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
