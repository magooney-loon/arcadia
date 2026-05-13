export interface Block {
	id: string;
	number: number;
	hash: string;
	parent_hash?: string;
	miner?: string;
	timestamp: number;
	gas_used?: number;
	gas_limit?: number;
	base_fee_per_gas?: string;
	size?: number;
	tx_count: number;
	block_time_ms?: number;
	utilization_pct?: number;
	[key: string]: unknown;
}

export interface BlocksResponse {
	blocks: Block[];
	count: number;
}

export interface Transaction {
	id: string;
	hash: string;
	block_number: number;
	transaction_index?: number;
	from_addr: string;
	to_addr?: string;
	value?: string;
	nonce?: number;
	sighash?: string;
	gas_price?: string;
	gas_limit?: number;
	gas_used?: number;
	cumulative_gas_used?: number;
	effective_gas_price?: string;
	max_fee_per_gas?: string;
	max_priority_fee_per_gas?: string;
	priority_fee_per_gas?: string;
	fee_usdc?: string;
	priority_fee_usdc?: string;
	tx_type?: number;
	status?: number;
	contract_address?: string;
	is_contract_deploy?: boolean;
	[key: string]: unknown;
}

export interface TransactionsResponse {
	transactions: Transaction[];
	count: number;
}

export interface TransactionFilter {
	limit?: number;
	offset?: number;
	block?: string;
	from?: string;
	to?: string;
}

export interface Trace {
	id: string;
	tx_hash: string;
	block_number: number;
	from_addr?: string;
	to_addr?: string;
	value?: string;
	call_type?: string;
	trace_type?: string;
	gas_used?: number;
	error_msg?: string;
	[key: string]: unknown;
}

export interface TracesResponse {
	traces: Trace[];
	count: number;
}

export interface TraceFilter {
	limit?: number;
	offset?: number;
	tx?: string;
	from?: string;
	to?: string;
}

export interface TxDetailResponse {
	transaction: Transaction;
	transfers: Record<string, unknown>[];
	traces: Trace[];
}

export interface BlockDetailResponse {
	block: Block;
	transactions: Transaction[];
	stats?: Record<string, unknown>;
}
