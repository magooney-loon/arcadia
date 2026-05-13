import { ChainCrudClient } from '../api/chain/crud.js';
import type {
	BlocksResponse,
	TransactionsResponse,
	TracesResponse,
	TransactionFilter,
	TraceFilter,
	TxDetailResponse,
	BlockDetailResponse
} from '../api/chain/types.js';

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

export interface TxDetailState {
	hash: string;
	data: TxDetailResponse | null;
	loading: boolean;
	error: string | null;
}

export interface BlockDetailState {
	number: number | null;
	data: BlockDetailResponse | null;
	loading: boolean;
	error: string | null;
}

export const blocks = $state<BlocksState>({ data: null, loading: false, error: null });
export const transactions = $state<TransactionsState>({ data: null, loading: false, error: null });
export const traces = $state<TracesState>({ data: null, loading: false, error: null });
export const txDetail = $state<TxDetailState>({ hash: '', data: null, loading: false, error: null });
export const blockDetail = $state<BlockDetailState>({ number: null, data: null, loading: false, error: null });

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

export async function fetchTxDetail(hash: string) {
	txDetail.hash = hash;
	txDetail.loading = true;
	txDetail.error = null;
	txDetail.data = null;
	try {
		txDetail.data = await client.txDetail(hash);
	} catch (e) {
		txDetail.error = String(e);
	} finally {
		txDetail.loading = false;
	}
}

export async function fetchBlockDetail(number: number) {
	blockDetail.number = number;
	blockDetail.loading = true;
	blockDetail.error = null;
	blockDetail.data = null;
	try {
		blockDetail.data = await client.blockDetail(number);
	} catch (e) {
		blockDetail.error = String(e);
	} finally {
		blockDetail.loading = false;
	}
}
