export interface FxTrade {
	id: string;
	trade_id: string;
	quote_id?: string;
	block_number: number;
	tx_hash?: string;
	maker?: string;
	taker?: string;
	taker_fee?: string;
	maker_fee?: string;
	status_code?: number;
	status: 'created' | 'taker_funded' | 'maker_funded' | 'settled' | 'cancelled';
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
