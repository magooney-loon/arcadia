/* Mock data for Arcadia Explorer — modelled after the API schema. */

const HEAD_BLOCK = 14_283_991;
const NOW = Date.now();

const rand = (seed) => {
  let s = seed | 0;
  return () => {
    s = (s * 1664525 + 1013904223) | 0;
    return ((s >>> 0) % 100000) / 100000;
  };
};
const r = rand(42);
const pick = (arr) => arr[Math.floor(r() * arr.length)];

const ADDR_POOL = [
  "0xae4b51c2c8a3f9c5a1b2d6e9f1c3b5a7e9d2c4f6",
  "0x7c1d3e5a9b2f4c6e8a0d2b4f6c8e0a2d4b6f8c0e",
  "0x3a5c7e9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9a1c",
  "0xb1c2d3e4f5a6b7c8d9e0a1b2c3d4e5f6a7b8c9d0",
  "0xff89aa4c5e3b1d2c8a9f6e5d4c3b2a1908f7e6d5",
  "0x4d6e8a0b2c4e6a8c0e2a4c6e8a0c2e4a6c8e0a2c",
  "0x9d8c7b6a5e4f3d2c1b0a9f8e7d6c5b4a3928170f",
  "0x2c4e6a8c0e2a4c6e8a0c2e4a6c8e0a2c4e6a8c0e",
  "0x5b7d9f1a3c5e7a9b1d3f5a7c9e1b3d5f7a9c1e3b",
  "0xcafe1234567890abcdef1234567890abcdef1234",
  "0xbeef0987654321fedcba0987654321fedcba0987",
  "0xdead11112222333344445555666677778888beef",
];

const AGENTS = [
  { addr: "0xae4b51c2c8a3f9c5a1b2d6e9f1c3b5a7e9d2c4f6", name: "Vesper.fi/orchestrator", domain: "vesper.fi", trust: 94, jobs: 1284, success: 98.2 },
  { addr: "0x7c1d3e5a9b2f4c6e8a0d2b4f6c8e0a2d4b6f8c0e", name: "stablefx-routing-v3",     domain: "router.arcadia.xyz", trust: 91, jobs: 8912, success: 99.4 },
  { addr: "0x3a5c7e9b1d3f5a7c9e1b3d5f7a9c1e3b5d7f9a1c", name: "treasury.policy.agent",   domain: "policy.dao", trust: 88, jobs: 312, success: 96.5 },
  { addr: "0xb1c2d3e4f5a6b7c8d9e0a1b2c3d4e5f6a7b8c9d0", name: "settlement-keeper",      domain: "ops.circle", trust: 99, jobs: 22409, success: 99.9 },
  { addr: "0xff89aa4c5e3b1d2c8a9f6e5d4c3b2a1908f7e6d5", name: "rfq.market-maker.07",   domain: "mm.arcadia.xyz", trust: 86, jobs: 4187, success: 97.1 },
  { addr: "0x4d6e8a0b2c4e6a8c0e2a4c6e8a0c2e4a6c8e0a2c", name: "Compliance.Sentinel",    domain: "comply.octopus", trust: 92, jobs: 671, success: 99.0 },
  { addr: "0x9d8c7b6a5e4f3d2c1b0a9f8e7d6c5b4a3928170f", name: "yield.optimizer.beta",   domain: "labs.kinetic", trust: 74, jobs: 92, success: 89.1 },
  { addr: "0x2c4e6a8c0e2a4c6e8a0c2e4a6c8e0a2c4e6a8c0e", name: "fxoracle.consensus",     domain: "oracle.arc", trust: 96, jobs: 14002, success: 99.7 },
];

const TOKENS = [
  { sym: "USDC",  decimals: 6,  color: "info" },
  { sym: "EURC",  decimals: 6,  color: "info" },
  { sym: "BRZ",   decimals: 6,  color: "ok"   },
  { sym: "MXNe",  decimals: 6,  color: "warn" },
  { sym: "ARC",   decimals: 18, color: "acc"  },
  { sym: "PYUSD", decimals: 6,  color: "info" },
];

const CHAINS = {
  0:  { name: "Ethereum",  short: "ETH" },
  1:  { name: "Avalanche", short: "AVAX" },
  2:  { name: "OP Mainnet", short: "OP" },
  3:  { name: "Arbitrum",  short: "ARB" },
  6:  { name: "Base",      short: "BASE" },
  7:  { name: "Polygon",   short: "POLY" },
  9:  { name: "Solana",    short: "SOL" },
  26: { name: "Arcadia",   short: "ARC" },
};

const shortAddr = (a) =>
  a ? a.slice(0, 6) + "…" + a.slice(-4) : "";

const shortHash = (h) =>
  h ? h.slice(0, 10) + "…" + h.slice(-6) : "";

const fmtNum = (n, d = 0) => {
  if (n === undefined || n === null) return "—";
  if (n >= 1e9) return (n / 1e9).toFixed(d || 2) + "B";
  if (n >= 1e6) return (n / 1e6).toFixed(d || 2) + "M";
  if (n >= 1e3) return (n / 1e3).toFixed(d || 1) + "k";
  return n.toLocaleString("en-US", { minimumFractionDigits: d, maximumFractionDigits: d });
};
const fmtFull = (n) => Number(n).toLocaleString("en-US");

const ago = (ms) => {
  const s = Math.floor((NOW - ms) / 1000);
  if (s < 60) return s + "s";
  if (s < 3600) return Math.floor(s / 60) + "m";
  if (s < 86400) return Math.floor(s / 3600) + "h";
  return Math.floor(s / 86400) + "d";
};

const randHash = () => {
  let h = "0x";
  for (let i = 0; i < 64; i++) h += "0123456789abcdef"[Math.floor(r() * 16)];
  return h;
};

// ── Stats ───────────────────────────────────────────────
const STATS = {
  tps: 4187,
  tps_peak: 6021,
  block_time_ms: 412,
  block_height: HEAD_BLOCK,
  indexed_block: HEAD_BLOCK,
  total_txs_24h: 312_457_891,
  fees_paid_24h: 18421,
  fees_unit: "USDC",
  transfer_vol_24h: 41_887_293_010,
  transfer_count_24h: 14_201_882,
  active_agents: 1287,
  cctp_mints_24h: 22_491,
  cctp_burns_24h: 19_872,
  fx_swaps_24h: 184_001,
  fx_notional_24h: 8_201_410_000,
};

// ── Block stats time series (sparkline data) ────────────
const SERIES = {
  tps:        Array.from({ length: 60 }, (_, i) => 3500 + Math.sin(i / 6) * 600 + r() * 400),
  block_time: Array.from({ length: 60 }, (_, i) => 410 + Math.cos(i / 5) * 25 + r() * 15),
  gas:        Array.from({ length: 60 }, (_, i) => 0.0012 + r() * 0.0006),
  vol:        Array.from({ length: 60 }, (_, i) => 1.6e9 + Math.sin(i / 9) * 4e8 + r() * 2e8),
  fx_notional: Array.from({ length: 24 }, (_, i) => 280e6 + r() * 120e6 + Math.sin(i / 3) * 90e6),
  cctp_in:     Array.from({ length: 24 }, (_, i) => 800 + r() * 400 + Math.sin(i / 4) * 200),
  cctp_out:    Array.from({ length: 24 }, (_, i) => 700 + r() * 350 + Math.cos(i / 4) * 220),
};

// ── Blocks ──────────────────────────────────────────────
const BLOCKS = Array.from({ length: 40 }, (_, i) => {
  const n = HEAD_BLOCK - i;
  return {
    number: n,
    hash: randHash(),
    parent_hash: randHash(),
    miner: pick(["0xae4b51c2c8a3f9c5a1b2d6e9f1c3b5a7e9d2c4f6", "0x7c1d3e5a9b2f4c6e8a0d2b4f6c8e0a2d4b6f8c0e", "0xb1c2d3e4f5a6b7c8d9e0a1b2c3d4e5f6a7b8c9d0"]),
    tx_count: Math.floor(1200 + r() * 2400),
    transfer_count: Math.floor(800 + r() * 1900),
    gas_used: Math.floor(8_000_000 + r() * 22_000_000),
    block_time_ms: Math.floor(380 + r() * 80),
    timestamp: NOW - i * 410,
    fees: (0.4 + r() * 0.9).toFixed(4),
  };
});

// ── Transactions ────────────────────────────────────────
const TX_KINDS = ["transfer", "swap", "cctp_burn", "cctp_mint", "agent_call", "deploy", "approve", "stake"];
const TXS = Array.from({ length: 80 }, (_, i) => {
  const kind = pick(TX_KINDS);
  return {
    hash: randHash(),
    block_number: HEAD_BLOCK - Math.floor(i / 6),
    index: i % 6,
    from_addr: pick(ADDR_POOL),
    to_addr: pick(ADDR_POOL),
    kind,
    value: kind === "transfer" ? (r() * 50000).toFixed(2) : "0",
    fee: (0.0001 + r() * 0.002).toFixed(6),
    gas_used: Math.floor(21000 + r() * 280000),
    status: r() > 0.04 ? "ok" : "reverted",
    timestamp: NOW - i * 410,
  };
});

// ── Transfers ───────────────────────────────────────────
const TRANSFERS = Array.from({ length: 60 }, (_, i) => {
  const tok = pick(TOKENS);
  return {
    tx_hash: randHash(),
    block_number: HEAD_BLOCK - Math.floor(i / 4),
    log_index: i % 4,
    token_symbol: tok.sym,
    token_color: tok.color,
    from_addr: pick(ADDR_POOL),
    to_addr: pick(ADDR_POOL),
    amount: (r() * 250000 + 100).toFixed(2),
    usd_value: (r() * 250000 + 100).toFixed(0),
    timestamp: NOW - i * 410,
  };
});

// ── Cross-chain events ──────────────────────────────────
const CROSSCHAIN = Array.from({ length: 40 }, (_, i) => {
  const inbound = r() > 0.45;
  const otherDomain = pick([0, 1, 2, 3, 6, 7, 9]);
  return {
    id: "cc_" + (HEAD_BLOCK - i),
    protocol: pick(["CCTP", "CCTP", "CCTP", "Gateway"]),
    event_type: inbound ? "MintAndWithdraw" : "DepositForBurn",
    source_domain: inbound ? otherDomain : 26,
    destination_domain: inbound ? 26 : otherDomain,
    sender: pick(ADDR_POOL),
    recipient: pick(ADDR_POOL),
    amount: (r() * 5_000_000 + 10000).toFixed(2),
    token: pick(["USDC", "USDC", "USDC", "EURC"]),
    block_number: HEAD_BLOCK - Math.floor(i / 2),
    timestamp: NOW - i * 1200,
    status: r() > 0.1 ? "finalized" : "pending",
  };
});

// ── StableFX trades ─────────────────────────────────────
const FX_PAIRS = [
  ["USDC", "EURC", 0.9203],
  ["USDC", "BRZ",  5.4231],
  ["USDC", "MXNe", 17.8912],
  ["EURC", "BRZ",  5.8901],
  ["USDC", "PYUSD", 1.0001],
];
const FX = Array.from({ length: 60 }, (_, i) => {
  const [base, quote, mid] = pick(FX_PAIRS);
  const px = mid * (1 + (r() - 0.5) * 0.001);
  const size = (r() * 2_000_000 + 5_000).toFixed(0);
  return {
    quote_id: "qid_" + (1000000 + Math.floor(r() * 9000000)),
    block_number: HEAD_BLOCK - Math.floor(i / 3),
    maker: pick(AGENTS).addr,
    taker: pick(ADDR_POOL),
    base_token: base,
    quote_token: quote,
    base_amount: size,
    quote_amount: (size * px).toFixed(0),
    price: px.toFixed(6),
    status: pick(["filled", "filled", "filled", "filled", "partial", "expired"]),
    timestamp: NOW - i * 600,
  };
});

// ── Agent jobs ──────────────────────────────────────────
const JOB_STATES = ["proposed", "active", "completed", "completed", "completed", "disputed"];
const JOBS = Array.from({ length: 40 }, (_, i) => {
  const employer = pick(AGENTS);
  const worker = pick(AGENTS);
  return {
    job_id: "job_" + (200000 + i),
    employer_address: employer.addr,
    employer_name: employer.name,
    worker_address: worker.addr,
    worker_name: worker.name,
    title: pick([
      "Rebalance USDC/EURC AMM weights",
      "Settle 24h CCTP burn batch → ETH",
      "Audit cross-chain liquidity drift",
      "Quote RFQ batch #4081",
      "Reprice MXNe pool against Bitso index",
      "Submit on-chain attestation 0x82…",
      "Refresh oracle feed: BRL/USD",
      "KYC verification batch (47 accts)",
    ]),
    bounty_usdc: (r() * 2400 + 80).toFixed(2),
    created_at_block: HEAD_BLOCK - Math.floor(r() * 8000),
    status: pick(JOB_STATES),
    created_at: NOW - r() * 1000 * 3600 * 24,
  };
});

// ── Wallet edges (graph) ────────────────────────────────
const EDGES = Array.from({ length: 80 }, () => ({
  from_wallet: pick(ADDR_POOL),
  to_wallet: pick(ADDR_POOL),
  tx_count: Math.floor(r() * 800 + 5),
  volume: (r() * 12_000_000 + 1000).toFixed(0),
  first_tx_block: HEAD_BLOCK - Math.floor(r() * 100000),
  last_tx_block: HEAD_BLOCK - Math.floor(r() * 200),
}));

Object.assign(window, {
  STATS, SERIES, BLOCKS, TXS, TRANSFERS, CROSSCHAIN, FX, JOBS, EDGES,
  AGENTS, ADDR_POOL, TOKENS, CHAINS, HEAD_BLOCK,
  shortAddr, shortHash, fmtNum, fmtFull, ago, randHash,
});
