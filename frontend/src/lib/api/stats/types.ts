export interface StatsResponse {
	block_number?: number;
	tx_count?: number;
	block_time_ms?: number;
	tps?: number;
	indexed_block?: string;
	syncing?: boolean;
	[key: string]: unknown;
}
