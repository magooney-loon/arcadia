export interface Edge {
	id: string;
	from_wallet: string;
	to_wallet: string;
	tx_count: number;
	[key: string]: unknown;
}

export interface EdgesResponse {
	edges: Edge[];
	count: number;
}

export interface EdgeFilter {
	limit?: number;
	offset?: number;
	wallet?: string;
}
