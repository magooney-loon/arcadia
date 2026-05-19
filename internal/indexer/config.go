package indexer

import "time"

// ── Startup ───────────────────────────────────────────────────────────────────

// startupDelay is how long to wait after server boot before connecting to HyperSync.
const startupDelay = 10 * time.Second

// catchupLookback is how many blocks behind the chain tip to start on a fresh
// run (no saved cursor). Increase to index more history; set to 0 to start from
// the tip.
const catchupLookback = uint64(3_600)

// ── Batch sizing & pacing ─────────────────────────────────────────────────────

// batchSize is the number of blocks requested per HyperSync fetch.
// Smaller = shorter SQLite write transactions (better read latency).
// Larger = fewer WAN round-trips (better catch-up throughput).
const batchSize = uint64(50)

// atTipPollInterval is how long to wait before re-checking when the indexer is
// caught up and there are no new blocks to process.
const atTipPollInterval = 2 * time.Second

// sprintLagThreshold: when lag exceeds this many blocks, skip all pacing sleeps
// and fetch as fast as possible.
const sprintLagThreshold = uint64(500)

// nearTipLagThreshold: when lag is between this value and sprintLagThreshold,
// apply nearTipSleep to ease pressure on the RPC.
const nearTipLagThreshold = uint64(100)

// nearTipSleep is the pacing delay when lag is between nearTipLagThreshold and
// sprintLagThreshold.
const nearTipSleep = 50 * time.Millisecond

// crawlSleep is the pacing delay when lag is below nearTipLagThreshold (nearly
// caught up to the tip).
const crawlSleep = 200 * time.Millisecond

// prefetchConcurrency is the maximum number of concurrent token metadata RPC
// lookups during the pre-transaction prefetch phase. Keep low to avoid competing
// with the indexer's own SQLite writes.
const prefetchConcurrency = 3

// ── Retry & error recovery ────────────────────────────────────────────────────

// rateLimitBackoff is how long to wait after receiving a 429 from HyperSync
// before retrying.
const rateLimitBackoff = 30 * time.Second

// crashRestartDelay is how long to wait before restarting the indexer loop after
// an unexpected error.
const crashRestartDelay = 5 * time.Second

// hypersyncMaxRetries is the number of times the HyperSync client retries a
// failed batch fetch before surfacing an error.
const hypersyncMaxRetries = 3

// hypersyncRetryBase is the initial retry backoff for the HyperSync client.
const hypersyncRetryBase = 500 * time.Millisecond

// hypersyncRetryBackoff is the per-attempt backoff increment.
const hypersyncRetryBackoff = 500 * time.Millisecond

// hypersyncRetryCeiling is the maximum retry backoff for the HyperSync client.
const hypersyncRetryCeiling = 3 * time.Second

// tipCheckTimeout is the context deadline for each chain-tip RPC call.
const tipCheckTimeout = 5 * time.Second

// ── Heartbeat ─────────────────────────────────────────────────────────────────

// heartbeatInterval is how often the indexer logs its current state.
const heartbeatInterval = 15 * time.Second

// heartbeatPersistInterval is how often a heartbeat is written to the
// indexer_events table. Less frequent than logging since it hits SQLite.
const heartbeatPersistInterval = time.Minute

// ── WAL maintenance ───────────────────────────────────────────────────────────

// walCheckpointCooldown is the minimum time between PRAGMA wal_checkpoint(TRUNCATE)
// calls. The checkpoint briefly blocks until readers drain, so it should not run
// too frequently.
const walCheckpointCooldown = time.Minute

// ── Event logging ─────────────────────────────────────────────────────────────

// eventChannelBuffer is the capacity of the async indexer-event channel. Events
// are silently dropped (never blocked) when the buffer is full.
const eventChannelBuffer = 256

// maxEventMessage is the maximum character length for the indexer_events message
// field, mirroring the collection schema cap.
const maxEventMessage = 500

// maxEventError is the maximum character length for the indexer_events error field.
const maxEventError = 1000
