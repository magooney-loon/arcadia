import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchStats } from '$lib/stores/stats.svelte';
import { fetchBlocks, fetchTransactions } from '$lib/stores/chain.svelte';
import { fetchBlockStats } from '$lib/stores/blockStats.svelte';
import {
	fetchAnalyticsOverview,
	fetchAnalyticsBridgeFlow,
	fetchAnalyticsVolume,
	fetchAgentLeaderboard
} from '$lib/stores/analytics.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchStats();
	fetchBlocks(10);
	fetchTransactions({ limit: 10 });
	fetchBlockStats(50);
	fetchAnalyticsOverview({ window: '24h' });
	fetchAnalyticsBridgeFlow({ window: '24h' });
	fetchAnalyticsVolume({ window: '24h' });
	fetchAgentLeaderboard(5);
	return {};
};
