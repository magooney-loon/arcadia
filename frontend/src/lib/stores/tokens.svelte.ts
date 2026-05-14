import { TokenCrudClient } from '../api/tokens/crud.js';
import type { TokensResponse, TokenDetailResponse } from '../api/tokens/types.js';

const client = new TokenCrudClient();

export interface TokensState {
	data: TokensResponse | null;
	loading: boolean;
	error: string | null;
}

export const tokens = $state<TokensState>({ data: null, loading: false, error: null });

export async function fetchTokens(limit = 100, offset = 0, search?: string) {
	tokens.loading = true;
	tokens.error = null;
	try {
		tokens.data = await client.list(limit, offset, search);
	} catch (e) {
		tokens.error = String(e);
	} finally {
		tokens.loading = false;
	}
}

export interface TokenDetailState {
	data: TokenDetailResponse | null;
	loading: boolean;
	error: string | null;
}

export const tokenDetail = $state<TokenDetailState>({ data: null, loading: false, error: null });

export async function fetchTokenDetail(address: string, limit = 50, offset = 0) {
	tokenDetail.loading = true;
	tokenDetail.error = null;
	try {
		tokenDetail.data = await client.detail(address, limit, offset);
	} catch (e) {
		tokenDetail.error = String(e);
	} finally {
		tokenDetail.loading = false;
	}
}
