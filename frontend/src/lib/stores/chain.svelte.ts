import { ChainCrudClient } from '../api/chain/crud.js';
import type { BlocksResponse, TransactionsResponse, TracesResponse, TransactionFilter, TraceFilter } from '../api/chain/types.js';

const client = new ChainCrudClient();

export interface BlocksState {
	data: BlocksResponse | null;
	loading: boolean;
	error: string | null;
}

export interface TransactionsState {
	data: TransactionsResponse | null;
	loading: boolean;
	error: string | null;
}

export interface TracesState {
	data: TracesResponse | null;
	loading: boolean;
	error: string | null;
}

export const blocks = $state<BlocksState>({ data: null, loading: false, error: null });
export const transactions = $state<TransactionsState>({ data: null, loading: false, error: null });
export const traces = $state<TracesState>({ data: null, loading: false, error: null });

export async function fetchBlocks(limit = 50, offset = 0) {
	blocks.loading = true;
	blocks.error = null;
	try {
		blocks.data = await client.blocks(limit, offset);
	} catch (e) {
		blocks.error = String(e);
	} finally {
		blocks.loading = false;
	}
}

export async function fetchTransactions(filter: TransactionFilter = {}) {
	transactions.loading = true;
	transactions.error = null;
	try {
		transactions.data = await client.transactions(filter);
	} catch (e) {
		transactions.error = String(e);
	} finally {
		transactions.loading = false;
	}
}

export async function fetchTraces(filter: TraceFilter = {}) {
	traces.loading = true;
	traces.error = null;
	try {
		traces.data = await client.traces(filter);
	} catch (e) {
		traces.error = String(e);
	} finally {
		traces.loading = false;
	}
}
