import { getApiUrl } from '../../stores/config.svelte.js';
import type { TransfersResponse, TransferFilter } from './types.js';

function qs(params: Record<string, string | number | undefined>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

export class TransfersCrudClient {
	async list(filter: TransferFilter = {}): Promise<TransfersResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		const res = await fetch(`${getApiUrl()}/api/v1/transfers${qs({ limit, offset, ...rest })}`);
		if (!res.ok) throw new Error(`transfers: ${res.status}`);
		return res.json();
	}
}
