// Dynamic route — fetched client-side. Skip prerendering.
import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchWallet } from '$lib/stores/wallet.svelte';

export const prerender = false;
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	abortAll();
	if (params.address) {
		// Don't await — let the page render immediately with loading state
		// while data fetches in the background
		fetchWallet(params.address, 50, 0);
	}
	return {};
};
