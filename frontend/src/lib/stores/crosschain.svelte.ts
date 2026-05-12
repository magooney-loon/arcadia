import { CrosschainCrudClient } from '../api/crosschain/crud.js';
import type { CrosschainResponse, CrosschainFilter } from '../api/crosschain/types.js';

const client = new CrosschainCrudClient();

export interface CrosschainState {
	data: CrosschainResponse | null;
	loading: boolean;
	error: string | null;
}

export const crosschain = $state<CrosschainState>({ data: null, loading: false, error: null });

export async function fetchCrosschain(filter: CrosschainFilter = {}) {
	crosschain.loading = true;
	crosschain.error = null;
	try {
		crosschain.data = await client.list(filter);
	} catch (e) {
		crosschain.error = String(e);
	} finally {
		crosschain.loading = false;
	}
}
