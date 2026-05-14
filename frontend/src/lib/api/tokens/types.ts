export interface Token {
	id: string;
	token_address: string;
	symbol: string;
	name: string;
	decimals: number;
	total_supply_raw: string;
	total_supply_human: string;
	transfer_count: number;
	unique_senders: number;
	unique_receivers: number;
	first_seen_block: number;
	last_seen_block: number;
	lookup_failed: boolean;
}

export interface TokensResponse {
	tokens: Token[];
	count: number;
}

export interface TokenDetailResponse {
	token: Token;
	transfers: TokenTransfer[];
}

export interface TokenTransfer {
	id: string;
	tx_hash: string;
	block_number: number;
	log_index: number;
	token_address: string;
	token_symbol: string;
	token_name: string;
	decimals: number;
	from_addr: string;
	to_addr: string;
	amount_raw: string;
	amount_human: string;
}
