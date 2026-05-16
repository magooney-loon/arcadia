// Dynamic route — fetched client-side. Skip prerendering.
import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchBlockDetail } from '$lib/stores/chain.svelte';

export const prerender = false;
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	abortAll();
	const number = Number(params.number);
	if (!isNaN(number)) {
		await fetchBlockDetail(number);
	}
	return {};
};
