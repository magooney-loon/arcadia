import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchEdges } from '$lib/stores/graph.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await fetchEdges({ limit: 500, offset: 0 });
	return {};
};
