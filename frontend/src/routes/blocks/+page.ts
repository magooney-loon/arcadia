import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchBlocks } from '$lib/stores/chain.svelte';
import { fetchBlockStats } from '$lib/stores/blockStats.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await Promise.all([
		fetchBlocks(50, 0),
		fetchBlockStats(50, 0)
	]);
	return {};
};
