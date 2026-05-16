import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTransactions } from '$lib/stores/chain.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchTransactions({ limit: 100, offset: 0 });
	return {};
};
