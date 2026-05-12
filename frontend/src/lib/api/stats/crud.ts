import { getApiUrl } from '../../stores/config.svelte.js';
import type { StatsResponse } from './types.js';

export class StatsCrudClient {
	async get(): Promise<StatsResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/stats`);
		if (!res.ok) throw new Error(`stats: ${res.status}`);
		return res.json();
	}
}
