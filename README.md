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
| `fee_usdc` | derived: `gas_used × effective_gas_price / 1e6` | **Real USD cost of this tx — Arc's killer differentiator** |
| `nonce` | TransactionField.NONCE | Sender sequence / burst detection |
| `input` | TransactionField.INPUT | Raw calldata |
| `sighash` | derived: first 4 bytes of `input` | Method signature for tx categorization |
| `type` | TransactionField.KIND | Tx type (legacy / EIP-1559) |
| `contract_address` | TransactionField.CONTRACT_ADDRESS | Non-null = contract deployment |
| `status` | TransactionField.STATUS | Success / failure |

---

#### Layer 3 — Token Transfers (ERC-20)
**`transfers` collection**

Covers: USDC, EURC, and all other ERC-20 tokens on Arc.

Event topic: `Transfer(address indexed from, address indexed to, uint256 value)`
`0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef`

| Field | Source | Why |
|---|---|---|
| `tx_hash` | LogField.TRANSACTION_HASH | Transaction linkage |
| `block_number` | LogField.BLOCK_NUMBER | Time axis |
| `log_index` | LogField.LOG_INDEX | Dedup key |
| `token_address` | LogField.ADDRESS | Contract identity |
| `token_symbol` | derived: match against known addresses | USDC / EURC / other |
| `from` | LogField.TOPIC1 (last 20 bytes) | Sender wallet node |
| `to` | LogField.TOPIC2 (last 20 bytes) | Receiver wallet node |
| `amount_raw` | LogField.DATA | Raw uint256 |
| `amount_usdc` | derived: `amount_raw / 1e6` | Human-readable (USDC has 6 decimals) |

Known token addresses (Arc Testnet):
- USDC: from `docs.arc.network/arc/references/contract-addresses.md`
- EURC: from `docs.arc.network/arc/references/contract-addresses.md`

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

#### Layer 6 — Cross-Chain Flows (CCTP)
**`cctp_events` collection**

CCTP events show capital entering/leaving Arc from other chains. Visualized as inbound/outbound arrows from the broader crypto ecosystem.

Events to index:
- `DepositForBurn` — USDC leaving a chain toward Arc
- `MintAndWithdraw` — USDC arriving on Arc from another chain

| Field | Why |
|---|---|
| `event_type` | deposit / mint |
| `source_chain` | Origin chain ID |
| `destination_chain` | Destination chain ID |
| `amount_usdc` | Transfer size |
| `sender` | Origin wallet |
| `recipient` | Destination wallet |
| `block_number` | Timeline |
| `nonce` | Dedup / cross-chain correlation |

CCTP contract address: from `docs.arc.network/arc/references/contract-addresses.md`

---

#### Layer 7 — Derived Block Stats (Pre-aggregated)
**`block_stats` collection**

Stored at index time so the frontend never does heavy aggregations at query time.

| Metric | Formula |
|---|---|
| `tps` | `tx_count / block_time_seconds` |
| `avg_fee_usdc` | `sum(fee_usdc) / tx_count` |
| `total_usdc_transferred` | `sum(amount_usdc) FROM transfers WHERE block = N` |
| `total_eurc_transferred` | same, filtered to EURC address |
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
        // All ERC-20 transfers (USDC, EURC, any token)
        {Topics: [][]string{{TransferEventTopic}}},
        // CCTP DepositForBurn
        {
            Address: []string{CCTPContractAddress},
            Topics:  [][]string{{DepositForBurnTopic}},
        },
        // CCTP MintAndWithdraw
        {
            Address: []string{CCTPContractAddress},
            Topics:  [][]string{{MintAndWithdrawTopic}},
        },
        // ERC-8004 agent registration
        {
            Address: []string{AgentRegistryAddress},
            Topics:  [][]string{{AgentRegisteredTopic}},
        },
        // ERC-8183 job lifecycle
        {
            Address: []string{JobEscrowAddress},
            Topics:  [][]string{{JobCreatedTopic, JobSettledTopic, JobDeliveredTopic}},
        },
    },
}
```

---

## Event Signatures (Keccak256 Topics)

```go
var (
    // ERC-20
    TransferEventTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

    // CCTP
    DepositForBurnTopic  = "0x2fa9ca894982930190727e75500a97d8dc500233a5065e0f3126c48fbe0343c0"
    MintAndWithdrawTopic = "0x1b2a7ff080b8cb6ff436ce0372e399692bbfb6d4ae5766fd8d58a7b8cc6142e9"

    // ERC-8004 Agent Registry (verify address from Arc docs)
    AgentRegisteredTopic = "..." // keccak256("AgentRegistered(address,string)")

    // ERC-8183 Job Escrow (verify address from Arc docs)
    JobCreatedTopic   = "..." // keccak256("JobCreated(uint256,address,address,uint256)")
    JobDeliveredTopic = "..." // keccak256("JobDelivered(uint256,address)")
    JobSettledTopic   = "..." // keccak256("JobSettled(uint256,uint256)")
)
```

---

## What the Frontend Gets

Everything above is queryable via PocketBase REST + real-time via PocketBase websockets.

**3D visualization layers:**
1. **Chain spine** — blocks as nodes, time on the Z axis, utilization as heat color
2. **Transaction particles** — particles flying between wallet nodes, sized by value
3. **USDC blood flow** — animated edges between wallets, thickness = transfer amount
4. **EURC layer** — separate color channel alongside USDC
5. **Agent network** — agent wallets highlighted, agent-to-agent edges glowing
6. **Cross-chain arrows** — CCTP inflows rendered as external capital entering the Arc sphere
7. **Fee heatmap** — per-block USDC fee burn, proving Arc's $0.01 tx cost claim
8. **Job economy** — ERC-8183 job arcs between employer and worker agent wallets

**Time-series charts (from `block_stats`):**
- TPS over time
- Avg USDC fee per tx over time
- Total USDC transferred per block
- Active wallets per block
- Block time distribution (sub-second finality histogram)
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

- HyperSync endpoint: `https://arc-testnet.hypersync.xyz`
- Block explorer: `https://testnet.arcscan.app`
- Contract addresses: `https://docs.arc.network/arc/references/contract-addresses.md`
- Faucet: `https://faucet.circle.com`
- RPC: `https://docs.arc.network/arc/references/connect-to-arc.md`
