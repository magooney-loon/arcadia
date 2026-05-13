import { HealthCrudClient } from '../api/health/crud.js';
import type { HealthResponse } from '../api/health/types.js';

const client = new HealthCrudClient();

export interface HealthState {
	data: HealthResponse | null;
	loading: boolean;
	error: string | null;
}

export const health = $state<HealthState>({ data: null, loading: false, error: null });

export async function fetchHealth() {
	health.loading = true;
	health.error = null;
	try {
		health.data = await client.get();
	} catch (e) {
		health.error = String(e);
	} finally {
		health.loading = false;
	}
}
