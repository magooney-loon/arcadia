// Dynamic route — fetched client-side. Skip prerendering.
import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTxDetail } from '$lib/stores/chain.svelte';

export const prerender = false;
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	abortAll();
	if (params.hash) {
		await fetchTxDetail(params.hash);
	}
	return {};
};
