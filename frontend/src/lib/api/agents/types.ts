export interface Agent {
	id: string;
	agent_address: string;
	metadata_uri?: string;
	registered_at_block: number;
	tx_hash?: string;
	tx_count?: number;
	usdc_spent_fees?: string;
	usdc_transferred?: string;
	[key: string]: unknown;
}

export interface AgentsResponse {
	agents: Agent[];
	count: number;
}

export interface AgentJob {
	id: string;
	job_id: string;
	employer_address?: string;
	worker_address?: string;
	payment_usdc?: string;
	status: 'created' | 'accepted' | 'delivered' | 'settled' | 'disputed';
	created_at_block: number;
	settled_at_block?: number;
	create_tx_hash?: string;
	settle_tx_hash?: string;
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
