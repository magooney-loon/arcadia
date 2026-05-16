import { apiFetch } from '../utils.js';
import type { StatsResponse } from './types.js';

export class StatsCrudClient {
	get(): Promise<StatsResponse> {
		return apiFetch<StatsResponse>('/api/v1/stats');
	}
}
