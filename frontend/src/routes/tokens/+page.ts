import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTokens } from '$lib/stores/tokens.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await fetchTokens(500);
	return {};
};
