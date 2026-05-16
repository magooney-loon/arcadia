import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchCrosschain } from '$lib/stores/crosschain.svelte';
import { fetchAnalyticsBridgeFlow } from '$lib/stores/analytics.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchCrosschain({ limit: 50, offset: 0 });
	fetchAnalyticsBridgeFlow();
	return {};
};
