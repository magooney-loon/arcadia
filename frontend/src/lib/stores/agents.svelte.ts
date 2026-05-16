import { AgentsCrudClient } from '../api/agents/crud.js';
import type {
	AgentsResponse,
	AgentResponse,
	AgentJobsResponse,
	AgentJobsFilter
} from '../api/agents/types.js';

const client = new AgentsCrudClient();

export interface AgentsState {
	data: AgentsResponse | null;
	loading: boolean;
	error: string | null;
}

export interface AgentState {
	address: string;
	data: AgentResponse | null;
	loading: boolean;
	error: string | null;
}

export interface AgentJobsState {
	data: AgentJobsResponse | null;
	loading: boolean;
	error: string | null;
}

export const agents = $state<AgentsState>({ data: null, loading: false, error: null });
export const agent = $state<AgentState>({ address: '', data: null, loading: false, error: null });
export const agentJobs = $state<AgentJobsState>({ data: null, loading: false, error: null });

export async function fetchAgents(limit = 50, offset = 0) {
	agents.loading = true;
	agents.error = null;
	try {
		agents.data = await client.list(limit, offset);
	} catch (e) {
		if (!String(e).includes('cancelled')) agents.error = String(e);
	} finally {
		agents.loading = false;
	}
}

export async function fetchAgent(address: string) {
	agent.address = address;
	agent.loading = true;
	agent.error = null;
	try {
		agent.data = await client.get(address);
	} catch (e) {
		if (!String(e).includes('cancelled')) agent.error = String(e);
	} finally {
		agent.loading = false;
	}
}

export async function fetchAgentJobs(filter: AgentJobsFilter = {}) {
	agentJobs.loading = true;
	agentJobs.error = null;
	try {
		agentJobs.data = await client.jobs(filter);
	} catch (e) {
		if (!String(e).includes('cancelled')) agentJobs.error = String(e);
	} finally {
		agentJobs.loading = false;
	}
}
