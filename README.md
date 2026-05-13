# arcadia

A real-time streaming blockchain indexer and UI dashboard for the Arc L1 chain. Built with Go + PocketBase + HyperSync + SvelteKit. Indexes every layer of Arc's onchain activity — blocks, transactions, USDC/EURC/USYC transfers, internal traces, AI agent registrations, job settlements, cross-chain flows, FX swaps, and derived economic metrics — then streams it live to a 3D frontend via PocketBase websockets.

---

## Get Started

```bash
# Clone the repo
git clone https://github.com/magooney-loon/arcadia.git
cd arcadia
```

Install the **pb-cli** toolchain (used to build and manage the PocketBase extension server):

```bash
go install github.com/magooney-loon/pb-ext/cmd/pb-cli@latest
```

Read more about the toolchain and available scripts:  
**https://github.com/magooney-loon/pb-ext/tree/main/pkg/scripts**

Set your Envio API token (get one at https://envio.dev):

```bash
export ENVIO_API_TOKEN=your_token_here
```

Run in dev mode:

```bash
go run ./cmd/server --dev
```

Or build and run:

```bash
go build -o arcadia ./cmd/server
./arcadia serve
```

The server starts on `http://127.0.0.1:8090`. PocketBase admin UI is available at `/_/`.

---

## Project Structure

```
arcadia/
├── cmd/server/           # Server entry point and all backend code
│   ├── main.go           # App bootstrap, flags (--dev, --generate-specs-dir, --validate-specs-dir)
│   ├── config.go         # Arc network constants, contract addresses, event topics, env vars
│   ├── collections.go    # PocketBase collection schema definitions (14 collections)
│   ├── routes.go         # Versioned REST API route registration (v1)
│   ├── handlers.go       # HTTP handler implementations for all API endpoints
│   ├── indexer.go        # HyperSync indexer loop, batch processing, record upserts
│   └── jobs.go           # Background cron jobs (health check, event cleanup)
├── docs/                 # Arc network reference documentation
├── frontend/             # SvelteKit 3D visualizer frontend (separate build)
├── go.mod                # Go module (Go 1.25, PocketBase v0.38, pb-ext, HyperSync)
└── README.md
```

---

## Data Architecture

### What We Index

#### Layer 0 — Indexer Metadata
**`indexer_meta` collection**

Key/value store for indexer cursor state.

| Field | Notes |
|---|---|
| `key` | e.g. `lastBlock` |
| `value` | string value |

**`indexer_events` collection**

Durable log of indexer lifecycle, progress, and error events. Auto-cleaned: records older than 2 hours are deleted every hour.

| Field | Notes |
|---|---|
| `timestamp` | Unix seconds |
| `level` | `debug` / `info` / `warn` / `error` |
| `event` | Short event key (e.g. `batch_done`, `run_error`, `heartbeat`) |
| `message` | Human-readable description |
| `attempt` / `batch` / `block` / `tip` / `lag` | Numeric context fields |
| `duration_ms` / `blocks` / `transactions` / `logs` | Batch metrics |
| `error` | Error string if applicable |

---

#### Layer 1 — Chain Skeleton
**`blocks` collection**

| Field | Source | Why |
|---|---|---|
| `number` | BlockField.NUMBER | Chain height |
| `hash` | BlockField.HASH | Identity |
| `parent_hash` | BlockField.PARENT_HASH | Reorg detection |
| `timestamp` | BlockField.TIMESTAMP | Time-series axis |
| `gas_used` | BlockField.GAS_USED | Activity heat |
| `gas_limit` | BlockField.GAS_LIMIT | Utilization % denominator |
| `base_fee_per_gas` | BlockField.BASE_FEE_PER_GAS (string) | Fee pressure signal |
| `miner` | BlockField.MINER | Validator tracking |
| `size` | BlockField.SIZE | Block weight |
| `tx_count` | derived | Throughput |
| `block_time_ms` | derived: `timestamp[n] - timestamp[n-1]` | Sub-second finality proof |
| `utilization_pct` | derived: `gas_used / gas_limit × 100` | Congestion heatmap |

---

#### Layer 2 — Transactions
**`transactions` collection**

| Field | Source | Why |
|---|---|---|
| `hash` | TransactionField.HASH | Identity |
| `block_number` | TransactionField.BLOCK_NUMBER | Block linkage |
| `transaction_index` | TransactionField.TRANSACTION_INDEX | Ordering |
| `from_addr` | TransactionField.FROM | Sender node |
| `to_addr` | TransactionField.TO | Receiver node |
| `value` | TransactionField.VALUE (string) | Native value transfer |
| `nonce` | TransactionField.NONCE | Sender sequence |
| `sighash` | derived: first 4 bytes of `input` | Method fingerprint |
| `gas_price` | TransactionField.GAS_PRICE (string) | Fee rate |
| `gas_limit` | TransactionField.GAS (gas field) | Gas budget |
| `gas_used` | TransactionField.GAS_USED | Actual gas consumed |
| `cumulative_gas_used` | TransactionField.CUMULATIVE_GAS_USED | Block-level gas accounting |
| `effective_gas_price` | TransactionField.EFFECTIVE_GAS_PRICE (string) | Post-EIP-1559 actual fee rate |
| `max_fee_per_gas` | TransactionField.MAX_FEE_PER_GAS (string) | EIP-1559 cap |
| `max_priority_fee_per_gas` | TransactionField.MAX_PRIORITY_FEE_PER_GAS (string) | EIP-1559 tip cap |
| `priority_fee_per_gas` | derived: `effective - base_fee` (string) | Actual tip |
| `fee_usdc` | derived: `gas_used × effective_gas_price / 1e18` (string) | **Real USD cost — Arc's differentiator** |
| `priority_fee_usdc` | derived: `gas_used × priority_fee_per_gas / 1e18` (string) | Tip in USD |
| `tx_type` | TransactionField.KIND | Tx type (legacy / EIP-1559) |
| `status` | TransactionField.STATUS | Success / failure |
| `contract_address` | TransactionField.CONTRACT_ADDRESS | Non-null = deployment |
| `is_contract_deploy` | derived | Bool flag |

---

#### Layer 3 — Token Transfers (ERC-20)
**`transfers` collection**

Covers: USDC, EURC, USYC, and all other ERC-20 tokens on Arc.

Event topic: `Transfer(address indexed from, address indexed to, uint256 value)`
`0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef`

| Field | Source | Why |
|---|---|---|
| `tx_hash` | LogField.TRANSACTION_HASH | Transaction linkage |
| `block_number` | LogField.BLOCK_NUMBER | Time axis |
| `log_index` | LogField.LOG_INDEX | Dedup key |
| `token_address` | LogField.ADDRESS | Contract identity |
| `token_symbol` | derived: match against known addresses | `USDC` / `EURC` / `USYC` / `OTHER` |
| `from_addr` | LogField.TOPIC1 (last 20 bytes) | Sender wallet node |
| `to_addr` | LogField.TOPIC2 (last 20 bytes) | Receiver wallet node |
| `amount_raw` | LogField.DATA (string) | Raw uint256 |
| `amount_human` | derived: `amount_raw / 1e6` (string) | Human-readable |

> **Decimal note**: The native USDC gas token uses 18 decimals; the ERC-20 interface uses 6 decimals. Always decode ERC-20 Transfer amounts with `/1e6`. Gas fee amounts (`fee_usdc`) use `/1e18`.

Known token addresses (Arc Testnet):

| Symbol | Address | Decimals (ERC-20) |
|---|---|---|
| **USDC** | `0x3600000000000000000000000000000000000000` | 6 |
| **EURC** | `0x89B50855Aa3bE2F677cD6303Cec089B5F319D72a` | 6 |
| **USYC** | `0xe9185F0c5F296Ed1797AaE4238D26CCaBEadb86C` | 6 |

---

#### Layer 4 — Internal Transactions (Traces)
**`traces` collection**

Traces capture contract-to-contract calls invisible in the transaction list.

| Field | Source | Why |
|---|---|---|
| `tx_hash` | TraceField.TRANSACTION_HASH | Parent tx linkage |
| `block_number` | TraceField.BLOCK_NUMBER | Time axis |
| `from_addr` | TraceField.FROM | Caller |
| `to_addr` | TraceField.TO | Callee |
| `value` | TraceField.VALUE (string) | Internal value transfer |
| `call_type` | TraceField.CALL_TYPE | `call` / `delegatecall` / `staticcall` |
| `trace_type` | TraceField.TYPE | `call` / `create` / `suicide` / `reward` |
| `gas_used` | TraceField.GAS_USED | Internal gas cost |
| `error_msg` | TraceField.ERROR | Failure reason |

> Note: `TraceSelection{{}}` is included in `newIndexerQuery` — all internal calls are fetched and stored. The traces collection is populated by the indexer alongside blocks, transactions, and logs.

---

#### Layer 5 — Arc Agent Economy (ERC-8004 + ERC-8183)
**`agents` collection** + **`agent_jobs` collection**

Arc has native onchain AI agent identity (ERC-8004) and a job/escrow system (ERC-8183).

**`agents` — ERC-8004 IdentityRegistry**

Agent registration is detected via ERC-721 `Transfer` mints (topic1 = zero address) from `AddrAgentRegistry`.

| Field | Why |
|---|---|
| `agent_address` | Wallet identity (from topic2) |
| `registered_at_block` | Onboarding timeline |
| `tx_hash` | Registration transaction |
| `tx_count` | Transactions sent by this agent (aggregated each batch) |
| `usdc_spent_fees` | Cumulative gas fees paid, stored as raw wei string |
| `usdc_transferred` | Cumulative USDC sent, stored as raw ERC-20 units string (6 decimals) |

**`agent_jobs` — ERC-8183 AgenticCommerce**

Job lifecycle events from `AddrAgenticCommerce` (proxy `0x0747…`, impl `0xA316…`). Indexes all seven state-changing events. Each record is upserted by `job_id` as events arrive.

| Field | Why |
|---|---|
| `job_id` | uint256 job ID (string) |
| `employer_address` | Client — topic2 of `JobCreated` |
| `worker_address` | Provider — topic3 of `JobCreated` |
| `payment_usdc` | Human-readable escrow amount set by `JobFunded` |
| `status` | `created` → `funded` → `submitted` → `completed`/`rejected`/`expired` → `paid` |
| `created_at_block` | Block of `JobCreated` |
| `settled_at_block` | Block of `PaymentReleased` |
| `create_tx_hash` | Tx of `JobCreated` |
| `settle_tx_hash` | Tx of `PaymentReleased` |

Indexed events (verified from impl ABI):

| Event | Signature | Status transition |
|---|---|---|
| `JobCreated` | `JobCreated(uint256,address,address,address,uint256,address)` | → `created` |
| `JobFunded` | `JobFunded(uint256,address,uint256)` | → `funded`, sets `payment_usdc` |
| `JobSubmitted` | `JobSubmitted(uint256,address,bytes32)` | → `submitted` |
| `JobCompleted` | `JobCompleted(uint256,address,bytes32)` | → `completed` |
| `JobRejected` | `JobRejected(uint256,address,bytes32)` | → `rejected` |
| `PaymentReleased` | `PaymentReleased(uint256,address,uint256)` | → `paid`, sets `settled_at_block` + `settle_tx_hash` |
| `JobExpired` | `JobExpired(uint256)` | → `expired` |

---

#### Layer 6 — Cross-Chain Flows (CCTP + Gateway)
**`crosschain_events` collection**

Shows capital entering/leaving Arc via CCTP and Gateway. Arc is CCTP domain **26**.

**CCTP contracts (Arc Testnet):**

| Contract | Address | Events |
|---|---|---|
| **TokenMessengerV2** | `0x8FE6B999Dc680CcFDD5Bf7EB0974218be2542DAA` | `DepositForBurn`, `MintAndWithdraw` |
| **MessageTransmitterV2** | `0xE737e5cEBEEBa77EFE34D4aa090756590b1CE275` | `MessageReceived` |
| **TokenMinterV2** | `0xb43db544E2c27092c107639Ad201b3dEfAbcF192` | mint/burn execution |
| **CCTPMessage** | `0xbaC0179bB358A8936169a63408C8481D582390C4` | Message routing |

**Gateway contracts (Arc Testnet):**

| Contract | Address | Events |
|---|---|---|
| **GatewayWallet** | `0x0077777d7EBA4688BDeF3E311b846F25870A19B9` | `Deposited`, `GatewayBurned` |
| **GatewayMinter** | `0x0022222ABE238Cc2C7Bb1f21003F0a260052475B` | `AttestationUsed` |

| Field | Why |
|---|---|
| `tx_hash` + `log_index` | Dedup key |
| `protocol` | `cctp` / `gateway` |
| `event_type` | `burn` / `mint` / `deposit` / `withdraw` / `attestation_used` |
| `source_domain` | CCTP domain ID of origin chain |
| `destination_domain` | CCTP domain ID of destination (Arc = 26) |
| `amount_usdc` | Transfer size |
| `sender` | Origin wallet |
| `recipient` | Destination wallet |
| `block_number` | Timeline |
| `nonce_val` | Cross-chain message correlation |

---

#### Layer 6b — FX Settlement (StableFX)
**`fx_swaps` collection**

StableFX is Circle's onchain FX engine on Arc. The `FxEscrow` contract settles USDC↔EURC swaps.

**StableFX contract:**

| Contract | Address |
|---|---|
| **FxEscrow** | `0x867650F5eAe8df91445971f14d89fd84F0C9a9f8` |

Indexed events (verified from implementation ABI):

| Event | Description |
|---|---|
| `TradeRecorded(uint256, bytes32)` | Swap executed on-chain |
| `MakerFunded(uint256, address)` | Liquidity provider funded |
| `TakerFunded(uint256, address)` | Swap initiator funded |
| `TradeStatusChanged(uint256, address, uint8)` | Status transition (created/settled/cancelled) |
| `FeesProcessed(uint256, uint256, uint256)` | Fee breakdown |

| Field | Why |
|---|---|
| `tx_hash` + `log_index` | Dedup key |
| `maker` | Liquidity provider |
| `taker` | Swap initiator |
| `sell_token` | Token sold (USDC or EURC address) |
| `buy_token` | Token received |
| `sell_amount` | Amount in (string) |
| `buy_amount` | Amount out (string) |
| `implied_rate` | `buy_amount / sell_amount` — live USDC/EURC rate |
| `block_number` | Timeline |
| `status` | `created` / `settled` / `cancelled` |

---

#### Layer 7 — Derived Block Stats (Pre-aggregated)
**`block_stats` collection**

Stored at index time so the frontend never does heavy aggregations at query time.

| Field | Formula |
|---|---|
| `block_number` | Block identity |
| `timestamp` | Unix seconds |
| `tx_count` | Transaction count |
| `block_time_ms` | `(timestamp[n] - timestamp[n-1]) × 1000` |
| `tps` | `tx_count / block_time_seconds` |
| `avg_fee_usdc` | `sum(fee_usdc) / tx_count` |
| `total_fee_usdc` | `sum(fee_usdc)` |
| `total_usdc_transferred` | `sum(amount_human) WHERE token = USDC` |
| `total_eurc_transferred` | `sum(amount_human) WHERE token = EURC` |
| `total_usyc_transferred` | `sum(amount_human) WHERE token = USYC` |
| `unique_senders` | `COUNT(DISTINCT from_addr)` |
| `unique_receivers` | `COUNT(DISTINCT to_addr)` |
| `new_contracts` | `COUNT WHERE is_contract_deploy = true` |
| `largest_usdc_transfer` | `MAX(amount_human) WHERE token = USDC` |
| `utilization_pct` | `gas_used / gas_limit × 100` |

---

#### Layer 8 — Wallet Graph (Live Network)
**`wallet_edges` collection**

Each unique (from, to) pair that has ever transferred value = one edge in the wallet graph.

| Field | Why |
|---|---|
| `from_wallet` | Source node |
| `to_wallet` | Target node |
| `total_usdc` | Cumulative raw amount transferred (string uint256) |
| `tx_count` | Edge strength / transfer count |
| `last_seen_block` | Recency for decay animation |
| `from_is_agent` | Highlight agent source nodes |
| `to_is_agent` | Highlight agent destination nodes |

---

## HyperSync Query Structure

```go
query := &types.Query{
    FromBlock:        new(big.Int).SetUint64(fromBlock),
    ToBlock:          new(big.Int).SetUint64(toBlock),
    IncludeAllBlocks: true,
    FieldSelection: types.FieldSelection{
        Block: []string{
            "number", "hash", "parent_hash", "timestamp",
            "gas_used", "gas_limit", "base_fee_per_gas", "miner", "size",
        },
        Transaction: []string{
            "hash", "block_number", "transaction_index",
            "from", "to", "value", "nonce", "input",
            "gas_price", "gas_used", "effective_gas_price",
            "gas", "cumulative_gas_used", "max_fee_per_gas",
            "max_priority_fee_per_gas", "type", "status", "contract_address",
        },
        Log: []string{
            "block_number", "transaction_hash", "log_index",
            "address", "topic0", "topic1", "topic2", "topic3", "data",
        },
        Trace: []string{
            "block_number", "transaction_hash",
            "from", "to", "value", "call_type", "type", "gas_used", "error",
        },
    },
    Transactions: []types.TransactionSelection{{}}, // all transactions
    Traces:       []types.TraceSelection{{}},        // all internal calls
    Logs: []types.LogSelection{
        // All ERC-20 Transfer events (USDC, EURC, USYC, any token)
        {Topics: [][]common.Hash{{TopicTransfer}}},
        // CCTP: DepositForBurn + MintAndWithdraw on TokenMessengerV2
        {
            Address: []common.Address{AddrCCTPTokenMessenger},
            Topics:  [][]common.Hash{{TopicDepositForBurn, TopicMintAndWithdraw}},
        },
        // CCTP: MessageReceived on MessageTransmitterV2
        {
            Address: []common.Address{AddrCCTPMessageTransmitter},
            Topics:  [][]common.Hash{{TopicMessageReceived}},
        },
        // Gateway: Deposited + GatewayBurned on GatewayWallet
        {
            Address: []common.Address{AddrGatewayWallet},
            Topics:  [][]common.Hash{{TopicGatewayDeposited, TopicGatewayBurned}},
        },
        // Gateway: AttestationUsed on GatewayMinter
        {
            Address: []common.Address{AddrGatewayMinter},
            Topics:  [][]common.Hash{{TopicAttestationUsed}},
        },
        // StableFX swap lifecycle
        {Address: []common.Address{AddrFxEscrow}},
        // ERC-8004 agent registration mints (Transfer from zero address)
        {Address: []common.Address{AddrAgentRegistry}, Topics: [][]common.Hash{{TopicAgentRegistered}}},
        // ERC-8183 job lifecycle: all seven state-changing events
        {
            Address: []common.Address{AddrAgenticCommerce},
            Topics: [][]common.Hash{{
                TopicJobCreated, TopicJobFunded, TopicJobSubmitted,
                TopicJobCompleted, TopicJobRejected, TopicPaymentReleased, TopicJobExpired,
            }},
        },
    },
}
```

Batches are 200 blocks wide, paced at 400ms between requests to stay within the Envio free-tier burst limit.

---

## Contract Addresses (Arc Testnet)

```go
// Stablecoins
AddrUSDC = "0x3600000000000000000000000000000000000000" // 6 decimals (ERC-20), 18 decimals (native gas)
AddrEURC = "0x89B50855Aa3bE2F677cD6303Cec089B5F319D72a" // 6 decimals
AddrUSYC = "0xe9185F0c5F296Ed1797AaE4238D26CCaBEadb86C" // 6 decimals

// USYC supporting contracts
AddrUSYCEntitlements = "0xcc205224862c7641930c87679e98999d23c26113"
AddrUSYCTeller       = "0x9fdF14c5B14173D74C08Af27AebFf39240dC105A"

// CCTP v2 (Arc domain = 26)
AddrCCTPTokenMessenger     = "0x8FE6B999Dc680CcFDD5Bf7EB0974218be2542DAA"
AddrCCTPMessageTransmitter = "0xE737e5cEBEEBa77EFE34D4aa090756590b1CE275"
AddrCCTPTokenMinter        = "0xb43db544E2c27092c107639Ad201b3dEfAbcF192"
AddrCCTPMessage            = "0xbaC0179bB358A8936169a63408C8481D582390C4"

// Gateway
AddrGatewayWallet = "0x0077777d7EBA4688BDeF3E311b846F25870A19B9"
AddrGatewayMinter = "0x0022222ABE238Cc2C7Bb1f21003F0a260052475B"

// StableFX
AddrFxEscrow = "0x867650F5eAe8df91445971f14d89fd84F0C9a9f8"

// Common Ethereum contracts deployed on Arc
AddrPermit2        = "0x000000000022D473030F116dDEE9F6B43aC78BA3"
AddrMulticall3     = "0xcA11bde05977b3631167028862bE2a173976CA11"
AddrCreate2Factory = "0x4e59b44847b379578588920cA78FbF26c0B4956C"

// ERC-8004 agent registries
AddrAgentRegistry      = "0x8004A818BFB912233c491871b3d84c89A494BD9e"
AddrReputationRegistry = "0x8004B663056A597Dffe9eCcC1965A193B7388713"
AddrValidationRegistry = "0x8004Cb1BF31DAf7788923b405b754f57acEB4272"

// ERC-8183 AgenticCommerce reference implementation
AddrAgenticCommerce = "0x0747EEf0706327138c69792bF28Cd525089e4583"
```

---

## Event Signatures (Keccak256 Topics)

All topics are verified against on-chain ABIs unless noted.

```go
// ERC-20: Transfer(address indexed from, address indexed to, uint256 value)
TopicTransfer = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

// ERC-20: Approval(address indexed owner, address indexed spender, uint256 value)
TopicApproval = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"

// CCTP v2 — TokenMessengerV2 (verified from impl 0xf07c0ad1)
// DepositForBurn(address,uint256,address,bytes32,uint32,bytes32,bytes32,uint256,uint32,bytes)
TopicDepositForBurn = keccak256("DepositForBurn(address,uint256,address,bytes32,uint32,bytes32,bytes32,uint256,uint32,bytes)")

// CCTP v2 — TokenMessengerV2
// MintAndWithdraw(address,uint256,address,uint256)
TopicMintAndWithdraw = keccak256("MintAndWithdraw(address,uint256,address,uint256)")

// CCTP v2 — MessageTransmitterV2 (verified from impl 0xa849059b)
// MessageReceived(address,uint32,bytes32,bytes32,uint32,bytes)
TopicMessageReceived = keccak256("MessageReceived(address,uint32,bytes32,bytes32,uint32,bytes)")

// GatewayWallet (verified from impl 0x44eeddc9)
// Deposited(address,address,address,uint256)
TopicGatewayDeposited = keccak256("Deposited(address,address,address,uint256)")

// GatewayWallet — outbound bridge (USDC leaving Arc)
// GatewayBurned(address,address,bytes32,uint32,bytes32,address,uint256,uint256,uint256,uint256)
TopicGatewayBurned = keccak256("GatewayBurned(address,address,bytes32,uint32,bytes32,address,uint256,uint256,uint256,uint256)")

// GatewayMinter (verified from impl 0x9ef4c7ad) — inbound bridge (USDC arriving on Arc)
// AttestationUsed(address,address,bytes32,uint32,bytes32,bytes32,uint256)
TopicAttestationUsed = keccak256("AttestationUsed(address,address,bytes32,uint32,bytes32,bytes32,uint256)")

// ERC-8004 — agent registration = ERC-721 mint from AddrAgentRegistry (reuses TopicTransfer, topic1 = zero)
TopicAgentRegistered = TopicTransfer

// ERC-8183 — AgenticCommerce (impl 0xA316fd02827242D537F84730F8a37D0BA5fd351a)
// JobCreated(uint256 indexed jobId, address indexed client, address indexed provider, address evaluator, uint256 expiredAt, address hook)
TopicJobCreated = keccak256("JobCreated(uint256,address,address,address,uint256,address)")
// JobFunded(uint256 indexed jobId, address indexed client, uint256 amount)
TopicJobFunded = keccak256("JobFunded(uint256,address,uint256)")
// JobSubmitted(uint256 indexed jobId, address indexed provider, bytes32 deliverable)
TopicJobSubmitted = keccak256("JobSubmitted(uint256,address,bytes32)")
// JobCompleted(uint256 indexed jobId, address indexed evaluator, bytes32 reason)
TopicJobCompleted = keccak256("JobCompleted(uint256,address,bytes32)")
// JobRejected(uint256 indexed jobId, address indexed rejector, bytes32 reason)
TopicJobRejected = keccak256("JobRejected(uint256,address,bytes32)")
// PaymentReleased(uint256 indexed jobId, address indexed provider, uint256 amount)
TopicPaymentReleased = keccak256("PaymentReleased(uint256,address,uint256)")
// JobExpired(uint256 indexed jobId)
TopicJobExpired = keccak256("JobExpired(uint256)")

// FxEscrow (StableFX) — verified from implementation ABI
TopicTradeRecorded      = keccak256("TradeRecorded(uint256,bytes32)")
TopicMakerFunded        = keccak256("MakerFunded(uint256,address)")
TopicTakerFunded        = keccak256("TakerFunded(uint256,address)")
TopicTradeStatusChanged = keccak256("TradeStatusChanged(uint256,address,uint8)")
TopicFeesProcessed      = keccak256("FeesProcessed(uint256,uint256,uint256)")
```

---

## REST API

Versioned under `/api/v1`. Default pagination: `limit=50&offset=0`, max `limit=500`. OpenAPI spec auto-generated at startup (public Swagger UI enabled).

### Core Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/stats` | Latest block stats + rolling 10-block avg TPS |
| `GET` | `/api/v1/block_stats` | Historical block stats for time-series charts — `?window=1h\|6h\|24h\|7d\|30d` |
| `GET` | `/api/v1/blocks` | Recent blocks — `?limit=N` |
| `GET` | `/api/v1/block/{number}` | Single block detail |
| `GET` | `/api/v1/transactions` | Recent transactions — `?block=N`, `?from=addr` |
| `GET` | `/api/v1/tx/{hash}` | Single transaction detail |
| `GET` | `/api/v1/traces` | Internal traces — `?tx_hash=hash` |
| `GET` | `/api/v1/transfers` | Token transfers — `?token=USDC\|EURC\|USYC`, `?from=addr`, `?to=addr` |
| `GET` | `/api/v1/wallet/{address}` | Wallet profile: sent/received transfers + graph edges + agent status |
| `GET` | `/api/v1/crosschain` | CCTP + Gateway events — `?protocol=cctp\|gateway` |
| `GET` | `/api/v1/fx` | StableFX USDC↔EURC swap events |
| `GET` | `/api/v1/agents` | Registered ERC-8004 AI agents |
| `GET` | `/api/v1/agents/{address}` | Single agent profile + job history |
| `GET` | `/api/v1/jobs` | Agent job marketplace — `?status=created\|settled\|…` |
| `GET` | `/api/v1/edges` | Wallet graph edges — `?wallet=addr` |
| `GET` | `/api/v1/health` | Indexer health: last block, chain tip, lag, sync status, error count, avg batch time |
| `GET` | `/api/v1/search` | Unified search — `?q=txHash\|address\|blockNumber` |

### Analytics Endpoints

Pre-aggregated, window-scoped analytics for dashboards.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/analytics/fees` | Fee analytics — `?window=1h\|6h\|24h\|7d\|30d` |
| `GET` | `/api/v1/analytics/volume` | Transfer volume per token — `?window=…`, whale threshold $10k+ |
| `GET` | `/api/v1/analytics/bridge_flow` | CCTP + Gateway inflow/outflow per chain — `?window=…` |
| `GET` | `/api/v1/analytics/agent_leaderboard` | Top agents by job count, fees, volume — `?window=…` |

Everything is also available via PocketBase REST + real-time websockets directly on the collections.

---

## Background Jobs

| Job | Schedule | Description |
|---|---|---|
| `indexerHealth` | Every hour (`0 * * * *`) | Logs indexer cursor and row counts for all collections |
| `indexerEventsCleanup` | Every hour (`0 * * * *`) | Deletes `indexer_events` records older than 2 hours |

---

## What the Frontend Gets

**3D visualization layers:**
1. **Chain spine** — blocks as nodes, time on the Z axis, utilization as heat color
2. **Transaction particles** — particles flying between wallet nodes, sized by value
3. **USDC blood flow** — animated edges between wallets, thickness = transfer amount
4. **EURC layer** — separate color channel alongside USDC (FX flows)
5. **USYC layer** — yield-bearing capital shown as a distinct particle type
6. **Agent network** — agent wallets highlighted, agent-to-agent edges glowing
7. **Cross-chain arrows** — CCTP and Gateway inflows rendered as external capital entering the Arc sphere
8. **FX swap arcs** — StableFX USDC↔EURC swaps as bidirectional currency exchange events
9. **Fee heatmap** — per-block USDC fee burn (in real dollars), proving Arc's $0.01 tx cost claim
10. **Job economy** — ERC-8183 job arcs between employer and worker agent wallets

**Time-series charts (from `block_stats` + analytics endpoints):**
- TPS over time
- Avg USDC fee per tx over time (real USD, unique to Arc)
- Total USDC / EURC / USYC transferred per block
- StableFX swap volume and implied USDC/EURC rate over time
- Active wallets per block
- Block time distribution (sub-second finality histogram)
- Cross-chain inflow/outflow volume per chain (CCTP + Gateway)
- New contract deployments over time
- Agent job settlement rate
- Agent leaderboard (top agents by activity)

---

## Stack

| Component | Technology |
|---|---|
| **Language** | Go 1.25 |
| **Indexer** | `github.com/enviodev/hypersync-client-go` |
| **Server framework** | `github.com/magooney-loon/pb-ext` (PocketBase extension with versioned API, job manager, OpenAPI spec gen) |
| **Database + API + Websockets** | PocketBase v0.38 (SQLite under the hood) |
| **Chain** | Arc Testnet — `https://arc-testnet.hypersync.xyz` |
| **Frontend** | SvelteKit dashboard (`frontend/`) |

---

## Key Arc Testnet References

- **Chain ID**: `5042002`
- HyperSync endpoint: `https://arc-testnet.hypersync.xyz` (also `https://5042002.hypersync.xyz`)
- Public RPC endpoints (round-robin pool, no key required):
  - `https://rpc.testnet.arc.network`
  - `https://rpc.blockdaemon.testnet.arc.network`
  - `https://rpc.drpc.testnet.arc.network`
  - `https://rpc.quicknode.testnet.arc.network`
- WebSocket endpoints:
  - `wss://rpc.testnet.arc.network`
  - `wss://rpc.drpc.testnet.arc.network`
  - `wss://rpc.quicknode.testnet.arc.network`
- Block explorer: `https://testnet.arcscan.app`
- Contract addresses: `https://docs.arc.network/arc/references/contract-addresses.md`
- Faucet (USDC + EURC): `https://faucet.circle.com`
- CCTP domain for Arc: **26**
- ABI lookup: `https://testnet.arcscan.app/address/<contract>` → Contract tab → ABI
- Envio API token: `https://envio.dev` (set as `ENVIO_API_TOKEN` env var)
