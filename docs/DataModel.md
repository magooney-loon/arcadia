# Arcadia — Data Model & Analytics Reference

> Complete guide to every data collection, API endpoint, frontend store,
> available metric, and how to surface it as a quant-grade analytics platform.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Data Collections (Schema)](#2-data-collections-schema)
3. [API Endpoints](#3-api-endpoints)
4. [Frontend Data Layer](#4-frontend-data-layer)
5. [Metrics & Visualizations — What to Show & How](#5-metrics--visualizations--what-to-show--how)
6. [Derived / Computed Metrics (Not Yet Stored)](#6-derived--computed-metrics-not-yet-stored)
7. [Page-by-Page Analytics Blueprint](#7-page-by-page-analytics-blueprint)
8. [Gaps & Opportunities](#8-gaps--opportunities)

---

## 1. Architecture Overview

```
┌──────────────┐     HyperSync / JSON-RPC      ┌──────────────┐
│  Arc L1 Node │ ◄────────────────────────────  │   Indexer    │
│  (EVM chain) │                                │  (Go daemon) │
└──────────────┘                                └──────┬───────┘
    USDC native gas · ERC-20s                          │
    CCTP · Gateway · StableFX · ERC-8004              │ upsert
                                                       ▼
                                              ┌──────────────┐
                                              │  PocketBase  │
                                              │  (12 colls)  │
                                              └──────┬───────┘
                                                     │ REST
                                                     ▼
                                              ┌──────────────┐
                                              │  SvelteKit   │
                                              │  Frontend    │
                                              └──────────────┘
```

### Key characteristics

| Trait | Value |
|---|---|
| Gas token | USDC (18 decimals) |
| Finality | Sub-second, single-slot |
| Stablecoins | USDC, EURC, USYC (6-decimal ERC-20s) |
| Cross-chain | CCTP v2 burns/mints, Gateway deposits/withdrawals |
| DeFi | StableFX (FxEscrow — RFQ maker/taker swap) |
| Agent economy | ERC-8004 registrations, ERC-8183 job lifecycle |
| Graph | wallet_edges — directional USDC flow + tx_count |

---

## 2. Data Collections (Schema)

### 2.1 `indexer_meta` — Key/Value cursor store

| Field | Type | Description |
|---|---|---|
| `key` | text (unique) | e.g. `lastBlock` |
| `value` | text | e.g. `"48291031"` |

**What it enables**: Shows indexer lag = `chain_tip - lastBlock`.

---

### 2.2 `indexer_events` — Indexer lifecycle & error log

| Field | Type | Description |
|---|---|---|
| `timestamp` | number | Unix epoch |
| `level` | select | `debug` / `info` / `warn` / `error` |
| `event` | text | Event name (max 80 chars) |
| `message` | text | Human-readable message |
| `attempt` | number | Retry attempt number |
| `batch` | number | Batch size |
| `block` | number | Current block |
| `tip` | number | Chain tip |
| `lag` | number | `tip - block` |
| `duration_ms` | number | Batch processing time |
| `blocks` | number | Blocks in batch |
| `transactions` | number | Txs in batch |
| `logs` | number | Logs in batch |
| `error` | text | Error string |

**Auto-cleaned**: records older than 2 hours are pruned every hour.

**What it enables**: Indexer health dashboard, error rate, processing throughput, lag over time.

---

### 2.3 `blocks` — L1 chain skeleton

| Field | Type | Indexed | Description |
|---|---|---|---|
| `number` | number | ✅ unique | Block height |
| `hash` | text | ✅ unique | Block hash |
| `parent_hash` | text | | Parent hash |
| `miner` | text | | Validator/miner address |
| `timestamp` | number | | Unix epoch |
| `gas_used` | number | | Gas consumed |
| `gas_limit` | number | | Block gas limit |
| `base_fee_per_gas` | text | | Big.Int string (sub-wei precision) |
| `size` | number | | Block size in bytes |
| `tx_count` | number | | Derived: number of transactions |
| `block_time_ms` | number | | Derived: ms since previous block |
| `utilization_pct` | number | | Derived: `gas_used / gas_limit * 100` |

**What it enables**: Block explorer, TPS calculation, gas utilization trends, block time histograms.

---

### 2.4 `transactions` — Every on-chain transaction

| Field | Type | Indexed | Description |
|---|---|---|---|
| `hash` | text | ✅ unique | Tx hash |
| `block_number` | number | ✅ | Block height |
| `transaction_index` | number | | Position in block |
| `from_addr` | text | ✅ | Sender |
| `to_addr` | text | ✅ | Recipient (nil for deploys) |
| `value` | text | | Native USDC transfer (uint256 string) |
| `nonce` | number | | Account nonce |
| `sighash` | text | | First 4 bytes of calldata (method selector) |
| `gas_price` | text | | Submitted gas price |
| `gas_limit` | number | | Gas limit |
| `gas_used` | number | | Actual gas consumed |
| `cumulative_gas_used` | number | | Cumulative gas in block up to this tx |
| `effective_gas_price` | text | | Post-EIP-1559 effective price |
| `max_fee_per_gas` | text | | EIP-1559 max fee |
| `max_priority_fee_per_gas` | text | | EIP-1559 priority fee cap |
| `priority_fee_per_gas` | text | | Actual priority fee |
| `fee_usdc` | text | | `gas_used × effective_gas_price / 1e18` |
| `priority_fee_usdc` | text | | `gas_used × priority_fee_per_gas / 1e18` |
| `tx_type` | number | | 0=legacy, 1=access-list, 2=EIP-1559 |
| `status` | number | | 1=success, 0=reverted |
| `contract_address` | text | | Created contract address (if deploy) |
| `is_contract_deploy` | bool | | True if contract creation |

**What it enables**: Mempool analytics, fee analysis, method popularity (sighash), success rate, contract deploy tracking.

---

### 2.5 `transfers` — ERC-20 token transfers

| Field | Type | Indexed | Description |
|---|---|---|---|
| `tx_hash` | text | ✅ (compound) | Transaction hash |
| `block_number` | number | ✅ | Block height |
| `log_index` | number | ✅ (compound) | Log position |
| `token_address` | text | ✅ | ERC-20 contract |
| `token_symbol` | select | | `USDC` / `EURC` / `USYC` / `OTHER` |
| `from_addr` | text | ✅ | Sender |
| `to_addr` | text | ✅ | Receiver |
| `amount_raw` | text | | Raw uint256 |
| `amount_human` | text | | `/1e6` human-readable |

**What it enables**: Token flow analysis, whale alerts, volume charts by token, top senders/receivers, transfer heatmaps.

---

### 2.6 `traces` — Internal contract-to-contract calls

| Field | Type | Indexed | Description |
|---|---|---|---|
| `tx_hash` | text | ✅ | Transaction hash |
| `block_number` | number | ✅ | Block height |
| `from_addr` | text | | Caller contract |
| `to_addr` | text | | Target contract |
| `value` | text | | Internal value transfer |
| `call_type` | text | | `CALL`, `DELEGATECALL`, `STATICCALL` |
| `trace_type` | text | | `call`, `create`, `suicide` |
| `gas_used` | number | | Gas consumed by internal call |
| `error_msg` | text | | Revert reason if failed |

**What it enables**: Contract interaction graphs, failed internal call analysis, delegate call monitoring, MEV detection.

---

### 2.7 `crosschain_events` — CCTP & Gateway bridge events

| Field | Type | Indexed | Description |
|---|---|---|---|
| `tx_hash` | text | ✅ (compound) | Transaction hash |
| `block_number` | number | ✅ | Block height |
| `log_index` | number | ✅ (compound) | Log position |
| `protocol` | select | | `cctp` / `gateway` |
| `event_type` | select | | `burn` / `mint` / `deposit` / `withdraw` |
| `source_domain` | number | | Origin chain domain ID |
| `destination_domain` | number | | Destination chain domain ID |
| `sender` | text | | Source address |
| `recipient` | text | | Destination address |
| `amount_usdc` | text | | Amount bridged |
| `nonce_val` | text | | CCTP nonce |

**CCTP events indexed**:
- `DepositForBurn` (outbound burn)
- `MintAndWithdraw` (inbound mint)
- `MessageReceived` (attestation confirmed)

**Gateway events indexed**:
- `Deposited` (inbound deposit)
- `Burned` (outbound burn)

**Domain ID 26** = Arc chain. Inbound = `destination_domain = 26`. Outbound = `source_domain = 26 && destination_domain != 26`.

**What it enables**: Bridge volume charts, net flow (in vs out), domain heatmaps, bridging latency, per-chain TVL impact.

---

### 2.8 `fx_swaps` — StableFX RFQ trade lifecycle

| Field | Type | Indexed | Description |
|---|---|---|---|
| `trade_id` | text | ✅ unique | uint256 trade identifier |
| `quote_id` | text | | bytes32 from TradeRecorded |
| `maker` | text | ✅ | Maker address |
| `taker` | text | ✅ | Taker address |
| `taker_fee` | text | | Fee paid by taker (raw wei) |
| `maker_fee` | text | | Fee paid by maker (raw wei) |
| `status_code` | number | | Raw uint8 from TradeStatusChanged |
| `status` | select | | `created` / `taker_funded` / `maker_funded` / `settled` / `cancelled` |
| `block_number` | number | ✅ | Block height |
| `tx_hash` | text | | Tx of TradeRecorded |

**Events upserted** into same row: `TradeRecorded` → `MakerFunded` → `TakerFunded` → `TradeStatusChanged` → `FeesProcessed`.

**What it enables**: FX volume, maker vs taker analysis, settlement time, fee revenue, trade status funnel, order book reconstruction.

---

### 2.9 `agents` — ERC-8004 AI agent registrations

| Field | Type | Indexed | Description |
|---|---|---|---|
| `agent_address` | text | ✅ unique | Agent on-chain address |
| `metadata_uri` | text | | Off-chain metadata pointer |
| `registered_at_block` | number | | Registration block |
| `tx_hash` | text | | Registration tx hash |
| `tx_count` | number | | Aggregated: tx count |
| `usdc_spent_fees` | text | | Aggregated: total fees in USDC |
| `usdc_transferred` | text | | Aggregated: total USDC moved |

**What it enables**: Agent leaderboard, fee spenders, most active agents, registration rate, agent economy size.

---

### 2.10 `agent_jobs` — ERC-8183 job lifecycle

| Field | Type | Indexed | Description |
|---|---|---|---|
| `job_id` | text | ✅ unique | Job identifier |
| `employer_address` | text | ✅ | Who posted the job |
| `worker_address` | text | ✅ | Agent doing the work |
| `payment_usdc` | text | | Escrowed payment |
| `status` | select | | `created` / `accepted` / `delivered` / `settled` / `disputed` |
| `created_at_block` | number | | Block when created |
| `settled_at_block` | number | | Block when settled |
| `create_tx_hash` | text | | Tx hash of job creation |
| `settle_tx_hash` | text | | Tx hash of settlement |

**What it enables**: Job marketplace stats, settlement rate, dispute rate, average job value, top employers/workers, time-to-settle.

---

### 2.11 `block_stats` — Pre-aggregated per-block metrics

| Field | Type | Indexed | Description |
|---|---|---|---|
| `block_number` | number | ✅ unique | Block height |
| `timestamp` | number | | Unix epoch |
| `tps` | number | | Transactions per second |
| `tx_count` | number | | Transaction count |
| `failed_tx_count` | number | | Failed transaction count |
| `avg_fee_usdc` | text | | Average fee in USDC |
| `total_fee_usdc` | text | | Total fees in USDC |
| `total_usdc_transferred` | text | | Total USDC transferred |
| `total_eurc_transferred` | text | | Total EURC transferred |
| `total_usyc_transferred` | text | | Total USYC transferred |
| `unique_senders` | number | | Unique sender addresses |
| `unique_receivers` | number | | Unique receiver addresses |
| `new_contracts` | number | | Contracts deployed |
| `largest_usdc_transfer` | text | | Biggest single USDC transfer |
| `utilization_pct` | number | | Gas utilization percentage |
| `block_time_ms` | number | | Block time in milliseconds |

**What it enables**: This is the **primary time-series table**. Drives every chart: TPS, fee trends, transfer volume curves, utilization sparklines, block time stability.

---

### 2.12 `wallet_edges` — Directional wallet graph

| Field | Type | Indexed | Description |
|---|---|---|---|
| `from_wallet` | text | ✅ (compound) | Source wallet |
| `to_wallet` | text | ✅ (compound) | Destination wallet |
| `total_usdc` | text | | Cumulative USDC sent along this edge |
| `tx_count` | number | | Number of transactions |
| `last_seen_block` | number | | Most recent activity |
| `from_is_agent` | bool | | Source is an ERC-8004 agent |
| `to_is_agent` | bool | | Destination is an ERC-8004 agent |

**What it enables**: 3D force-directed graph, money flow visualization, agent-to-wallet flows, network centrality, cluster detection.

---

## 3. API Endpoints

All endpoints live under `/api/v1/`. Every endpoint supports `limit` (max 500) and `offset` for pagination.

### 3.1 Live Stats

| Method | Path | Handler | Returns |
|---|---|---|---|
| `GET` | `/stats` | `statsHandler` | Latest `block_stats` row + rolling 10-block TPS/block_time + indexer cursor |

**Response fields**: `block_number`, `tps`, `tx_count`, `failed_tx_count`, `block_time_ms`, `avg_fee_usdc`, `total_fee_usdc`, `total_usdc_transferred`, `total_eurc_transferred`, `total_usyc_transferred`, `unique_senders`, `unique_receivers`, `new_contracts`, `largest_usdc_transfer`, `utilization_pct`, `indexed_block`, `syncing`.

### 3.2 Block Stats History

| Method | Path | Handler | Returns |
|---|---|---|---|
| `GET` | `/block_stats` | `blockStatsHandler` | Array of `block_stats` records, sorted newest first |

**Params**: `limit`, `offset`.

### 3.3 Chain Data

| Method | Path | Handler | Filters |
|---|---|---|---|
| `GET` | `/blocks` | `blocksHandler` | `limit`, `offset` |
| `GET` | `/transactions` | `transactionsHandler` | `block`, `from`, `to`, `limit`, `offset` |
| `GET` | `/traces` | `tracesHandler` | `tx`, `from`, `to`, `limit`, `offset` |

### 3.4 Token Transfers

| Method | Path | Handler | Filters |
|---|---|---|---|
| `GET` | `/transfers` | `transfersHandler` | `block`, `token`, `from`, `to`, `limit`, `offset` |

### 3.5 Wallet Profile

| Method | Path | Handler | Returns |
|---|---|---|---|
| `GET` | `/wallet/{address}` | `walletHandler` | Complete wallet profile |

**Returns**: `address`, `is_agent`, `agent` (agent record or null), `txs_sent`, `txs_received`, `sent` (transfers), `received` (transfers), `outgoing_edges`, `incoming_edges`.

### 3.6 Cross-Chain

| Method | Path | Handler | Filters |
|---|---|---|---|
| `GET` | `/crosschain` | `crosschainHandler` | `protocol`, `event_type`, `sender`, `recipient`, `direction` (`inbound`/`outbound`), `limit`, `offset` |

### 3.7 StableFX

| Method | Path | Handler | Filters |
|---|---|---|---|
| `GET` | `/fx` | `fxHandler` | `status`, `maker`, `taker`, `quote_id`, `limit`, `offset` |

### 3.8 Agent Economy

| Method | Path | Handler | Filters |
|---|---|---|---|
| `GET` | `/agents` | `agentsHandler` | `limit`, `offset` |
| `GET` | `/agents/{address}` | `agentHandler` | Agent + their jobs |
| `GET` | `/jobs` | `agentJobsHandler` | `status`, `employer`, `worker`, `limit`, `offset` |

### 3.9 Graph Edges

| Method | Path | Handler | Filters |
|---|---|---|---|
| `GET` | `/edges` | `edgesHandler` | `wallet`, `limit`, `offset` |

---

## 4. Frontend Data Layer

### 4.1 API Clients (`src/lib/api/`)

Each domain has a `types.ts` (TypeScript interfaces) and `crud.ts` (fetch wrapper):

| Module | Type File | Client Class | Endpoints Hit |
|---|---|---|---|
| `stats/` | `StatsResponse` | `StatsCrudClient` | `/stats` |
| `block_stats/` | `BlockStat`, `BlockStatsResponse` | `BlockStatsCrudClient` | `/block_stats` |
| `chain/` | `Block`, `Transaction`, `Trace` + filters/responses | `ChainCrudClient` | `/blocks`, `/transactions`, `/traces` |
| `transfers/` | `Transfer`, `TransferFilter` | `TransfersCrudClient` | `/transfers` |
| `wallet/` | `WalletResponse`, `WalletEdge`, `AgentRecord` | `WalletCrudClient` | `/wallet/{address}` |
| `crosschain/` | `CrosschainEvent`, `CrosschainFilter` | `CrosschainCrudClient` | `/crosschain` |
| `fx/` | `FxTrade`, `FxFilter` | `FxCrudClient` | `/fx` |
| `agents/` | `Agent`, `AgentJob`, filters/responses | `AgentsCrudClient` | `/agents`, `/agents/{addr}`, `/jobs` |
| `graph/` | `Edge`, `EdgeFilter` | `GraphCrudClient` | `/edges` |
| `auth/` | `LoginRequest`, `AuthUser`, etc. | `AuthCrudClient` | PocketBase auth |

### 4.2 Svelte 5 Stores (`src/lib/stores/`)

All stores use `$state` runes. Each exports a reactive state object + a `fetch*()` action:

| Store | State Interface | Data Type |
|---|---|---|
| `stats.svelte.ts` | `StatsState` | `StatsResponse` |
| `blockStats.svelte.ts` | `BlockStatsState` | `BlockStatsResponse` |
| `chain.svelte.ts` | `BlocksState`, `TransactionsState`, `TracesState` | `BlocksResponse`, `TransactionsResponse`, `TracesResponse` |
| `transfers.svelte.ts` | `TransfersState` | `TransfersResponse` |
| `wallet.svelte.ts` | `WalletState` | `WalletResponse` |
| `crosschain.svelte.ts` | `CrosschainState` | `CrosschainResponse` |
| `fx.svelte.ts` | `FxState` | `FxResponse` |
| `agents.svelte.ts` | `AgentsState`, `AgentState`, `AgentJobsState` | `AgentsResponse`, `AgentResponse`, `AgentJobsResponse` |
| `graph.svelte.ts` | `GraphState` | `EdgesResponse` |
| `auth.svelte.ts` | `AuthState` | User info |
| `config.svelte.ts` | — | `getApiUrl()`, PocketBase singleton |

### 4.3 Frontend Routes (`src/routes/`)

| Route | Page File | Status | Currently Shows |
|---|---|---|---|
| `/overview/` | `overview/+page.svelte` | 🟡 Placeholder | Static stat cards, chart placeholders, "loading…" |
| `/blocks/` | `blocks/+page.svelte` | 🟡 Placeholder | Table header only, "loading…" |
| `/txs/` | `txs/+page.svelte` | 🟡 Placeholder | Table header only, filter chips |
| `/transfers/` | `transfers/+page.svelte` | 🟡 Placeholder | Table header, token filter chips |
| `/traces/` | `traces/+page.svelte` | 🟡 Placeholder | Table header only |
| `/crosschain/` | `crosschain/+page.svelte` | 🟡 Placeholder | Mint/burn stat cards, table header |
| `/fx/` | `fx/+page.svelte` | 🟡 Placeholder | Table header, pair filter chips |
| `/agents/` | `agents/+page.svelte` | 🟡 Placeholder | Stat cards, filter chips |
| `/jobs/` | `jobs/+page.svelte` | 🟡 Placeholder | Tabs (Open/In progress/Completed/Failed), table header |
| `/graph/` | `graph/+page.svelte` | 🟡 Placeholder | "3D graph coming soon" |
| `/debug/` | `debug/+page.svelte` | ✅ Working | Raw JSON for every endpoint with all filters |

**All pages have their shell/metadata wired but don't yet fetch or render real data** (except `/debug/` which is a raw JSON playground).

---

## 5. Metrics & Visualizations — What to Show & How

### 5.1 Overview Dashboard (`/overview/`)

The money page. Every card auto-refreshes.

| Card / Widget | Data Source | Metric | Display |
|---|---|---|---|
| **TPS** | `GET /stats` → `tps` | Rolling 10-block average | Big number + sparkline from `GET /block_stats?limit=60` |
| **Block time** | `GET /stats` → `block_time_ms` | Rolling 10-block average | `XXms` + mini bar chart |
| **Transfers 24h** | `GET /block_stats?limit=86400` → sum `tx_count` | Total transfers in 24h | Formatted number |
| **Fees paid 24h** | `GET /block_stats?limit=86400` → sum `total_fee_usdc` | Total fees | `$X,XXX.XX` |
| **FX notional 24h** | `GET /fx?limit=500` (client-side filter) | Sum of settled trade notional | `$X,XXX,XXX` |
| **Active agents** | `GET /agents?limit=500` | Count of agents with `tx_count > 0` recently | Integer |
| **Throughput chart** | `GET /block_stats?limit=60` | `tps` + `total_usdc_transferred` over time | Dual-axis line chart (Chart.js / uPlot) |
| **Cross-chain pulse** | `GET /crosschain?limit=100` | Recent mints vs burns | Stacked bar (inbound vs outbound by domain) |
| **Latest blocks** | `GET /blocks?limit=10` | Recent block feed | Mini table: block #, age, txs, fees |
| **Latest transactions** | `GET /transactions?limit=10` | Recent tx feed | Mini table: hash, from→to, value, age |
| **Top agents** | `GET /agents?limit=10` | Sorted by `usdc_transferred` desc | Agent leaderboard |
| **StableFX live** | `GET /fx?limit=10` | Most recent trades | Mini table: pair, size, maker, status |
| **Indexer lag** | `GET /stats` → `indexed_block` vs `block_number` | `block_number - indexed_block` | Pill in status bar: `🔴 lag: N blocks` or `🟢 synced` |

### 5.2 Blocks Page (`/blocks/`)

| Element | Data | Display |
|---|---|---|
| **Block table** | `GET /blocks?limit=50&offset=N` | Columns: block #, age, txs, miner, gas util%, fees |
| **Age column** | `timestamp` → relative time | "12s ago", "3m ago" |
| **Gas utilization bar** | `utilization_pct` | Inline progress bar (color-coded green/yellow/red) |
| **Block time delta** | `block_time_ms` | "420ms" pill |
| **Pagination** | `offset` + `limit` | Prev/Next buttons |

**Additional metrics per block (expandable row)**:
- `base_fee_per_gas` — formatted as USDC
- `size` — bytes, formatted
- `parent_hash` — link to parent block

### 5.3 Transactions Page (`/txs/`)

| Element | Data | Display |
|---|---|---|
| **Tx table** | `GET /transactions` with filters | Columns: hash, type, from, →, to, value, fee, status, age |
| **Method badge** | `sighash` decoded | "transfer()", "approve()", "swap()", etc. |
| **Status indicator** | `status` (0/1) | ✅ green / ❌ red pill |
| **Fee breakdown** | `fee_usdc`, `priority_fee_usdc` | Tooltip on hover |
| **Type filter chips** | Filter by `sighash` or `tx_type` | "all", "transfer", "swap", "contract deploy" |
| **Address links** | `from_addr`, `to_addr` | Clickable → wallet profile |

**Per-tx detail (expandable/click-through)**:
- Full gas breakdown: `gas_price`, `effective_gas_price`, `max_fee_per_gas`, `max_priority_fee_per_gas`
- `nonce`, `transaction_index`
- `contract_address` if deploy
- Related `transfers` via `GET /transfers?block=N`
- Related `traces` via `GET /traces?tx={hash}`

### 5.4 Transfers Page (`/transfers/`)

| Element | Data | Display |
|---|---|---|
| **Transfer table** | `GET /transfers` with filters | Columns: tx, token, from, →, to, amount, age |
| **Token filter chips** | `token_symbol` filter | "all", "USDC", "EURC", "USYC" |
| **Amount formatting** | `amount_human` | `$1,234,567.89` for USDC |
| **Token badge** | `token_symbol` | Color-coded pill (blue=USDC, green=EURC, orange=USYC) |
| **Address links** | `from_addr`, `to_addr` | Clickable → wallet profile |

**Additional analytics widgets (top of page)**:
- 24h transfer volume by token (stacked area chart from `block_stats`)
- Top transfers (whale alerts): `largest_usdc_transfer` from stats
- Transfer count trend (sparkline)

### 5.5 Traces Page (`/traces/`)

| Element | Data | Display |
|---|---|---|
| **Trace table** | `GET /traces` with filters | Columns: tx, type, from, to, value, gas, error, age |
| **Call type badge** | `call_type` | Color-coded: CALL (blue), DELEGATECALL (orange), STATICCALL (gray) |
| **Error highlight** | `error_msg` | Red row if error present |
| **Filter** | `tx`, `from`, `to` | Input fields |

### 5.6 Cross-Chain Page (`/crosschain/`)

| Element | Data | Display |
|---|---|---|
| **Mints 24h stat** | `GET /crosschain?event_type=mint&limit=500` | Count + total amount |
| **Burns 24h stat** | `GET /crosschain?event_type=burn&limit=500` | Count + total amount |
| **Event table** | `GET /crosschain` with filters | Columns: from chain, →, to chain, token, amount, status, age |
| **Direction filter** | `direction` param | "all", "inbound", "outbound" tabs |
| **Protocol filter** | `protocol` param | "CCTP", "Gateway" chips |
| **Chain name mapping** | `source_domain` / `destination_domain` | Domain ID → chain name (0=Ethereum, 1=Avalanche, 2=Optimism, etc.) |
| **Sankey diagram** | Aggregate crosschain volumes | Domain → Arc → Domain flow |

**Domain mapping** (common CCTP domains):
| Domain | Chain |
|---|---|
| 0 | Ethereum |
| 1 | Avalanche |
| 2 | Optimism |
| 6 | Polygon |
| 10 | Arbitrum |
| 12 | Solana (SPL) |
| 23 | Base |
| 26 | **Arc** |

### 5.7 StableFX Page (`/fx/`)

| Element | Data | Display |
|---|---|---|
| **Trade table** | `GET /fx` with filters | Columns: pair, size, price, maker, status, age |
| **Status badges** | `status` field | Color-coded: created (gray), funded (yellow), settled (green), cancelled (red) |
| **Pair filter chips** | Client-side filter | "all pairs", "USDC/EURC", "USDC/BRZ", "USDC/MXNe" |
| **Status filter** | `status` param | Dropdown |

**Analytics widgets**:
- 24h FX volume (sum of settled trades)
- Trade funnel: created → funded → settled → cancelled (funnel chart)
- Average settlement time (from `block_number` of TradeRecorded to TradeStatusChanged)
- Top makers by volume
- Fee revenue chart (sum of `taker_fee` + `maker_fee`)

### 5.8 Agent Registry Page (`/agents/`)

| Element | Data | Display |
|---|---|---|
| **Stats cards** | Aggregated from `GET /agents` | Total registered, active 24h, jobs in-flight, avg activity |
| **Agent table** | `GET /agents?limit=50` | Columns: address, metadata, registered block, tx count, USDC transferred, fees |
| **Filter chips** | Client-side filter | "all", "active", "market_making", "settlement", "compliance" |
| **Agent card/row** | Per-agent data | Address (truncated), tx_count badge, usdc_transferred bar, fee pie |

**Agent detail (click-through to `/agents/[address]`)**:
- Full profile from `GET /agents/{address}`
- Job history table from `jobs` array
- Transaction history (via wallet endpoint)
- Graph edges (connected wallets)

### 5.9 Job Market Page (`/jobs/`)

| Element | Data | Display |
|---|---|---|
| **Status tabs** | `GET /jobs?status=X` | Open (created), In progress (accepted), Completed (settled), Failed (disputed) |
| **Job table** | `GET /jobs` with filters | Columns: job_id, agent, kind, status, reward (USDC), posted |
| **Reward column** | `payment_usdc` | Formatted USDC amount |
| **Status count badges** | Per-status counts | In each tab |

**Analytics**:
- Total escrowed value (sum of open `payment_usdc`)
- Average job value
- Settlement rate (settled / total)
- Dispute rate (disputed / total)
- Time to settle (block delta)

### 5.10 Wallet Graph Page (`/graph/`)

| Element | Data | Display |
|---|---|---|
| **3D force graph** | `GET /edges?limit=500` | Nodes = wallets, edges = USDC flow, edge width = `tx_count` |
| **Node size** | Total volume through node | Bigger node = more USDC moved |
| **Node color** | Agent vs non-agent | `from_is_agent` / `to_is_agent` = special color |
| **Edge label** | `total_usdc` | Tooltip with amount |
| **Wallet filter** | `GET /edges?wallet=X` | Focus on single wallet's neighborhood |
| **Reset / Export** | Controls | Reset zoom, export as PNG |

### 5.11 Wallet Profile (search → `/wallet/{address}`)

| Element | Data | Display |
|---|---|---|
| **Identity card** | `GET /wallet/{address}` | Address, is_agent badge, agent metadata |
| **Balance estimate** | `received - sent` (client-side sum) | Approximate USDC balance |
| **Sent transfers** | `sent` array | Table of outgoing transfers |
| **Received transfers** | `received` array | Table of incoming transfers |
| **Sent transactions** | `txs_sent` array | Table of sent transactions |
| **Received transactions** | `txs_received` array | Table of received transactions |
| **Outgoing edges** | `outgoing_edges` | Who this wallet sends to |
| **Incoming edges** | `incoming_edges` | Who sends to this wallet |
| **Agent badge** | `is_agent` + `agent` data | If agent: show registration, tx_count, fees |

---

## 6. Derived / Computed Metrics (Not Yet Stored)

These can be computed client-side or added as new API endpoints:

| Metric | Source | Computation |
|---|---|---|
| **USDC balance (approx)** | `wallet/{addr}` | Sum `received.amount_human` − Sum `sent.amount_human` |
| **Whale transfer alerts** | `GET /transfers` | Filter `amount_human > threshold` |
| **TPS percentile** | `GET /block_stats` | Rank current TPS vs historical distribution |
| **Fee percentiles** | `GET /block_stats` | P25, P50, P75, P95 of `avg_fee_usdc` |
| **Cross-chain net flow** | `GET /crosschain` | Inbound USD − Outbound USD per time window |
| **Agent economy TVL** | `GET /agents` | Sum of all `usdc_transferred` |
| **Job market total escrow** | `GET /jobs?status=created` | Sum of `payment_usdc` |
| **Avg settlement time** | `GET /jobs` | Mean of `settled_at_block - created_at_block` |
| **FX maker concentration** | `GET /fx` | Group by `maker`, compute market share |
| **Network centrality** | `GET /edges` | PageRank or betweenness centrality on wallet graph |
| **Transfer velocity** | `GET /block_stats` | `total_usdc_transferred / unique_senders` |
| **Block time stability** | `GET /block_stats` | Std dev of `block_time_ms` over last N blocks |
| **Method popularity** | `GET /transactions` | Group by `sighash`, count, show bar chart |
| **Gas price distribution** | `GET /transactions` | Histogram of `effective_gas_price` |
| **Contract deploy rate** | `GET /transactions` | Count `is_contract_deploy=true` per time window |
| **Indexer health score** | `indexer_events` + `indexer_meta` | Lag + error rate + processing throughput |
| **Domain volume breakdown** | `GET /crosschain` | Group by `source_domain`, sum `amount_usdc` |
| **Failed tx ratio** | `GET /block_stats` | `failed_tx_count / tx_count` per block |

---

## 7. Page-by-Page Analytics Blueprint

### Overview — "The Bloomberg Terminal for Arc"

```
┌──────────────────────────────────────────────────────────┐
│  TPS    Block Time    Transfers 24h    Fees 24h          │
│  42.1   380ms         12,847           $1,234.56         │
│  ▁▂▃▅▇█▆▅▃▂           (sparkline)                        │
├──────────────────────────┬───────────────────────────────┤
│  Throughput + Volume     │  Cross-chain Pulse            │
│  [dual-axis line chart]  │  [stacked bar: in vs out]     │
│  x = block, y1=tps      │  grouped by source domain     │
│           y2=usdc_vol    │                               │
├──────────────────────────┬───────────────────────────────┤
│  Latest Blocks 🔴       │  Latest Transactions 🔴       │
│  #48291031  3s  12txs   │  0xabc... → 0xdef  $1,200    │
│  #48291030  1s   4txs   │  0x123... → 0x456  $50       │
│  (mini table, 10 rows)  │  (mini table, 10 rows)        │
├──────────────────────────┬───────────────────────────────┤
│  Top Agents · 24h       │  StableFX · Live Trades       │
│  🤖 0xabc  $2.4M moved  │  USDC/EURC  $50K  settled    │
│  🤖 0xdef  $1.1M moved  │  USDC/BRZ   $30K  created    │
└──────────────────────────┴───────────────────────────────┘
```

### Blocks — Table with Expandable Rows

```
Block       Age      Txs   Gas Used    Gas Limit   Util%    Fees
#48291031   3s ago   12    8,421,000   30,000,000  28.1%    $0.42
#48291030   1s ago    4    2,100,000   30,000,000   7.0%    $0.14
...

[expanded row]
  Hash:        0xabc123...
  Parent:      0xdef456...
  Miner:       0x789...
  Base Fee:    0.000042 USDC
  Size:        24,521 bytes
  Block Time:  380ms
```

### Transactions — Filterable Table

```
Hash          Method      From          →  To            Value      Fee      Status  Age
0xabc123...   transfer()  0x111...      →  0x222...     $1,200.00  $0.002   ✅     5s
0xdef456...   approve()   0x333...      →  0x444...     —          $0.001   ✅     12s
0x789abc...   swap()      0x555...      →  0x666...     —          $0.008   ✅     18s
0xbaaeee...   deploy      0x777...      →  (new)        —          $0.150   ✅     25s

[filter chips: all | transfer | swap | approve | mint | burn | deploy]
[address filter: from=0x... to=0x... block=...]
```

### Transfers — Token Flow Table

```
Tx           Token    From          →  To            Amount          Age
0xabc...     USDC     0x111...      →  0x222...     $1,200,000.00   3s
0xdef...     EURC     0x333...      →  0x444...     €500,000.00     8s

[token chips: all | USDC | EURC | USYC | OTHER]
[top widgets: 24h volume by token chart, whale alert feed]
```

### Cross-Chain — Bridge Flow

```
┌─────────────────┬─────────────────┐
│  Mints 24h      │  Burns 24h      │
│  342 ($4.2M)    │  128 ($1.8M)    │
│  ↘ inbound      │  ↗ outbound     │
└─────────────────┴─────────────────┘

From Chain    →   To Chain    Token    Amount        Status    Age
Ethereum      →   Arc         USDC     $500,000      Minted    2m ago
Arc           →   Base        USDC     $120,000      Burned    5m ago

[direction tabs: all | inbound | outbound]
[protocol chips: all | CCTP | Gateway]
[sankey diagram: domain → Arc → domain volume flow]
```

### StableFX — Trade Book

```
Pair        Size         Price     Maker         Status     Age
USDC/EURC   $50,000.00   0.92      0xabc...      Settled    1m ago
USDC/BRZ    $30,000.00   5.10      0xdef...      Created    3m ago

[pair chips: all | USDC/EURC | USDC/BRZ | USDC/MXNe]
[top widgets: 24h volume, trade funnel, fee revenue]
```

### Agent Registry — Leaderboard

```
Total Registered    Active 24h    Jobs In-Flight    Avg Activity
42                  18            7                  128 txs/day

[filter chips: all | active | idle | market_making | settlement | compliance]

Address         Registered    TXs     USDC Transferred    Fees Spent
0xabc...        Block 100     1,247   $2,400,000          $42.50
0xdef...        Block 450       892   $1,100,000          $28.10
```

### Job Market — Kanban/Table

```
[Open (7)]  [In Progress (3)]  [Completed (142)]  [Failed (2)]

Job ID      Agent        Status      Reward        Posted
#0x1a2b...  0xabc...     created     $500.00       2m ago
#0x3c4d...  0xdef...     settled     $1,200.00     1h ago

[analytics: total escrow, avg job value, settlement rate, dispute rate]
```

### Wallet Graph — 3D Force

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│     ○ ──── ● ──── ○                                    │
│    / \      |                                             │
│   ○   ○    ○ (agent)                                    │
│    \ /      |                                             │
│     ○ ──── ● ──── ○ ──── ○                              │
│                                                         │
│  ● = agent node    ○ = wallet node                      │
│  edge width = tx_count  edge label = total_usdc         │
│                                                         │
│  [wallet search: 0x...]  [reset] [export PNG]           │
└─────────────────────────────────────────────────────────┘
```

---

## 8. Gaps & Opportunities

### Currently Missing (Recommended Next Steps)

| Gap | Impact | Effort |
|---|---|---|
| **No real data fetching in route pages** | Pages are all static shells | Wire stores to `onMount` |
| **No chart library** | Can't render time-series | Add `uplot` or `chart.js` |
| **No address search** | Search bar is non-functional | Parse input → route to `/wallet/{addr}` or `/txs?hash=X` |
| **No tx detail page** | Can't click into a single transaction | Add `/txs/[hash]` route |
| **No block detail page** | Can't click into a single block | Add `/blocks/[number]` route |
| **No agent detail page** | Can't click into agent profile | Add `/agents/[address]` route |
| **No real-time updates** | Data is fetch-once, no WS/polling | Add polling or PocketBase realtime |
| **No domain name mapping** | Cross-chain shows raw domain IDs | Add `domainMap` constant |
| **No sighash decoder** | Method shows as `0xa9059cbb` | Add known sighash lookup table |
| **No value formatting utils** | Raw USDC wei strings shown | Add `formatUSDC()`, `formatAge()` |
| **No 3D graph renderer** | Graph page is empty | Add `3d-force-graph` or `d3-force-3d` |
| **No CSV/PNG export** | Export buttons are non-functional | Add client-side export |

### Potential New Backend Endpoints

| Endpoint | Purpose |
|---|---|
| `GET /api/v1/search?q=X` | Unified search (address, tx hash, block number, agent) |
| `GET /api/v1/analytics/fees?window=1h` | Pre-aggregated fee analytics |
| `GET /api/v1/analytics/volume?token=USDC&window=24h` | Transfer volume aggregates |
| `GET /api/v1/analytics/bridge_flow` | Cross-chain net flow summary |
| `GET /api/v1/analytics/agent_leaderboard` | Top agents by various metrics |
| `GET /api/v1/tx/{hash}` | Single transaction detail with related transfers + traces |
| `GET /api/v1/block/{number}` | Single block detail with transactions + stats |
| `GET /api/v1/health` | Indexer health (lag, error rate, uptime) |

### Potential New Collections

| Collection | Purpose |
|---|---|
| `daily_stats` | Pre-aggregated daily metrics for long-term charts |
| `token_prices` | EURC/USDC, BRZ/USDC exchange rates (for FX price display) |
| `contract_labels` | Human-readable names for known addresses |
| `sighash_map` | Method signatures for known sighashes |

---

## Appendix: Field Reference Quick Card

### USDC Formatting

```
Raw wei (18 decimals):  "4200000000000000000" = 4.2 USDC
ERC-20 amount (6 decimals): "4200000" = 4.20 USDC
amount_human field:          "4.200000"
```

### Domain IDs → Chain Names

```typescript
const DOMAIN_NAMES: Record<number, string> = {
  0: 'Ethereum',
  1: 'Avalanche',
  2: 'Optimism',
  6: 'Polygon',
  10: 'Arbitrum',
  12: 'Solana',
  23: 'Base',
  26: 'Arc',
};
```

### Common Sighashes

```typescript
const SIGHASH_MAP: Record<string, string> = {
  '0xa9059cbb': 'transfer(address,uint256)',
  '0x095ea7b3': 'approve(address,uint256)',
  '0x23b872dd': 'transferFrom(address,address,uint256)',
  '0x38ed1739': 'swapExactTokensForTokens(...)',
  '0x7ff36ab5': 'swapExactETHForTokens(...)',
  '0x3593564c': 'execute(...)',  // Uniswap Universal Router
  '0x5ae401dc': 'multicall(...)',
};
```

### Status Color Mapping

```typescript
const TX_STATUS = { 0: { label: 'Failed', color: 'red' }, 1: { label: 'Success', color: 'green' } };
const FX_STATUS = {
  created:      { color: 'gray' },
  taker_funded: { color: 'yellow' },
  maker_funded: { color: 'yellow' },
  settled:      { color: 'green' },
  cancelled:    { color: 'red' },
};
const JOB_STATUS = {
  created:   { color: 'gray' },
  accepted:  { color: 'blue' },
  delivered: { color: 'yellow' },
  settled:   { color: 'green' },
  disputed:  { color: 'red' },
};
```

---

*Generated for Arcadia v0.4.2-rc1 — Arc L1 blockchain indexer & analytics platform.*
