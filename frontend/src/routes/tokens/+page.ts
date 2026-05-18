import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTokens } from '$lib/stores/tokens.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchTokens(45, 0);
	return {};
};
