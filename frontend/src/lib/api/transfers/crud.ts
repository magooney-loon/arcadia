import { apiFetch } from '../utils.js';
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
	list(filter: TransferFilter = {}): Promise<TransfersResponse> {
		const { limit = 50, offset = 0, block, token, from, to } = filter;
		return apiFetch<TransfersResponse>(
			`/api/v1/transfers${qs({ limit, offset, block, token, from, to })}`
		);
	}
}
