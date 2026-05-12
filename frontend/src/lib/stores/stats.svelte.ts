import { StatsCrudClient } from '../api/stats/crud.js';
import type { StatsResponse } from '../api/stats/types.js';

const client = new StatsCrudClient();

export interface StatsState {
	data: StatsResponse | null;
	loading: boolean;
	error: string | null;
}

export const stats = $state<StatsState>({ data: null, loading: false, error: null });

export async function fetchStats() {
	stats.loading = true;
	stats.error = null;
	try {
		stats.data = await client.get();
	} catch (e) {
		stats.error = String(e);
	} finally {
		stats.loading = false;
	}
}
