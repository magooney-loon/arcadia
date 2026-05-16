import { apiFetch } from '../utils.js';
import type { BlockStatsResponse } from './types.js';

export class BlockStatsCrudClient {
	list(limit = 50, offset = 0): Promise<BlockStatsResponse> {
		const p = new URLSearchParams({ limit: String(limit), offset: String(offset) });
		return apiFetch<BlockStatsResponse>(`/api/v1/block_stats?${p}`);
	}
}
