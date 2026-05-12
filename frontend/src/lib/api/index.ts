// ==================== API CLIENT ====================
export { ApiClient } from './client.js';

// ==================== API UTILITIES ====================
export { formatTimestamp } from './utils.js';

// ==================== AUTH ====================
export type {
	LoginRequest,
	RegisterRequest,
	AuthUser,
	LoginResponse,
	RegisterResponse,
	PasswordResetRequest,
	PasswordResetResponse,
	EmailVerificationRequest,
	EmailVerificationResponse
} from './auth/types.js';

// ==================== STATS ====================
export type { StatsResponse } from './stats/types.js';
export { StatsCrudClient } from './stats/crud.js';

// ==================== CHAIN ====================
export type {
	Block,
	BlocksResponse,
	Transaction,
	TransactionsResponse,
	TransactionFilter,
	Trace,
	TracesResponse,
	TraceFilter
} from './chain/types.js';
export { ChainCrudClient } from './chain/crud.js';

// ==================== TRANSFERS ====================
export type { Transfer, TransfersResponse, TransferFilter } from './transfers/types.js';
export { TransfersCrudClient } from './transfers/crud.js';

// ==================== WALLET ====================
export type { WalletResponse, WalletEdge, AgentRecord } from './wallet/types.js';
export { WalletCrudClient } from './wallet/crud.js';

// ==================== CROSSCHAIN ====================
export type { CrosschainEvent, CrosschainResponse, CrosschainFilter } from './crosschain/types.js';
export { CrosschainCrudClient } from './crosschain/crud.js';

// ==================== FX ====================
export type { FxTrade, FxResponse, FxFilter } from './fx/types.js';
export { FxCrudClient } from './fx/crud.js';

// ==================== AGENTS ====================
export type {
	Agent,
	AgentsResponse,
	AgentJob,
	AgentResponse,
	AgentJobsResponse,
	AgentJobsFilter
} from './agents/types.js';
export { AgentsCrudClient } from './agents/crud.js';

// ==================== GRAPH ====================
export type { Edge, EdgesResponse, EdgeFilter } from './graph/types.js';
export { GraphCrudClient } from './graph/crud.js';
