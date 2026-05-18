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

let reqSeq = 0;

export async function runSearch(q: string) {
	const myReq = ++reqSeq;
	search.query = q;
	search.loading = true;
	search.error = null;
	// keep previous data visible while loading — caller can animate the transition
	try {
		const data = await client.search(q);
		if (myReq !== reqSeq) return; // a newer search has superseded us
		search.data = data;
	} catch (e) {
		if (myReq !== reqSeq) return;
		search.error = String(e);
		search.data = null;
	} finally {
		if (myReq === reqSeq) search.loading = false;
	}
}

export function clearSearch() {
	search.query = '';
	search.data = null;
	search.error = null;
}
