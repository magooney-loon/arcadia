import { apiFetch } from '../utils.js';
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
	list(limit = 50, offset = 0): Promise<AgentsResponse> {
		return apiFetch<AgentsResponse>(`/api/v1/agents${qs({ limit, offset })}`);
	}

	get(address: string): Promise<AgentResponse> {
		return apiFetch<AgentResponse>(`/api/v1/agents/${address}`);
	}

	jobs(filter: AgentJobsFilter = {}): Promise<AgentJobsResponse> {
		const { limit = 50, offset = 0, ...rest } = filter;
		return apiFetch<AgentJobsResponse>(`/api/v1/jobs${qs({ limit, offset, ...rest })}`);
	}
}
