import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchFx } from '$lib/stores/fx.svelte';

export const load: PageLoad = async () => {
	abortAll();
	await fetchFx({ limit: 50, offset: 0 });
	return {};
};
