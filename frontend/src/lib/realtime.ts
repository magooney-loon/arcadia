// SSE subscription manager. Subscribes to the server's two custom
// PocketBase topics ("indexer" and "analytics") and fans incoming
// payloads into the existing Svelte stores so any view bound to those
// stores updates automatically.
//
// The PocketBase JS SDK manages the SSE connection lifecycle for us:
// auto-reconnect, clientId handshake, re-subscription on reconnect.
// We just need to call subscribe()/unsubscribe().

import { pb } from '$lib/stores/config.svelte';
import { stats } from '$lib/stores/stats.svelte';
import { health } from '$lib/stores/health.svelte';
import { blocks, transactions } from '$lib/stores/chain.svelte';
import { blockStats } from '$lib/stores/blockStats.svelte';
import {
	analyticsOverview,
	analyticsBridgeFlow,
	analyticsVolume
} from '$lib/stores/analytics.svelte';
import type { StatsResponse } from '$lib/api/stats/types.js';
import type { HealthResponse } from '$lib/api/health/types.js';
import type { BlocksResponse, TransactionsResponse } from '$lib/api/chain/types.js';
import type { BlockStatsResponse } from '$lib/api/block_stats/types.js';
import type {
	OverviewResponse,
	BridgeFlowResponse,
	VolumeResponse
} from '$lib/api/analytics/types.js';

interface IndexerPayload {
	stats: StatsResponse;
	health: HealthResponse;
	blocks: BlocksResponse;
	transactions: TransactionsResponse;
}

interface ChartsPayload {
	block_stats: BlockStatsResponse;
}

interface AnalyticsPayload {
	window: string;
	overview: OverviewResponse;
	bridge_flow: BridgeFlowResponse;
	volume: VolumeResponse;
}

// Window currently selected by the UI. The server broadcasts all three
// windows (1h/24h/7d) every snapshot tick; we only apply the one the
// user is viewing.
let activeWindow: string = '24h';

export function setRealtimeWindow(window: string) {
	activeWindow = window;
}

export async function connectRealtime() {
	await pb.realtime.subscribe('indexer', (e: unknown) => {
		const p = e as IndexerPayload;
		if (p.stats) stats.data = p.stats;
		if (p.health) health.data = p.health;
		if (p.blocks) blocks.data = p.blocks;
		if (p.transactions) transactions.data = p.transactions;
	});

	await pb.realtime.subscribe('analytics', (e: unknown) => {
		const p = e as AnalyticsPayload;
		if (p.window !== activeWindow) return;
		if (p.overview) analyticsOverview.data = p.overview;
		if (p.bridge_flow) analyticsBridgeFlow.data = p.bridge_flow;
		if (p.volume) analyticsVolume.data = p.volume;
	});
}

export async function disconnectRealtime() {
	await pb.realtime.unsubscribe('indexer');
	await pb.realtime.unsubscribe('analytics');
}

// The `charts` topic carries the 200-row block_stats series. It's only
// useful to pages that render charts (overview), so it's a separate
// subscription with its own lifecycle — view-bound, not app-bound.
export async function connectCharts() {
	await pb.realtime.subscribe('charts', (e: unknown) => {
		const p = e as ChartsPayload;
		if (p.block_stats) blockStats.data = p.block_stats;
	});
}

export async function disconnectCharts() {
	await pb.realtime.unsubscribe('charts');
}
