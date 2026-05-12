import { BlockStatsCrudClient } from '../api/block_stats/crud.js';
import type { BlockStatsResponse } from '../api/block_stats/types.js';

const client = new BlockStatsCrudClient();

export interface BlockStatsState {
	data: BlockStatsResponse | null;
	loading: boolean;
	error: string | null;
}

export const blockStats = $state<BlockStatsState>({ data: null, loading: false, error: null });

export async function fetchBlockStats(limit = 50, offset = 0) {
	blockStats.loading = true;
	blockStats.error = null;
	try {
		blockStats.data = await client.list(limit, offset);
	} catch (e) {
		blockStats.error = String(e);
	} finally {
		blockStats.loading = false;
	}
}
