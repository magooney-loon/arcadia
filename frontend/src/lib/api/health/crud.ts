import { getApiUrl } from '../../stores/config.svelte.js';
import type { HealthResponse } from './types.js';

export class HealthCrudClient {
	async get(): Promise<HealthResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/health`);
		if (!res.ok) throw new Error(`health: ${res.status}`);
		return res.json();
	}
}
