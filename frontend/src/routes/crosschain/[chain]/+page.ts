import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchCrosschain } from '$lib/stores/crosschain.svelte';
import { fetchAnalyticsBridgeFlow } from '$lib/stores/analytics.svelte';

export const prerender = false;
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	abortAll();
	const chainId = Number(params.chain);
	if (!isNaN(chainId)) {
		fetchCrosschain({ chain: chainId, limit: 50 });
		fetchAnalyticsBridgeFlow();
	}
	return {};
};
