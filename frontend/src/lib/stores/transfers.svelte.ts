import { TransfersCrudClient } from '../api/transfers/crud.js';
import type { TransfersResponse, TransferFilter } from '../api/transfers/types.js';

const client = new TransfersCrudClient();

export interface TransfersState {
	data: TransfersResponse | null;
	loading: boolean;
	error: string | null;
}

export const transfers = $state<TransfersState>({ data: null, loading: false, error: null });

export async function fetchTransfers(filter: TransferFilter = {}) {
	transfers.loading = true;
	transfers.error = null;
	try {
		transfers.data = await client.list(filter);
	} catch (e) {
		transfers.error = String(e);
	} finally {
		transfers.loading = false;
	}
}
