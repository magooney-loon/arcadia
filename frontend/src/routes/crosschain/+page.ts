import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchCrosschain } from '$lib/stores/crosschain.svelte';
import { fetchAnalyticsBridgeFlow } from '$lib/stores/analytics.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await Promise.all([
		fetchCrosschain({ limit: 50, offset: 0 }),
		fetchAnalyticsBridgeFlow()
	]);
	return {};
};
