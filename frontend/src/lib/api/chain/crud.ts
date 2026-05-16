import { apiFetch } from '../utils.js';
import type {
	BlocksResponse,
	TransactionsResponse,
	TransactionFilter,
	TracesResponse,
	TraceFilter,
	TxDetailResponse,
	BlockDetailResponse
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
	blocks(limit = 50, offset = 0): Promise<BlocksResponse> {
		return apiFetch<BlocksResponse>(`/api/v1/blocks${qs({ limit, offset })}`);
	}

	transactions(filter: TransactionFilter = {}): Promise<TransactionsResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		return apiFetch<TransactionsResponse>(`/api/v1/transactions${qs({ limit, offset, ...rest })}`);
	}

	traces(filter: TraceFilter = {}): Promise<TracesResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		return apiFetch<TracesResponse>(`/api/v1/traces${qs({ limit, offset, ...rest })}`);
	}

	txDetail(hash: string): Promise<TxDetailResponse> {
		return apiFetch<TxDetailResponse>(`/api/v1/tx/${encodeURIComponent(hash)}`);
	}

	blockDetail(number: number): Promise<BlockDetailResponse> {
		return apiFetch<BlockDetailResponse>(`/api/v1/block/${number}`);
	}
}
