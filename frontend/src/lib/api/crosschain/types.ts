export interface CrosschainEvent {
	id: string;
	block_number: number;
	log_index?: number;
	tx_hash: string;
	protocol: 'cctp' | 'gateway';
	event_type: 'burn' | 'mint' | 'deposit' | 'withdraw';
	source_domain?: number;
	destination_domain?: number;
	sender?: string;
	recipient?: string;
	amount_usdc?: string;
	nonce_val?: string;
	[key: string]: unknown;
}

export interface CrosschainResponse {
	events: CrosschainEvent[];
	count: number;
	total: number;
}

export interface CrosschainFilter {
	limit?: number;
	offset?: number;
	protocol?: 'cctp' | 'gateway';
	event_type?: 'burn' | 'mint' | 'deposit' | 'withdraw';
	sender?: string;
	recipient?: string;
	direction?: 'inbound' | 'outbound';
	chain?: number;
}
