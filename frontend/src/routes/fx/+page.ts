import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchFx } from '$lib/stores/fx.svelte';

export const load: PageLoad = async () => {
	abortAll();
	// Don't await — let the page render immediately with loading state
	// while data fetches in the background
	fetchFx({ limit: 50, offset: 0 });
	return {};
};
