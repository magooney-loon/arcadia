import { getApiUrl } from '../../stores/config.svelte.js';
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
} from './types.js';

function qs(params: Record<string, string | number | undefined | null>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== null && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

export class AnalyticsCrudClient {
	async fees(filter: FeesFilter = {}): Promise<FeesResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/analytics/fees${qs(filter as Record<string, string | number | undefined | null>)}`);
		if (!res.ok) throw new Error(`analytics/fees: ${res.status}`);
		return res.json();
	}

	async volume(filter: VolumeFilter = {}): Promise<VolumeResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/analytics/volume${qs(filter as Record<string, string | number | undefined | null>)}`);
		if (!res.ok) throw new Error(`analytics/volume: ${res.status}`);
		return res.json();
	}

	async bridgeFlow(filter: BridgeFlowFilter = {}): Promise<BridgeFlowResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/analytics/bridge_flow${qs(filter as Record<string, string | number | undefined | null>)}`);
		if (!res.ok) throw new Error(`analytics/bridge_flow: ${res.status}`);
		return res.json();
	}

	async agentLeaderboard(limit = 50): Promise<AgentLeaderboardResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/analytics/agent_leaderboard${qs({ limit })}`);
		if (!res.ok) throw new Error(`analytics/agent_leaderboard: ${res.status}`);
		return res.json();
	}

	async overview(filter: OverviewFilter = {}): Promise<OverviewResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/analytics/overview${qs(filter as Record<string, string | number | undefined | null>)}`);
		if (!res.ok) throw new Error(`analytics/overview: ${res.status}`);
		return res.json();
	}
}
