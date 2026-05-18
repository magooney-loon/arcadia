export type Window = '1h' | '24h' | '7d';

// ── /analytics/fees ───────────────────────────────────────────────────────────

export interface FeesFilter {
	window?: Window;
}

export interface FeesResponse {
	window: string;
	block_count: number;
	total_fees: number;
	avg_fee_p25: number;
	avg_fee_p50: number;
	avg_fee_p75: number;
	avg_fee_p95: number;
	avg_block_time_ms: number;
	failed_tx_ratio: number;
}

// ── /analytics/volume ─────────────────────────────────────────────────────────

export interface VolumeFilter {
	window?: Window;
	token?: string;
}

export interface TokenStats {
	volume: number;
	count: number;
	whale_count: number;
}

export interface VolumeResponse {
	window: string;
	token: string;
	total_transfers: number;
	unique_senders: number;
	unique_receivers: number;
	whale_transfers: number;
	by_token: Record<string, TokenStats>;
}

// ── /analytics/bridge_flow ───────────────────────────────────────────────────

export interface BridgeFlowFilter {
	window?: Window;
}

export interface ChainFlow {
	inbound_vol: number;
	inbound_count: number;
	outbound_vol: number;
	outbound_count: number;
}

export interface BridgeFlowResponse {
	window: string;
	inbound_vol: number;
	inbound_count: number;
	outbound_vol: number;
	outbound_count: number;
	net_flow: number;
	by_chain: Record<string, ChainFlow>;
}

// ── /analytics/agent_leaderboard ─────────────────────────────────────────────

export interface AgentLeaderboardEntry {
	id: string;
	agent_address: string;
	tx_count: number;
	usdc_transferred: string;
	usdc_transferred_human?: string;
	usdc_spent_fees: string;
	usdc_spent_fees_human?: string;
	registered_at_block: number;
	job_count: number;
	total_escrow: number;
	paid_jobs: number;
	rejected_jobs: number;
	[key: string]: unknown;
}

export interface AgentLeaderboardSummary {
	total_escrow: number;
	jobs_total: number;
	jobs_paid: number;
	jobs_rejected: number;
	jobs_in_flight: number;
}

export interface AgentLeaderboardResponse {
	leaderboard: AgentLeaderboardEntry[];
	count: number;
	total: number;
	summary?: AgentLeaderboardSummary;
}

// ── /analytics/overview ───────────────────────────────────────────────────────

export interface OverviewFilter {
	window?: Window;
}

export interface OverviewResponse {
	window: string;
	snapshot_at: number;
	transfers_count: number;
	transfer_volume: number;
	largest_transfer: number;
	largest_transfer_block: number;
	fees_total: number;
	fee_p50: number;
	fee_p95: number;
	failed_tx_ratio: number;
	bridge_inbound_vol: number;
	bridge_inbound_count: number;
	bridge_outbound_vol: number;
	bridge_outbound_count: number;
	bridge_net_flow: number;
	agent_count: number;
}

// ── /analytics/history ────────────────────────────────────────────────────────

export interface HistoryFilter {
	window?: Window;
	limit?: number;
}

export interface AnalyticsSnapshot {
	id: string;
	snapshot_at: number;
	block_number: number;
	window: string;
	transfers_count: number;
	transfer_volume: number;
	largest_transfer: number;
	largest_transfer_block: number;
	usdc_volume: number;
	eurc_volume: number;
	usyc_volume: number;
	usdc_count: number;
	eurc_count: number;
	usyc_count: number;
	whale_transfers: number;
	unique_senders: number;
	unique_receivers: number;
	total_transfers: number;
	fees_total: number;
	fee_p25: number;
	fee_p50: number;
	fee_p75: number;
	fee_p95: number;
	failed_tx_ratio: number;
	total_txs: number;
	failed_txs: number;
	avg_block_time_ms: number;
	block_count: number;
	bridge_inbound_vol: number;
	bridge_inbound_count: number;
	bridge_outbound_vol: number;
	bridge_outbound_count: number;
	bridge_net_flow: number;
	bridge_by_chain: Record<string, ChainFlow>;
	agent_count: number;
}

export interface HistoryResponse {
	window: string;
	snapshots: AnalyticsSnapshot[];
	count: number;
}
