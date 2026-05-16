// Dynamic route — fetched client-side. Skip prerendering.
import type { PageLoad } from './$types';
import { abortAll } from '$lib/api/utils';
import { fetchWallet } from '$lib/stores/wallet.svelte';

export const prerender = false;
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	abortAll();
	if (params.address) {
		await fetchWallet(params.address, 50, 0);
	}
	return {};
};
