import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchAgentJobs } from '$lib/stores/agents.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await fetchAgentJobs({ limit: 50, offset: 0 });
	return {};
};
