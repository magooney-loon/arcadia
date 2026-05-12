export interface Transfer {
	id: string;
	block_number: number;
	tx_hash: string;
	from_addr: string;
	to_addr: string;
	token_symbol: string;
	amount?: string;
	[key: string]: unknown;
}

export interface TransfersResponse {
	transfers: Transfer[];
	count: number;
}

export interface TransferFilter {
	limit?: number;
	offset?: number;
	token?: string;
	from?: string;
	to?: string;
}
