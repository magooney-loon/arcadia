export interface BlockStat {
	id: string;
	block_number: number;
	timestamp: number;
	tps?: number;
	tx_count?: number;
	failed_tx_count?: number;
	avg_fee_usdc?: string;
	total_fee_usdc?: string;
	total_usdc_transferred?: string;
	total_eurc_transferred?: string;
	total_usyc_transferred?: string;
	unique_senders?: number;
	unique_receivers?: number;
	new_contracts?: number;
	largest_usdc_transfer?: string;
	utilization_pct?: number;
	block_time_ms?: number;
	[key: string]: unknown;
}

export interface BlockStatsResponse {
	stats: BlockStat[];
	count: number;
}
