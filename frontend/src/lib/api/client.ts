import PocketBase from 'pocketbase';
import { getPocketBaseInstance } from '../stores/config.svelte.js';
import { AuthCrudClient } from './auth/crud.js';
import { StatsCrudClient } from './stats/crud.js';
import { BlockStatsCrudClient } from './block_stats/crud.js';
import { ChainCrudClient } from './chain/crud.js';
import { TransfersCrudClient } from './transfers/crud.js';
import { WalletCrudClient } from './wallet/crud.js';
import { CrosschainCrudClient } from './crosschain/crud.js';
import { FxCrudClient } from './fx/crud.js';
import { AgentsCrudClient } from './agents/crud.js';
import { GraphCrudClient } from './graph/crud.js';

export class ApiClient {
	private pb: PocketBase;
	private _auth: AuthCrudClient;
	private _stats: StatsCrudClient;
	private _blockStats: BlockStatsCrudClient;
	private _chain: ChainCrudClient;
	private _transfers: TransfersCrudClient;
	private _wallet: WalletCrudClient;
	private _crosschain: CrosschainCrudClient;
	private _fx: FxCrudClient;
	private _agents: AgentsCrudClient;
	private _graph: GraphCrudClient;

	constructor() {
		this.pb = getPocketBaseInstance();
		this._auth = new AuthCrudClient(this.pb);
		this._stats = new StatsCrudClient();
		this._blockStats = new BlockStatsCrudClient();
		this._chain = new ChainCrudClient();
		this._transfers = new TransfersCrudClient();
		this._wallet = new WalletCrudClient();
		this._crosschain = new CrosschainCrudClient();
		this._fx = new FxCrudClient();
		this._agents = new AgentsCrudClient();
		this._graph = new GraphCrudClient();
	}

	getPocketBase(): PocketBase { return this.pb; }

	get auth() { return this._auth; }
	get stats() { return this._stats; }
	get blockStats() { return this._blockStats; }
	get chain() { return this._chain; }
	get transfers() { return this._transfers; }
	get wallet() { return this._wallet; }
	get crosschain() { return this._crosschain; }
	get fx() { return this._fx; }
	get agents() { return this._agents; }
	get graph() { return this._graph; }
}
