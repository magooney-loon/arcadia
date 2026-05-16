import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchEdges } from '$lib/stores/graph.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchEdges({ limit: 500, offset: 0 });
	return {};
};
