import { getApiUrl } from '../../stores/config.svelte.js';
import type { SearchResponse } from './types.js';

export class SearchCrudClient {
	async search(q: string): Promise<SearchResponse> {
		const params = new URLSearchParams({ q });
		const res = await fetch(`${getApiUrl()}/api/v1/search?${params}`);
		if (!res.ok) throw new Error(`search: ${res.status}`);
		return res.json();
	}
}
