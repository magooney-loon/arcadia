import { apiFetch } from '../utils.js';
import type { SearchResponse } from './types.js';

export class SearchCrudClient {
	search(q: string): Promise<SearchResponse> {
		const params = new URLSearchParams({ q });
		// Search is interactive — keep timeout shorter so a stuck query
		// fails fast and the user can keep typing.
		return apiFetch<SearchResponse>(`/api/v1/search?${params}`, {}, 8_000);
	}
}
