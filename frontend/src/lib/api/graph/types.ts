export interface Edge {
	id: string;
	from_wallet: string;
	to_wallet: string;
	total_usdc?: string;
	total_usdc_human?: string;
	tx_count: number;
	last_seen_block?: number;
	from_is_agent?: boolean;
	to_is_agent?: boolean;
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
