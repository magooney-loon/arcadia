import { apiFetch } from '../utils.js';
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
	// Wallet endpoint runs 7 concurrent queries server-side, so it's
	// the slowest single REST call we make — give it more headroom
	// than the default before timing out.
	get(address: string, limit = 50, offset = 0): Promise<WalletResponse> {
		return apiFetch<WalletResponse>(`/api/v1/wallet/${address}${qs({ limit, offset })}`, {}, 25_000);
	}
}
