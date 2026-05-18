export type SearchType =
	| 'tx'
	| 'block'
	| 'wallet'
	| 'agent'
	| 'token'
	| 'multi'
	| 'not_found'
	| 'unknown';

export interface TokenHit {
	token_address: string;
	symbol: string;
	name: string;
	transfer_count: number;
	holder_count: number;
	id: string;
	created: string;
	updated: string;
}

export interface AgentHit {
	agent_address: string;
	tx_count: number;
	usdc_transferred: string;
	usdc_transferred_num: number;
	registered_at_block: number;
	id: string;
	created: string;
	updated: string;
}

export interface SearchResponse {
	type: SearchType;
	result?: Record<string, unknown>;
	tokens?: TokenHit[];
	agents?: AgentHit[];
}
