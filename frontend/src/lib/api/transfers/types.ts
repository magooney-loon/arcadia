export interface Transfer {
	id: string;
	block_number: number;
	log_index?: number;
	tx_hash: string;
	token_address: string;
	token_symbol: 'USDC' | 'EURC' | 'USYC' | 'OTHER';
	from_addr?: string;
	to_addr?: string;
	amount_raw?: string;
	amount_human?: string;
	[key: string]: unknown;
}

export interface TransfersResponse {
	transfers: Transfer[];
	count: number;
	total: number;
}

export interface TransferFilter {
	limit?: number;
	offset?: number;
	block?: string;
	token?: 'USDC' | 'EURC' | 'USYC' | 'OTHER';
	from?: string;
	to?: string;
}
