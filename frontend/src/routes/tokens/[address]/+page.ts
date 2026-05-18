// Dynamic route — fetched client-side. Skip prerendering.
import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchTokenDetail } from '$lib/stores/tokens.svelte';

export const prerender = false;
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	abortAll();
	const address = params.address;
	if (address) {
		// Don't await — let the page render immediately with loading state
		// while data fetches in the background
		fetchTokenDetail(address);
	}
	return {};
};
