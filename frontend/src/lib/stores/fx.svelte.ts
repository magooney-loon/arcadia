import { FxCrudClient } from '../api/fx/crud.js';
import type { FxResponse, FxFilter } from '../api/fx/types.js';

const client = new FxCrudClient();

export interface FxState {
	data: FxResponse | null;
	loading: boolean;
	error: string | null;
}

export const fx = $state<FxState>({ data: null, loading: false, error: null });

export async function fetchFx(filter: FxFilter = {}) {
	fx.loading = true;
	fx.error = null;
	try {
		fx.data = await client.list(filter);
	} catch (e) {
		if (!String(e).includes('cancelled')) fx.error = String(e);
	} finally {
		fx.loading = false;
	}
}
