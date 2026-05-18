import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchAgentLeaderboard } from '$lib/stores/analytics.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchAgentLeaderboard(40, 0);
	return {};
};
