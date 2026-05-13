import { SearchCrudClient } from '../api/search/crud.js';
import type { SearchResponse } from '../api/search/types.js';

const client = new SearchCrudClient();

export interface SearchState {
	query: string;
	data: SearchResponse | null;
	loading: boolean;
	error: string | null;
}

export const search = $state<SearchState>({ query: '', data: null, loading: false, error: null });

export async function runSearch(q: string) {
	search.query = q;
	search.loading = true;
	search.error = null;
	search.data = null;
	try {
		search.data = await client.search(q);
	} catch (e) {
		search.error = String(e);
	} finally {
		search.loading = false;
	}
}

export function clearSearch() {
	search.query = '';
	search.data = null;
	search.error = null;
}
