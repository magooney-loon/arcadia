import { apiFetch } from '../utils.js';
import type { HealthResponse } from './types.js';

export class HealthCrudClient {
	get(): Promise<HealthResponse> {
		return apiFetch<HealthResponse>('/api/v1/health');
	}
}
