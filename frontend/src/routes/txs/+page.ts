import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTransactions } from '$lib/stores/chain.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await fetchTransactions({ limit: 100, offset: 0 });
	return {};
};
