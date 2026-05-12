export interface CrosschainEvent {
	id: string;
	block_number: number;
	tx_hash: string;
	protocol: string;
	event_type: string;
	sender?: string;
	recipient?: string;
	source_domain?: number;
	destination_domain?: number;
	amount?: string;
	[key: string]: unknown;
}

export interface CrosschainResponse {
	events: CrosschainEvent[];
	count: number;
}

export interface CrosschainFilter {
	limit?: number;
	offset?: number;
	protocol?: string;
	event_type?: string;
	sender?: string;
	recipient?: string;
	direction?: 'inbound' | 'outbound';
}
