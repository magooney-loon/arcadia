import { getApiUrl } from '../../stores/config.svelte.js';
import type {
	BlocksResponse,
	TransactionsResponse,
	TransactionFilter,
	TracesResponse,
	TraceFilter
} from './types.js';

function qs(params: Record<string, string | number | undefined>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

export class ChainCrudClient {
	async blocks(limit = 50, offset = 0): Promise<BlocksResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/blocks${qs({ limit, offset })}`);
		if (!res.ok) throw new Error(`blocks: ${res.status}`);
		return res.json();
	}

	async transactions(filter: TransactionFilter = {}): Promise<TransactionsResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		const res = await fetch(`${getApiUrl()}/api/v1/transactions${qs({ limit, offset, ...rest })}`);
		if (!res.ok) throw new Error(`transactions: ${res.status}`);
		return res.json();
	}

	async traces(filter: TraceFilter = {}): Promise<TracesResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		const res = await fetch(`${getApiUrl()}/api/v1/traces${qs({ limit, offset, ...rest })}`);
		if (!res.ok) throw new Error(`traces: ${res.status}`);
		return res.json();
	}
}
