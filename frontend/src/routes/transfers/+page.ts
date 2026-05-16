import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTransfers } from '$lib/stores/transfers.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await fetchTransfers({ limit: 50, offset: 0 });
	return {};
};
