import { WalletCrudClient } from '../api/wallet/crud.js';
import type { WalletResponse } from '../api/wallet/types.js';

const client = new WalletCrudClient();

export interface WalletState {
	address: string;
	data: WalletResponse | null;
	loading: boolean;
	error: string | null;
}

export const wallet = $state<WalletState>({ address: '', data: null, loading: false, error: null });

export async function fetchWallet(address: string, limit = 50, offset = 0) {
	wallet.address = address;
	wallet.loading = true;
	wallet.error = null;
	try {
		wallet.data = await client.get(address, limit, offset);
	} catch (e) {
		wallet.error = String(e);
	} finally {
		wallet.loading = false;
	}
}
