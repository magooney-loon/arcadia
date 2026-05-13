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
	usdc_spent_fees: string;
	registered_at_block: number;
	job_count: number;
	total_escrow: number;
	settled_jobs: number;
	disputed_jobs: number;
	[key: string]: unknown;
}

export interface AgentLeaderboardResponse {
	leaderboard: AgentLeaderboardEntry[];
	count: number;
}
