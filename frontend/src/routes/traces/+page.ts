import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTraces } from '$lib/stores/chain.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await fetchTraces({ limit: 50, offset: 0 });
	return {};
};
