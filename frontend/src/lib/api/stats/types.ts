export interface StatsResponse {
	// identity
	block_number?: number;
	timestamp?: number;
	// throughput (rolling 10-block avg computed server-side)
	tps?: number;
	tx_count?: number;
	failed_tx_count?: number;
	block_time_ms?: number;
	// fees
	avg_fee_usdc?: string;
	total_fee_usdc?: string;
	// transfer volumes
	total_usdc_transferred?: string;
	total_eurc_transferred?: string;
	total_usyc_transferred?: string;
	largest_usdc_transfer?: string;
	// activity
	unique_senders?: number;
	unique_receivers?: number;
	new_contracts?: number;
	utilization_pct?: number;
	// indexer cursor
	indexed_block?: string;
	// fallback when indexer hasn't caught up yet
	syncing?: boolean;
	[key: string]: unknown;
}
