import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchAgentLeaderboard } from '$lib/stores/analytics.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await fetchAgentLeaderboard(50);
	return {};
};
