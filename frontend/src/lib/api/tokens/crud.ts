import { getApiUrl } from '../../stores/config.svelte.js';
import type { TokensResponse, TokenDetailResponse } from './types.js';

export class TokenCrudClient {
	async list(limit = 50, offset = 0, search?: string): Promise<TokensResponse> {
		const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
		if (search) params.set('search', search);
		const res = await fetch(`${getApiUrl()}/api/v1/tokens?${params}`);
		if (!res.ok) throw new Error(`tokens: ${res.status}`);
		return res.json();
	}

	async detail(address: string, limit = 50, offset = 0): Promise<TokenDetailResponse> {
		const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
		const res = await fetch(`${getApiUrl()}/api/v1/tokens/${address}?${params}`);
		if (!res.ok) throw new Error(`token detail: ${res.status}`);
		return res.json();
	}
}
