import { getApiUrl } from '../../stores/config.svelte.js';
import type { WalletResponse } from './types.js';

function qs(params: Record<string, string | number | undefined>): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== '') p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

export class WalletCrudClient {
	async get(address: string, limit = 50, offset = 0): Promise<WalletResponse> {
		const res = await fetch(`${getApiUrl()}/api/v1/wallet/${address}${qs({ limit, offset })}`);
		if (!res.ok) throw new Error(`wallet: ${res.status}`);
		return res.json();
	}
}
