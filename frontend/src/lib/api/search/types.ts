export type SearchType = 'tx' | 'block' | 'wallet' | 'agent' | 'not_found' | 'unknown';

export interface SearchResponse {
	type: SearchType;
	result?: Record<string, unknown>;
}
