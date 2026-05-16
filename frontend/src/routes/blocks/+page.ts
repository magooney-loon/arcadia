import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchBlocks } from '$lib/stores/chain.svelte';
import { fetchBlockStats } from '$lib/stores/blockStats.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchBlocks(50, 0);
	fetchBlockStats(50, 0);
	return {};
};
