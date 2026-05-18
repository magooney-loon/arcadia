import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTransfers } from '$lib/stores/transfers.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchTransfers({ limit: 45, offset: 0 });
	return {};
};
