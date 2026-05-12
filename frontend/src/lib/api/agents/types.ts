export interface Agent {
	id: string;
	agent_address: string;
	registered_at_block: number;
	[key: string]: unknown;
}

export interface AgentsResponse {
	agents: Agent[];
	count: number;
}

export interface AgentJob {
	id: string;
	employer_address: string;
	worker_address: string;
	status: string;
	created_at_block: number;
	[key: string]: unknown;
}

export interface AgentResponse {
	agent: Agent;
	jobs: AgentJob[];
}

export interface AgentJobsResponse {
	jobs: AgentJob[];
	count: number;
}

export interface AgentJobsFilter {
	limit?: number;
	offset?: number;
	status?: string;
	employer?: string;
	worker?: string;
}
