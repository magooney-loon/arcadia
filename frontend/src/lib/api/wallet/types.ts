export interface WalletEdge {
	id: string;
	from_wallet: string;
	to_wallet: string;
	tx_count: number;
	[key: string]: unknown;
}

export interface AgentRecord {
	id: string;
	agent_address: string;
	registered_at_block: number;
	[key: string]: unknown;
}

export interface WalletResponse {
	address: string;
	is_agent: boolean;
	agent: AgentRecord | null;
	txs_sent: Record<string, unknown>[];
	txs_received: Record<string, unknown>[];
	sent: Record<string, unknown>[];
	received: Record<string, unknown>[];
	outgoing_edges: WalletEdge[];
	incoming_edges: WalletEdge[];
}
