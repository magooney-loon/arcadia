import { apiFetch } from '../utils.js';
import type { FxResponse, FxFilter } from './types.js';

function qs(params: Record<string, string | number | undefined>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

export class FxCrudClient {
	list(filter: FxFilter = {}): Promise<FxResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		return apiFetch<FxResponse>(`/api/v1/fx${qs({ limit, offset, ...rest })}`);
	}
}
