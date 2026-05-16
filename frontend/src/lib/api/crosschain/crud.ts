import { apiFetch } from '../utils.js';
import type { CrosschainResponse, CrosschainFilter } from './types.js';

function qs(params: Record<string, string | number | undefined>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

export class CrosschainCrudClient {
	list(filter: CrosschainFilter = {}): Promise<CrosschainResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		return apiFetch<CrosschainResponse>(`/api/v1/crosschain${qs({ limit, offset, ...rest })}`);
	}
}
