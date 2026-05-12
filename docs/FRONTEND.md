# Arcadia Frontend Plan

## Vision

A fullscreen 3D visualizer for the Arc L1 chain. The centrepiece is a wireframe globe representing Arc — capital flows in from outside (cross-chain), circulates inside (transfers, agent jobs, FX swaps), and the chain spine runs through the Z axis as time. Everything is live via PocketBase websockets. No dashboards, no tables — just the chain breathing.

---

## Stack

| Layer | Choice | Reason |
|---|---|---|
| Framework | SvelteKit (already set up) | SSR shell, file-based routing |
| 3D renderer | **Threlte** (`@threlte/core`, `@threlte/extras`) | Svelte-native Three.js; scene graph = component tree; Svelte 5 runes-compatible |
| Underlying 3D | Three.js (via Threlte) | Full access when needed |
| Graph layout | `d3-force-3d` | Force-directed wallet node placement in 3D space |
| Real-time | PocketBase websockets (already on the `pb` instance) | Subscribe to collection changes, push to stores |
| Charts (HUD) | `layerchart` or raw SVG `$derived` | Lightweight time-series for the 2D overlay |
| Types / state | Existing stores in `$lib/stores/` | Already built for all 10 endpoints |

**Dependencies to add:**
```
@threlte/core
@threlte/extras
three
@types/three
d3-force-3d
```

---

## Routes

```
/          → main 3D scene (Scene.svelte as full viewport canvas)
/debug     → API explorer (already built)
```

The root page is a full-viewport Threlte `<Canvas>` with a 2D HUD overlay rendered in a Svelte layer on top via CSS `position: absolute`.

---

## Visual Design Language

Derived from the wireframe SVG.

| Element | Value |
|---|---|
| Background | `#0a0a0f` |
| Sphere grid / skeleton | `#9aa0b4` at 40–60% opacity |
| ARCADIA logotype | `#e6e6ee`, monospace, wide letter-spacing |
| Wallet nodes | `#ffffff` |
| Agent nodes | `#7ee5a8` (green) |
| Cross-chain inflows | `#6be3ff` (cyan) |
| USDC flow | `#2775ca` (Circle blue) |
| EURC flow | `#e8b84b` (EUR gold) |
| USYC flow | `#7b61ff` (purple, yield) |
| FX swap arcs | gradient USDC→EURC |
| Block heat (cold) | `#4a90d9` |
| Block heat (hot) | `#ff6b35` |
| Job arcs | `#f0a500` (amber) |

---

## Scene Architecture

```
<Canvas> (full viewport, #0a0a0f background)
│
├── <SceneCamera>          OrbitControls, auto-rotate off while interacting
├── <SceneLighting>        Ambient low + two point lights (cyan + green tints)
│
├── <ArcSphere>            The wireframe globe — visual anchor of the whole scene
│   ├── <SphereGrid>       Latitude + longitude lines, #9aa0b4 wireframe
│   └── <CrosschainArrows> Bezier arcs entering the sphere from outside (CCTP/Gateway)
│
├── <ChainSpine>           Blocks placed along the Z axis (time)
│   └── <BlockNodes>       InstancedMesh — one instance per block, colour = utilization heat
│
├── <WalletGraph>          Force-directed graph of wallet activity
│   ├── <WalletNodes>      InstancedMesh — large pool, visible nodes = active wallets
│   ├── <AgentNodes>       Subset of wallet nodes with green glow + pulse animation
│   ├── <TransferEdges>    Lines between wallets, width proportional to transfer amount
│   └── <JobArcs>          Curved lines between employer↔worker agent wallets (amber)
│
├── <TokenParticles>       Particles that travel along transfer edges, per token type
│   ├── <USDCParticles>    Blue — most common, highest density
│   ├── <EURCParticles>    Gold
│   └── <USYCParticles>    Purple
│
├── <FXSwapArcs>           Bidirectional USDC↔EURC arcs (StableFX), gradient colour
│
└── <FeeHeatmap>           Per-block USDC fee burn encoded into block node colour intensity
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
    └── <WalletInspector>   Right panel, slides in on wallet node click (wallet profile data)
```

---

## Data Flow

### Initial load
```
onMount
  → fetchStats() + fetchBlocks() + fetchAgents() + fetchEdges() (parallel)
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
  → pushes to a ring buffer (last 200 transfers)
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
double-click → focus camera on node (GSAP or built-in Threlte useTween)
```

---

## Component File Structure

```
src/
├── routes/
│   ├── +page.svelte              full-viewport shell: <Scene> + <HUD>
│   └── debug/
│       └── +page.svelte          API explorer (done)
│
└── lib/
    ├── scene/
    │   ├── Scene.svelte           Threlte <Canvas> entry point
    │   ├── SceneCamera.svelte     PerspectiveCamera + OrbitControls
    │   ├── SceneLighting.svelte   Ambient + point lights
    │   │
    │   ├── sphere/
    │   │   ├── ArcSphere.svelte
    │   │   ├── SphereGrid.svelte
    │   │   └── CrosschainArrows.svelte
    │   │
    │   ├── chain/
    │   │   ├── ChainSpine.svelte
    │   │   └── BlockNodes.svelte
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
    ├── api/                       (done)
    └── stores/                    (done)
```

---

## Key Technical Decisions

### InstancedMesh for nodes
Wallet nodes will number in the thousands as the chain grows. `THREE.InstancedMesh` renders all of them in a single draw call. Each instance gets a matrix (position/scale) and a colour attribute updated per frame from the stores.

### Ring buffer for particles
Live transfers come in continuously. A fixed-size ring buffer (e.g. 500 slots) holds active particles. Each frame `useFrame` advances particle `t` from 0→1 along its edge curve, then frees the slot. This bounds GPU memory regardless of chain activity.

### d3-force-3d for wallet layout
The `wallet_edges` data already has `tx_count` as edge strength. Running a force simulation on load gives organic, stable 3D positioning. Nodes with more transfers cluster together. Agent nodes get a separate charge modifier so they stand out spatially. Re-run the simulation when new edges arrive (debounced 5s).

### Separate geometry per token type
USDC, EURC, and USYC particles use different `THREE.BufferGeometry` instances so each can have its own shader/colour without branching in the shader. Small memory cost, big visual clarity gain.

### Raycasting for interaction
`useThree` gives access to the renderer + camera. On click, cast a ray against the WalletNodes InstancedMesh, get the instance index, look up the wallet address from the layout store, trigger `fetchWallet`.

### PocketBase subscription cleanup
All `pb.collection(...).subscribe(...)` calls return an unsubscribe function. Clean up in `onDestroy` on the Scene component to prevent leaks on HMR.

---

## Performance Budget

| Component | Strategy |
|---|---|
| Wallet nodes (up to ~10k) | InstancedMesh, single draw call |
| Transfer edges | BufferGeometry LineSegments, rebuild on new edges (debounced) |
| Particles | Ring buffer, max 500 alive at once |
| Block nodes | ~50 visible blocks max (sliding window), InstancedMesh |
| Cross-chain arrows | Low frequency events, max 20 animated at once |
| FX arcs | Max 20 animated at once, fade out after 3s |
| HUD charts | Raw SVG, $derived from stores — no canvas overhead |

Target: 60fps on mid-range hardware with all layers active.

---

## Implementation Phases

### Phase 1 — Scene shell
- Install Threlte + Three.js
- Full-viewport `<Canvas>` in `+page.svelte`
- Camera with OrbitControls
- Lighting
- ArcSphere wireframe globe (static)
- "ARCADIA" logotype in 3D space (Threlte Text)

### Phase 2 — Chain spine
- Fetch blocks on mount
- BlockNodes InstancedMesh along Z axis
- Utilization heat colour
- FeeHeatmap colour intensity overlay

### Phase 3 — Wallet graph
- Load wallet edges on mount
- Run d3-force-3d to position nodes
- WalletNodes InstancedMesh
- TransferEdges as LineSegments
- AgentNodes highlighted green with pulse

### Phase 4 — Particles
- Ring buffer state
- USDCParticles travelling along edges
- Extend to EURC + USYC

### Phase 5 — Live data
- Wire up all PocketBase subscriptions
- New blocks → animate into spine
- New transfers → spawn particles
- New agents → add glowing node
- New jobs → draw job arc

### Phase 6 — Cross-chain + FX
- CrosschainArrows entering the sphere from outside
- FXSwapArcs USDC↔EURC bidirectional

### Phase 7 — HUD
- StatsBar (TPS, block time, fee)
- TokenFlowPanel (volumes)
- MiniCharts sparklines
- LayerToggles

### Phase 8 — Interaction
- Raycast on click → WalletInspector panel
- Camera focus animation on selected node
- Double-click to zoom/reset

### Phase 9 — Polish
- Post-processing: bloom on agent nodes + hot blocks (`@threlte/extras` EffectComposer)
- Fog for depth cueing
- Smooth camera auto-orbit when idle
- Mobile fallback (simplified scene, no particles)
