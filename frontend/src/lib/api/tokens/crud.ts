import { apiFetch } from '../utils.js';
import type { TokensResponse, TokenDetailResponse } from './types.js';

export class TokenCrudClient {
	list(limit = 50, offset = 0, search?: string): Promise<TokensResponse> {
		const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
		if (search) params.set('search', search);
		return apiFetch<TokensResponse>(`/api/v1/tokens?${params}`);
	}

	detail(address: string, limit = 50, offset = 0): Promise<TokenDetailResponse> {
		const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
		return apiFetch<TokenDetailResponse>(`/api/v1/tokens/${address}?${params}`);
	}
}
