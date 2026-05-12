import type { Transaction } from '../chain/types.js';
import type { Transfer } from '../transfers/types.js';

export interface WalletEdge {
	id: string;
	from_wallet: string;
	to_wallet: string;
	total_usdc?: string;
	tx_count: number;
	last_seen_block?: number;
	from_is_agent?: boolean;
	to_is_agent?: boolean;
	[key: string]: unknown;
}

export interface AgentRecord {
	id: string;
	agent_address: string;
	metadata_uri?: string;
	registered_at_block: number;
	tx_hash?: string;
	tx_count?: number;
	usdc_spent_fees?: string;
	usdc_transferred?: string;
	[key: string]: unknown;
}

export interface WalletResponse {
	address: string;
	is_agent: boolean;
	agent: AgentRecord | null;
	txs_sent: Transaction[];
	txs_received: Transaction[];
	sent: Transfer[];
	received: Transfer[];
	outgoing_edges: WalletEdge[];
	incoming_edges: WalletEdge[];
}
