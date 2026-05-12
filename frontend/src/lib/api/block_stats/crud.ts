import { getApiUrl } from '../../stores/config.svelte.js';
import type { BlockStatsResponse } from './types.js';

export class BlockStatsCrudClient {
	async list(limit = 50, offset = 0): Promise<BlockStatsResponse> {
		const p = new URLSearchParams({ limit: String(limit), offset: String(offset) });
		const res = await fetch(`${getApiUrl()}/api/v1/block_stats?${p}`);
		if (!res.ok) throw new Error(`block_stats: ${res.status}`);
		return res.json();
	}
}
