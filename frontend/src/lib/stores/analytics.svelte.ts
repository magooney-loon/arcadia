import { AnalyticsCrudClient } from '../api/analytics/crud.js';
import type {
	FeesFilter,
	FeesResponse,
	VolumeFilter,
	VolumeResponse,
	BridgeFlowFilter,
	BridgeFlowResponse,
	AgentLeaderboardResponse,
	OverviewFilter,
	OverviewResponse
} from '../api/analytics/types.js';

const client = new AnalyticsCrudClient();

export interface FeesState {
	data: FeesResponse | null;
	loading: boolean;
	error: string | null;
}

export interface VolumeState {
	data: VolumeResponse | null;
	loading: boolean;
	error: string | null;
}

export interface BridgeFlowState {
	data: BridgeFlowResponse | null;
	loading: boolean;
	error: string | null;
}

export interface AgentLeaderboardState {
	data: AgentLeaderboardResponse | null;
	loading: boolean;
	error: string | null;
}

export interface OverviewState {
	data: OverviewResponse | null;
	loading: boolean;
	error: string | null;
}

export const analyticsOverview = $state<OverviewState>({ data: null, loading: false, error: null });
export const analyticsFees = $state<FeesState>({ data: null, loading: false, error: null });
export const analyticsVolume = $state<VolumeState>({ data: null, loading: false, error: null });
export const analyticsBridgeFlow = $state<BridgeFlowState>({ data: null, loading: false, error: null });
export const analyticsAgentLeaderboard = $state<AgentLeaderboardState>({ data: null, loading: false, error: null });

export async function fetchAnalyticsFees(filter: FeesFilter = {}) {
	analyticsFees.loading = true;
	analyticsFees.error = null;
	try {
		analyticsFees.data = await client.fees(filter);
	} catch (e) {
		analyticsFees.error = String(e);
	} finally {
		analyticsFees.loading = false;
	}
}

export async function fetchAnalyticsVolume(filter: VolumeFilter = {}) {
	analyticsVolume.loading = true;
	analyticsVolume.error = null;
	try {
		analyticsVolume.data = await client.volume(filter);
	} catch (e) {
		analyticsVolume.error = String(e);
	} finally {
		analyticsVolume.loading = false;
	}
}

export async function fetchAnalyticsBridgeFlow(filter: BridgeFlowFilter = {}) {
	analyticsBridgeFlow.loading = true;
	analyticsBridgeFlow.error = null;
	try {
		analyticsBridgeFlow.data = await client.bridgeFlow(filter);
	} catch (e) {
		analyticsBridgeFlow.error = String(e);
	} finally {
		analyticsBridgeFlow.loading = false;
	}
}

export async function fetchAnalyticsOverview(filter: OverviewFilter = {}) {
	analyticsOverview.loading = true;
	analyticsOverview.error = null;
	try {
		analyticsOverview.data = await client.overview(filter);
	} catch (e) {
		analyticsOverview.error = String(e);
	} finally {
		analyticsOverview.loading = false;
	}
}

export async function fetchAgentLeaderboard(limit = 50) {
	analyticsAgentLeaderboard.loading = true;
	analyticsAgentLeaderboard.error = null;
	try {
		analyticsAgentLeaderboard.data = await client.agentLeaderboard(limit);
	} catch (e) {
		analyticsAgentLeaderboard.error = String(e);
	} finally {
		analyticsAgentLeaderboard.loading = false;
	}
}
