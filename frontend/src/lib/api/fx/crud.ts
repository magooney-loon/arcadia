import { getApiUrl } from '../../stores/config.svelte.js';
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
	async list(filter: FxFilter = {}): Promise<FxResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		const res = await fetch(`${getApiUrl()}/api/v1/fx${qs({ limit, offset, ...rest })}`);
		if (!res.ok) throw new Error(`fx: ${res.status}`);
		return res.json();
	}
}
