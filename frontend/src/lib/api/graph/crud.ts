import { apiFetch } from '../utils.js';
import type { EdgesResponse, EdgeFilter } from './types.js';

function qs(params: Record<string, string | number | undefined>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

export class GraphCrudClient {
	edges(filter: EdgeFilter = {}): Promise<EdgesResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		return apiFetch<EdgesResponse>(`/api/v1/edges${qs({ limit, offset, ...rest })}`);
	}
}
