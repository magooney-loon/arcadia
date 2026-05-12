import { getApiUrl } from '../../stores/config.svelte.js';
import type { AgentsResponse, AgentResponse, AgentJobsResponse, AgentJobsFilter } from './types.js';

function qs(params: Record<string, string | number | undefined>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

export class AgentsCrudClient {
	async list(limit = 50, offset = 0): Promise<AgentsResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/agents${qs({ limit, offset })}`);
		if (!res.ok) throw new Error(`agents: ${res.status}`);
		return res.json();
	}

	async get(address: string): Promise<AgentResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/agents/${address}`);
		if (!res.ok) throw new Error(`agent: ${res.status}`);
		return res.json();
	}

	async jobs(filter: AgentJobsFilter = {}): Promise<AgentJobsResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		const res = await fetch(`${getApiUrl()}/api/v1/jobs${qs({ limit, offset, ...rest })}`);
		if (!res.ok) throw new Error(`jobs: ${res.status}`);
		return res.json();
	}
}
