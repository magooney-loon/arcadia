export interface Block {
	id: string;
	number: number;
	hash: string;
	timestamp: string;
	tx_count: number;
	gas_used?: number;
	gas_limit?: number;
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
	from_addr: string;
	to_addr: string;
	value?: string;
	gas_used?: number;
	status?: string;
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
	from_addr: string;
	to_addr: string;
	call_type?: string;
	value?: string;
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
