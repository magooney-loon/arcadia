import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchStats } from '$lib/stores/stats.svelte';
import { fetchHealth } from '$lib/stores/health.svelte';
import { fetchBlockStats } from '$lib/stores/blockStats.svelte';
import { fetchBlocks, fetchTransactions, fetchTraces } from '$lib/stores/chain.svelte';
import { fetchTransfers } from '$lib/stores/transfers.svelte';
import { fetchCrosschain } from '$lib/stores/crosschain.svelte';
import { fetchFx } from '$lib/stores/fx.svelte';
import { fetchAgents, fetchAgentJobs } from '$lib/stores/agents.svelte';
import { fetchEdges } from '$lib/stores/graph.svelte';
import { fetchTokens } from '$lib/stores/tokens.svelte';
import {
	fetchAnalyticsOverview,
	fetchAnalyticsFees,
	fetchAnalyticsVolume,
	fetchAnalyticsBridgeFlow,
	fetchAgentLeaderboard
} from '$lib/stores/analytics.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await Promise.all([
		fetchStats(),
		fetchHealth(),
		fetchBlockStats(50),
		fetchBlocks(50),
		fetchTransactions(),
		fetchTraces(),
		fetchTransfers(),
		fetchCrosschain(),
		fetchFx(),
		fetchAgents(),
		fetchAgentJobs(),
		fetchEdges(),
		fetchTokens(),
		fetchAnalyticsOverview(),
		fetchAnalyticsFees(),
		fetchAnalyticsVolume(),
		fetchAnalyticsBridgeFlow(),
		fetchAgentLeaderboard()
	]);
	return {};
};
