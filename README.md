# arcadia

A real-time streaming blockchain indexer and 3D visualizer for the Arc L1 chain. Built with Go + PocketBase + HyperSync. Indexes every layer of Arc's onchain activity — blocks, transactions, USDC/EURC transfers, internal traces, AI agent registrations, job settlements, cross-chain flows, and derived economic metrics — then streams it live to a 3D frontend via PocketBase websockets.

---

## Data Architecture

### What We Index

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
| `base_fee_per_gas` | BlockField.BASE_FEE_PER_GAS | Fee pressure signal |
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
| `transaction_index` | TransactionField.TRANSACTION_INDEX | Ordering / MEV analysis |
| `from` | TransactionField.FROM | Sender node |
| `to` | TransactionField.TO | Receiver node |
| `value` | TransactionField.VALUE | Native value transfer |
| `gas_price` | TransactionField.GAS_PRICE | Fee rate |
| `gas_used` | TransactionField.GAS_USED | Actual gas consumed |
| `effective_gas_price` | TransactionField.EFFECTIVE_GAS_PRICE | Post-EIP-1559 actual fee rate |
| `fee_usdc` | derived: `gas_used × effective_gas_price / 1e18` | **Real USD cost of this tx — Arc's killer differentiator** |
| `nonce` | TransactionField.NONCE | Sender sequence / burst detection |
| `input` | TransactionField.INPUT | Raw calldata |
| `sighash` | derived: first 4 bytes of `input` | Method signature for tx categorization |
| `type` | TransactionField.KIND | Tx type (legacy / EIP-1559) |
| `contract_address` | TransactionField.CONTRACT_ADDRESS | Non-null = contract deployment |
| `status` | TransactionField.STATUS | Success / failure |

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
| `token_symbol` | derived: match against known addresses | USDC / EURC / USYC / other |
| `from` | LogField.TOPIC1 (last 20 bytes) | Sender wallet node |
| `to` | LogField.TOPIC2 (last 20 bytes) | Receiver wallet node |
| `amount_raw` | LogField.DATA | Raw uint256 |
| `amount_human` | derived: `amount_raw / 1e6` | Human-readable (all stablecoins use 6 decimals via ERC-20 interface) |

> **Decimal note**: The native USDC gas token uses 18 decimals; the ERC-20 interface (what Transfer events use) uses 6 decimals. Always decode ERC-20 Transfer amounts with `/1e6`. Gas fee amounts (`fee_usdc`) use `/1e18`.

Known token addresses (Arc Testnet):

| Symbol | Address | Decimals (ERC-20) |
|---|---|---|
| **USDC** | `0x3600000000000000000000000000000000000000` | 6 |
| **EURC** | `0x89B50855Aa3bE2F677cD6303Cec089B5F319D72a` | 6 |
| **USYC** | `0xe9185F0c5F296Ed1797AaE4238D26CCaBEadb86C` | 6 |

---

#### Layer 4 — Internal Transactions (Traces)
**`traces` collection**

Traces capture contract-to-contract calls invisible in the transaction list. Essential once DeFi/agent contracts are deployed on Arc.

| Field | Source | Why |
|---|---|---|
| `transaction_hash` | TraceField.TRANSACTION_HASH | Parent tx linkage |
| `block_number` | TraceField.BLOCK_NUMBER | Time axis |
| `from` | TraceField.FROM | Caller |
| `to` | TraceField.TO | Callee |
| `value` | TraceField.VALUE | Internal value transfer |
| `call_type` | TraceField.CALL_TYPE | call / delegatecall / staticcall |
| `type` | TraceField.TYPE | call / create / suicide / reward |
| `input` | TraceField.INPUT | Calldata |
| `output` | TraceField.OUTPUT | Return data |
| `gas_used` | TraceField.GAS_USED | Internal gas cost |
| `error` | TraceField.ERROR | Failure reason |
| `trace_address` | TraceField.TRACE_ADDRESS | Position in call tree |

---

#### Layer 5 — Arc Agent Economy (ERC-8004 + ERC-8183)
**`agents` collection** + **`jobs` collection**

Arc has native onchain AI agent identity (ERC-8004) and a job/escrow system (ERC-8183). This is the layer no other block explorer tracks.

**Agent Registration events (ERC-8004)**
| Field | Why |
|---|---|
| `agent_address` | Wallet identity |
| `metadata_uri` | Agent name / capabilities |
| `registered_at_block` | Onboarding timeline |
| `tx_count` | Activity metric post-registration |
| `usdc_spent_fees` | Total fees paid by this agent |
| `usdc_transferred` | Total economic throughput |

**Job Lifecycle events (ERC-8183)**
| Field | Why |
|---|---|
| `job_id` | Identity |
| `employer_address` | Who created the job |
| `worker_address` | Which agent took it |
| `payment_usdc` | Escrow amount |
| `state` | created / accepted / delivered / settled / disputed |
| `created_at_block` | Timeline |
| `settled_at_block` | Time-to-settlement metric |

---

#### Layer 6 — Cross-Chain Flows (CCTP + Gateway)
**`crosschain_events` collection**

Shows capital entering/leaving Arc from other chains via CCTP and Gateway. Visualized as inbound/outbound arrows from the broader crypto ecosystem. Arc is CCTP domain **26**.

**CCTP contracts (Arc Testnet):**

| Contract | Address | Events |
|---|---|---|
| **TokenMessengerV2** | `0x8FE6B999Dc680CcFDD5Bf7EB0974218be2542DAA` | `DepositForBurn` (USDC exits a chain) |
| **MessageTransmitterV2** | `0xE737e5cEBEEBa77EFE34D4aa090756590b1CE275` | `MessageReceived` (USDC arrives on Arc) |
| **TokenMinterV2** | `0xb43db544E2c27092c107639Ad201b3dEfAbcF192` | mint/burn execution |

**Gateway contracts (Arc Testnet):**

| Contract | Address | Purpose |
|---|---|---|
| **GatewayWallet** | `0x0077777d7EBA4688BDeF3E311b846F25870A19B9` | User-facing unified balance |
| **GatewayMinter** | `0x0022222ABE238Cc2C7Bb1f21003F0a260052475B` | Cross-chain mint/burn handler |

| Field | Why |
|---|---|
| `event_type` | `cctp_burn` / `cctp_received` / `gateway_deposit` / `gateway_withdraw` |
| `protocol` | `cctp` / `gateway` |
| `source_domain` | CCTP domain ID of origin chain |
| `destination_domain` | CCTP domain ID of destination (Arc = 26) |
| `amount_usdc` | Transfer size |
| `sender` | Origin wallet |
| `recipient` | Destination wallet |
| `block_number` | Timeline |
| `nonce` | Dedup / cross-chain message correlation |

---

#### Layer 6b — FX Settlement (StableFX)
**`fx_swaps` collection**

StableFX is Circle's onchain FX engine on Arc. The `FxEscrow` contract settles USDC↔EURC swaps. Tracking this gives visibility into cross-currency flows — which is a unique signal on Arc that no other chain has.

**StableFX contract:**

| Contract | Address |
|---|---|
| **FxEscrow** | `0x867650F5eAe8df91445971f14d89fd84F0C9a9f8` |

| Field | Why |
|---|---|
| `swap_id` | Identity |
| `maker` | Liquidity provider |
| `taker` | Swap initiator |
| `sell_token` | Token sold (USDC or EURC address) |
| `buy_token` | Token received |
| `sell_amount` | Amount in |
| `buy_amount` | Amount out |
| `implied_rate` | `buy_amount / sell_amount` — live USDC/EURC rate |
| `block_number` | Timeline |
| `status` | `created` / `settled` / `cancelled` |

> Event signatures for FxEscrow: verify against the ABI via `testnet.arcscan.app/address/0x867650F5eAe8df91445971f14d89fd84F0C9a9f8`

---

#### Layer 7 — Derived Block Stats (Pre-aggregated)
**`block_stats` collection**

Stored at index time so the frontend never does heavy aggregations at query time.

| Metric | Formula |
|---|---|
| `tps` | `tx_count / block_time_seconds` |
| `avg_fee_usdc` | `sum(fee_usdc) / tx_count` |
| `total_usdc_transferred` | `sum(amount_human) FROM transfers WHERE block = N AND token_address = USDCAddress` |
| `total_eurc_transferred` | same, filtered to `EURCAddress` |
| `total_usyc_transferred` | same, filtered to `USYCAddress` — yield-bearing capital flows |
| `fx_swap_volume_usdc` | `sum(sell_amount) FROM fx_swaps WHERE block = N` |
| `unique_senders` | `COUNT(DISTINCT from) FROM transactions WHERE block = N` |
| `unique_receivers` | `COUNT(DISTINCT to)` |
| `new_contracts_deployed` | `COUNT WHERE contract_address IS NOT NULL` |
| `failed_tx_count` | `COUNT WHERE status = 0` |
| `success_rate_pct` | `(tx_count - failed_tx_count) / tx_count × 100` |
| `largest_usdc_transfer` | `MAX(amount_usdc)` |
| `agent_txs` | `COUNT WHERE from IN (agent_addresses)` |

---

#### Layer 8 — Wallet Graph (Live Network)
**`wallet_edges` collection**

Each unique (from, to) pair that has ever transferred value = one edge in the wallet graph. Used to render the 3D network visualization.

| Field | Why |
|---|---|
| `from_wallet` | Source node |
| `to_wallet` | Target node |
| `total_usdc_transferred` | Edge weight |
| `tx_count` | Edge strength |
| `last_seen_block` | Recency for decay animation |
| `is_agent` | Highlight agent-to-agent flows |

---

## HyperSync Query Structure

```go
query := hypersync.Query{
    FromBlock: lastBlock,
    IncludeAllBlocks: true,
    FieldSelection: hypersync.FieldSelection{
        Block: []string{
            "number", "hash", "parent_hash", "timestamp",
            "gas_used", "gas_limit", "base_fee_per_gas", "miner", "size",
        },
        Transaction: []string{
            "hash", "block_number", "transaction_index",
            "from", "to", "value", "nonce", "input",
            "gas_price", "gas_used", "effective_gas_price",
            "max_fee_per_gas", "max_priority_fee_per_gas",
            "type", "status", "contract_address",
        },
        Log: []string{
            "block_number", "transaction_hash", "log_index",
            "address", "topic0", "topic1", "topic2", "topic3", "data",
        },
        Trace: []string{
            "transaction_hash", "block_number", "from", "to",
            "value", "call_type", "type", "input", "output",
            "gas_used", "error", "trace_address",
        },
    },
    Logs: []hypersync.LogSelection{
        // All ERC-20 transfers (USDC, EURC, USYC, any token)
        {Topics: [][]string{{TransferEventTopic}}},
        // CCTP DepositForBurn — emitted by TokenMessengerV2 when USDC exits a chain
        {
            Address: []string{"0x8FE6B999Dc680CcFDD5Bf7EB0974218be2542DAA"},
            Topics:  [][]string{{DepositForBurnTopic}},
        },
        // CCTP MessageReceived — emitted by MessageTransmitterV2 when USDC arrives on Arc
        {
            Address: []string{"0xE737e5cEBEEBa77EFE34D4aa090756590b1CE275"},
            Topics:  [][]string{{MessageReceivedTopic}},
        },
        // Gateway — GatewayWallet and GatewayMinter events
        {
            Address: []string{
                "0x0077777d7EBA4688BDeF3E311b846F25870A19B9", // GatewayWallet
                "0x0022222ABE238Cc2C7Bb1f21003F0a260052475B", // GatewayMinter
            },
        },
        // StableFX — FxEscrow swap lifecycle events
        {
            Address: []string{"0x867650F5eAe8df91445971f14d89fd84F0C9a9f8"},
        },
        // ERC-8004 agent registration (address TBD — check Arc docs for registry deployment)
        {
            Address: []string{AgentRegistryAddress},
            Topics:  [][]string{{AgentRegisteredTopic}},
        },
        // ERC-8183 job lifecycle (address TBD — check Arc docs for escrow deployment)
        {
            Address: []string{JobEscrowAddress},
            Topics:  [][]string{{JobCreatedTopic, JobSettledTopic, JobDeliveredTopic}},
        },
    },
}
```

---

## Contract Addresses (Arc Testnet)

```go
// Stablecoins
const (
    USDCAddress = "0x3600000000000000000000000000000000000000" // 6 decimals (ERC-20), 18 decimals (native gas)
    EURCAddress = "0x89B50855Aa3bE2F677cD6303Cec089B5F319D72a" // 6 decimals
    USYCAddress = "0xe9185F0c5F296Ed1797AaE4238D26CCaBEadb86C" // 6 decimals
)

// CCTP (Arc domain = 26)
const (
    CCTPTokenMessengerV2     = "0x8FE6B999Dc680CcFDD5Bf7EB0974218be2542DAA"
    CCTPMessageTransmitterV2 = "0xE737e5cEBEEBa77EFE34D4aa090756590b1CE275"
    CCTPTokenMinterV2        = "0xb43db544E2c27092c107639Ad201b3dEfAbcF192"
    CCTPMessageV2            = "0xbaC0179bB358A8936169a63408C8481D582390C4"
)

// Gateway
const (
    GatewayWallet = "0x0077777d7EBA4688BDeF3E311b846F25870A19B9"
    GatewayMinter = "0x0022222ABE238Cc2C7Bb1f21003F0a260052475B"
)

// StableFX
const (
    FxEscrow = "0x867650F5eAe8df91445971f14d89fd84F0C9a9f8"
)

// Common
const (
    Permit2      = "0x000000000022D473030F116dDEE9F6B43aC78BA3"
    Multicall3   = "0xcA11bde05977b3631167028862bE2a173976CA11"
    Create2Factory = "0x4e59b44847b379578588920cA78FbF26c0B4956C"
)
```

## Event Signatures (Keccak256 Topics)

```go
var (
    // ERC-20
    TransferEventTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

    // CCTP v2
    // keccak256("DepositForBurn(uint64,address,uint256,address,bytes32,uint32,bytes32,bytes32,uint256,uint32,bytes)")
    DepositForBurnTopic  = "0x2fa9ca894982930190727e75500a97d8dc500233a5065e0f3126c48fbe0343c0"
    // keccak256("MessageReceived(address,uint32,uint64,bytes32,bytes)")
    MessageReceivedTopic = "0x58200b4c34ae05ee816d710053fff3ad1bcea173d0113462f6fd5162ab9adca5"

    // ERC-8004 Agent Registry — address TBD, verify from Arc docs
    AgentRegisteredTopic = "..." // keccak256("AgentRegistered(address,string)")

    // ERC-8183 Job Escrow — address TBD, verify from Arc docs
    JobCreatedTopic   = "..." // keccak256("JobCreated(uint256,address,address,uint256)")
    JobDeliveredTopic = "..." // keccak256("JobDelivered(uint256,address)")
    JobSettledTopic   = "..." // keccak256("JobSettled(uint256,uint256)")

    // StableFX FxEscrow — verify signatures via testnet.arcscan.app ABI viewer
    // SwapCreated, SwapSettled, SwapCancelled
)
```

---

## What the Frontend Gets

Everything above is queryable via PocketBase REST + real-time via PocketBase websockets.

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

**Time-series charts (from `block_stats`):**
- TPS over time
- Avg USDC fee per tx over time (real USD, unique to Arc)
- Total USDC / EURC / USYC transferred per block
- StableFX swap volume and implied USDC/EURC rate over time
- Active wallets per block
- Block time distribution (sub-second finality histogram)
- Cross-chain inflow volume (CCTP + Gateway)
- New contract deployments over time
- Agent job settlement rate

---

## Stack

- **Indexer**: Go + `github.com/enviodev/hypersync-client-go`
- **Database + API + Websockets**: PocketBase (SQLite under the hood)
- **Chain**: Arc Testnet — `https://arc-testnet.hypersync.xyz`
- **Frontend**: 3D visualizer (separate repo)

---

## Key Arc Testnet References

- **Chain ID**: `5042002`
- HyperSync endpoint: `https://arc-testnet.hypersync.xyz` (also `https://5042002.hypersync.xyz`)
- RPC endpoint: `https://rpc.testnet.arc-node.thecanteenapp.com/v1/<key>` — key-gated, get yours via `arc-canteen login && arc-canteen rpc-url`
- Block explorer: `https://testnet.arcscan.app`
- Contract addresses: `https://docs.arc.network/arc/references/contract-addresses.md`
- Faucet (USDC + EURC): `https://faucet.circle.com`
- CCTP domain for Arc: **26**
- ABI lookup: `https://testnet.arcscan.app/address/<contract>` → Contract tab → ABI

## Outstanding TODOs

- [ ] Confirm ERC-8004 Agent Registry contract address from Arc docs / Discord
- [ ] Confirm ERC-8183 Job Escrow contract address from Arc docs / Discord
- [ ] Verify `DepositForBurnTopic` and `MessageReceivedTopic` hashes against live CCTP v2 ABI
- [ ] Pull StableFX FxEscrow event signatures from ABI via arcscan
- [ ] Verify Gateway event signatures from ABI via arcscan
