import { apiFetch } from '../utils.js';
import type {
	FeesFilter,
	FeesResponse,
	VolumeFilter,
	VolumeResponse,
	BridgeFlowFilter,
	BridgeFlowResponse,
	AgentLeaderboardResponse,
	OverviewFilter,
	OverviewResponse,
	HistoryFilter,
	HistoryResponse
} from './types.js';

function qs(params: Record<string, string | number | undefined | null>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== null && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

type AnyFilter = Record<string, string | number | undefined | null>;

export class AnalyticsCrudClient {
	fees(filter: FeesFilter = {}): Promise<FeesResponse> {
		return apiFetch<FeesResponse>(`/api/v1/analytics/fees${qs(filter as AnyFilter)}`);
	}

	volume(filter: VolumeFilter = {}): Promise<VolumeResponse> {
		return apiFetch<VolumeResponse>(`/api/v1/analytics/volume${qs(filter as AnyFilter)}`);
	}

	bridgeFlow(filter: BridgeFlowFilter = {}): Promise<BridgeFlowResponse> {
		return apiFetch<BridgeFlowResponse>(`/api/v1/analytics/bridge_flow${qs(filter as AnyFilter)}`);
	}

	agentLeaderboard(limit = 50): Promise<AgentLeaderboardResponse> {
		return apiFetch<AgentLeaderboardResponse>(`/api/v1/analytics/agent_leaderboard${qs({ limit })}`);
	}

	overview(filter: OverviewFilter = {}): Promise<OverviewResponse> {
		return apiFetch<OverviewResponse>(`/api/v1/analytics/overview${qs(filter as AnyFilter)}`);
	}

	history(filter: HistoryFilter = {}): Promise<HistoryResponse> {
		return apiFetch<HistoryResponse>(`/api/v1/analytics/history${qs(filter as AnyFilter)}`);
	}
}
