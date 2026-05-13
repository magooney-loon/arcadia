export interface HealthResponse {
	last_indexed_block: number;
	chain_tip: number;
	lag_blocks: number;
	syncing: boolean;
	errors_1h: number;
	avg_batch_ms: number;
}
