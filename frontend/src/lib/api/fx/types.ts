export interface FxTrade {
	id: string;
	block_number: number;
	tx_hash: string;
	quote_id: string;
	maker: string;
	taker?: string;
	status: string;
	[key: string]: unknown;
}

export interface FxResponse {
	trades: FxTrade[];
	count: number;
}

export interface FxFilter {
	limit?: number;
	offset?: number;
	status?: string;
	maker?: string;
	taker?: string;
	quote_id?: string;
}
