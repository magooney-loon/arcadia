<script lang="ts">
	import { stats, fetchStats } from '$lib/stores/stats.svelte';
	import { blockStats, fetchBlockStats } from '$lib/stores/blockStats.svelte';
	import {
		blocks,
		transactions,
		traces,
		txDetail,
		blockDetail,
		fetchBlocks,
		fetchTransactions,
		fetchTraces,
		fetchTxDetail,
		fetchBlockDetail
	} from '$lib/stores/chain.svelte';
	import { transfers, fetchTransfers } from '$lib/stores/transfers.svelte';
	import { wallet, fetchWallet } from '$lib/stores/wallet.svelte';
	import { crosschain, fetchCrosschain } from '$lib/stores/crosschain.svelte';
	import { fx, fetchFx } from '$lib/stores/fx.svelte';
	import {
		agents,
		agent,
		agentJobs,
		fetchAgents,
		fetchAgent,
		fetchAgentJobs
	} from '$lib/stores/agents.svelte';
	import { graph, fetchEdges } from '$lib/stores/graph.svelte';
	import { health, fetchHealth } from '$lib/stores/health.svelte';
	import { tokens, fetchTokens } from '$lib/stores/tokens.svelte';
	import { search, runSearch } from '$lib/stores/search.svelte';
	import {
		analyticsFees,
		analyticsVolume,
		analyticsBridgeFlow,
		analyticsAgentLeaderboard,
		analyticsOverview,
		analyticsHistory,
		fetchAnalyticsFees,
		fetchAnalyticsVolume,
		fetchAnalyticsBridgeFlow,
		fetchAgentLeaderboard,
		fetchAnalyticsOverview,
		fetchAnalyticsHistory
	} from '$lib/stores/analytics.svelte';

	// existing filters
	let blockStatsLimit = $state('');
	let blockStatsOffset = $state('');
	let blocksLimit = $state('');
	let blocksOffset = $state('');
	let txFilter = $state({ block: '', from: '', to: '', limit: '', offset: '' });
	let transferFilter = $state({ token: '', from: '', to: '', block: '', limit: '', offset: '' });
	let traceFilter = $state({ tx: '', from: '', to: '', limit: '', offset: '' });
	let crosschainFilter = $state({
		protocol: '',
		event_type: '',
		sender: '',
		recipient: '',
		direction: '',
		limit: '',
		offset: ''
	});
	let fxFilter = $state({ status: '', maker: '', taker: '', quote_id: '', limit: '', offset: '' });
	let agentsLimit = $state('');
	let agentsOffset = $state('');
	let jobsFilter = $state({ status: '', employer: '', worker: '', limit: '', offset: '' });
	let edgesFilter = $state({ wallet: '', limit: '', offset: '' });
	let walletAddress = $state('');
	let walletLimit = $state('');
	let walletOffset = $state('');
	let agentAddress = $state('');

	// new endpoint filters
	let searchQuery = $state('');
	let txDetailHash = $state('');
	let blockDetailNumber = $state('');
	let feesWindow = $state('24h');
	let volumeWindow = $state('24h');
	let volumeToken = $state('');
	let bridgeWindow = $state('24h');
	let overviewWindow = $state('24h');
	let historyWindow = $state('24h');
	let historyLimit = $state('');
	let leaderboardLimit = $state('');
	let tokensSearch = $state('');
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Debug</div>
			<div class="view-sub">API Explorer · all endpoints</div>
		</div>
	</div>

	<!-- HEALTH -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Health</div>
			<div class="card-sub">GET /api/v1/health</div>
			<div class="card-actions">
				<button class="btn ghost" onclick={() => fetchHealth()}>refetch</button>
			</div>
		</div>
		<div class="card-body">
			{#if health.loading}<p class="mono muted">loading…</p>{/if}
			{#if health.error}<p class="mono err-text">{health.error}</p>{/if}
			{#if health.data}<pre>{JSON.stringify(health.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- SEARCH -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Search</div>
			<div class="card-sub">GET /api/v1/search?q=X</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<input
					bind:value={searchQuery}
					placeholder="tx hash (0x+64), address (0x+40), or block number"
					style="width:380px"
				/>
				<button
					class="btn acc"
					onclick={() => {
						if (searchQuery.trim()) runSearch(searchQuery.trim());
					}}>search</button
				>
			</div>
			{#if search.loading}<p class="mono muted">loading…</p>{/if}
			{#if search.error}<p class="mono err-text">{search.error}</p>{/if}
			{#if search.data}<pre>{JSON.stringify(search.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- TX DETAIL -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">TX Detail</div>
			<div class="card-sub">GET /api/v1/tx/{'{hash}'}</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<input bind:value={txDetailHash} placeholder="0x… tx hash" style="width:380px" />
				<button
					class="btn acc"
					onclick={() => {
						if (txDetailHash.trim()) fetchTxDetail(txDetailHash.trim());
					}}>lookup</button
				>
			</div>
			{#if txDetail.loading}<p class="mono muted">loading…</p>{/if}
			{#if txDetail.error}<p class="mono err-text">{txDetail.error}</p>{/if}
			{#if txDetail.data}<pre>{JSON.stringify(txDetail.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- BLOCK DETAIL -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Block Detail</div>
			<div class="card-sub">GET /api/v1/block/{'{number}'}</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<input bind:value={blockDetailNumber} placeholder="block number" style="width:160px" />
				<button
					class="btn acc"
					onclick={() => {
						const n = Number(blockDetailNumber);
						if (n > 0) fetchBlockDetail(n);
					}}>lookup</button
				>
			</div>
			{#if blockDetail.loading}<p class="mono muted">loading…</p>{/if}
			{#if blockDetail.error}<p class="mono err-text">{blockDetail.error}</p>{/if}
			{#if blockDetail.data}<pre>{JSON.stringify(blockDetail.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- ANALYTICS: OVERVIEW -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Analytics · Overview</div>
			<div class="card-sub">GET /api/v1/analytics/overview</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>window
					<select bind:value={overviewWindow}>
						<option>1h</option><option>24h</option><option>7d</option>
					</select>
				</label>
				<button
					class="btn acc"
					onclick={() => fetchAnalyticsOverview({ window: overviewWindow as '1h' | '24h' | '7d' })}
					>fetch</button
				>
			</div>
			{#if analyticsOverview.loading}<p class="mono muted">loading…</p>{/if}
			{#if analyticsOverview.error}<p class="mono err-text">{analyticsOverview.error}</p>{/if}
			{#if analyticsOverview.data}<pre>{JSON.stringify(analyticsOverview.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- ANALYTICS: HISTORY -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Analytics · History</div>
			<div class="card-sub">GET /api/v1/analytics/history</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>window
					<select bind:value={historyWindow}>
						<option>1h</option><option>24h</option><option>7d</option>
					</select>
				</label>
				<label class="dim"
					>limit <input bind:value={historyLimit} placeholder="50" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchAnalyticsHistory({
							window: historyWindow as '1h' | '24h' | '7d',
							limit: historyLimit ? Number(historyLimit) : undefined
						})}>fetch</button
				>
			</div>
			{#if analyticsHistory.loading}<p class="mono muted">loading…</p>{/if}
			{#if analyticsHistory.error}<p class="mono err-text">{analyticsHistory.error}</p>{/if}
			{#if analyticsHistory.data}<pre>{JSON.stringify(analyticsHistory.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- ANALYTICS: FEES -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Analytics · Fees</div>
			<div class="card-sub">GET /api/v1/analytics/fees</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>window
					<select bind:value={feesWindow}>
						<option>1h</option><option>24h</option><option>7d</option>
					</select>
				</label>
				<button
					class="btn acc"
					onclick={() => fetchAnalyticsFees({ window: feesWindow as '1h' | '24h' | '7d' })}
					>fetch</button
				>
			</div>
			{#if analyticsFees.loading}<p class="mono muted">loading…</p>{/if}
			{#if analyticsFees.error}<p class="mono err-text">{analyticsFees.error}</p>{/if}
			{#if analyticsFees.data}<pre>{JSON.stringify(analyticsFees.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- ANALYTICS: VOLUME -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Analytics · Volume</div>
			<div class="card-sub">GET /api/v1/analytics/volume</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>window
					<select bind:value={volumeWindow}>
						<option>1h</option><option>24h</option><option>7d</option>
					</select>
				</label>
				<label class="dim"
					>token
					<select bind:value={volumeToken}>
						<option value="">all</option>
						<option>USDC</option><option>EURC</option><option>USYC</option>
					</select>
				</label>
				<button
					class="btn acc"
					onclick={() =>
						fetchAnalyticsVolume({
							window: volumeWindow as '1h' | '24h' | '7d',
							token: volumeToken || undefined
						})}>fetch</button
				>
			</div>
			{#if analyticsVolume.loading}<p class="mono muted">loading…</p>{/if}
			{#if analyticsVolume.error}<p class="mono err-text">{analyticsVolume.error}</p>{/if}
			{#if analyticsVolume.data}<pre>{JSON.stringify(analyticsVolume.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- ANALYTICS: BRIDGE FLOW -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Analytics · Bridge Flow</div>
			<div class="card-sub">GET /api/v1/analytics/bridge_flow</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>window
					<select bind:value={bridgeWindow}>
						<option>1h</option><option>24h</option><option>7d</option>
					</select>
				</label>
				<button
					class="btn acc"
					onclick={() =>
						fetchAnalyticsBridgeFlow({
							window: bridgeWindow as '1h' | '24h' | '7d'
						})}>fetch</button
				>
			</div>
			{#if analyticsBridgeFlow.loading}<p class="mono muted">loading…</p>{/if}
			{#if analyticsBridgeFlow.error}<p class="mono err-text">{analyticsBridgeFlow.error}</p>{/if}
			{#if analyticsBridgeFlow.data}<pre>{JSON.stringify(
						analyticsBridgeFlow.data,
						null,
						2
					)}</pre>{/if}
		</div>
	</div>

	<!-- ANALYTICS: AGENT LEADERBOARD -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Analytics · Agent Leaderboard</div>
			<div class="card-sub">GET /api/v1/analytics/agent_leaderboard</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>limit <input bind:value={leaderboardLimit} placeholder="50" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() => fetchAgentLeaderboard(leaderboardLimit ? Number(leaderboardLimit) : 50)}
					>fetch</button
				>
			</div>
			{#if analyticsAgentLeaderboard.loading}<p class="mono muted">loading…</p>{/if}
			{#if analyticsAgentLeaderboard.error}
				<p class="mono err-text">{analyticsAgentLeaderboard.error}</p>
			{/if}
			{#if analyticsAgentLeaderboard.data}
				<pre>{JSON.stringify(analyticsAgentLeaderboard.data, null, 2)}</pre>
			{/if}
		</div>
	</div>

	<!-- STATS -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Stats</div>
			<div class="card-sub">GET /api/v1/stats</div>
			<div class="card-actions">
				<button class="btn ghost" onclick={() => fetchStats()}>refetch</button>
			</div>
		</div>
		<div class="card-body">
			{#if stats.loading}<p class="mono muted">loading…</p>{/if}
			{#if stats.error}<p class="mono err-text">{stats.error}</p>{/if}
			{#if stats.data}<pre>{JSON.stringify(stats.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- BLOCK STATS HISTORY -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Block Stats History</div>
			<div class="card-sub">GET /api/v1/block_stats</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>limit <input bind:value={blockStatsLimit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input bind:value={blockStatsOffset} placeholder="0" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchBlockStats(
							blockStatsLimit ? Number(blockStatsLimit) : 50,
							blockStatsOffset ? Number(blockStatsOffset) : 0
						)}>fetch</button
				>
			</div>
			{#if blockStats.loading}<p class="mono muted">loading…</p>{/if}
			{#if blockStats.error}<p class="mono err-text">{blockStats.error}</p>{/if}
			{#if blockStats.data}<pre>{JSON.stringify(blockStats.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- BLOCKS -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Blocks</div>
			<div class="card-sub">GET /api/v1/blocks</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>limit <input bind:value={blocksLimit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input bind:value={blocksOffset} placeholder="0" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchBlocks(
							blocksLimit ? Number(blocksLimit) : 50,
							blocksOffset ? Number(blocksOffset) : 0
						)}>fetch</button
				>
			</div>
			{#if blocks.loading}<p class="mono muted">loading…</p>{/if}
			{#if blocks.error}<p class="mono err-text">{blocks.error}</p>{/if}
			{#if blocks.data}<pre>{JSON.stringify(blocks.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- TRANSACTIONS -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Transactions</div>
			<div class="card-sub">GET /api/v1/transactions</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>block <input bind:value={txFilter.block} placeholder="block number" /></label
				>
				<label class="dim">from <input bind:value={txFilter.from} placeholder="0x…" /></label>
				<label class="dim">to <input bind:value={txFilter.to} placeholder="0x…" /></label>
				<label class="dim"
					>limit <input bind:value={txFilter.limit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input bind:value={txFilter.offset} placeholder="0" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchTransactions({
							block: txFilter.block || undefined,
							from: txFilter.from || undefined,
							to: txFilter.to || undefined,
							limit: txFilter.limit ? Number(txFilter.limit) : undefined,
							offset: txFilter.offset ? Number(txFilter.offset) : undefined
						})}>fetch</button
				>
			</div>
			{#if transactions.loading}<p class="mono muted">loading…</p>{/if}
			{#if transactions.error}<p class="mono err-text">{transactions.error}</p>{/if}
			{#if transactions.data}<pre>{JSON.stringify(transactions.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- TRANSFERS -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Transfers</div>
			<div class="card-sub">GET /api/v1/transfers</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>block <input bind:value={transferFilter.block} placeholder="block number" /></label
				>
				<label class="dim"
					>token
					<select bind:value={transferFilter.token}>
						<option value="">all</option>
						<option>USDC</option><option>EURC</option><option>USYC</option><option>OTHER</option>
					</select>
				</label>
				<label class="dim">from <input bind:value={transferFilter.from} placeholder="0x…" /></label>
				<label class="dim">to <input bind:value={transferFilter.to} placeholder="0x…" /></label>
				<label class="dim"
					>limit <input
						bind:value={transferFilter.limit}
						placeholder="50"
						style="width:60px"
					/></label
				>
				<label class="dim"
					>offset <input
						bind:value={transferFilter.offset}
						placeholder="0"
						style="width:60px"
					/></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchTransfers({
							block: transferFilter.block || undefined,
							token:
								(transferFilter.token as 'USDC' | 'EURC' | 'USYC' | 'OTHER' | undefined) ||
								undefined,
							from: transferFilter.from || undefined,
							to: transferFilter.to || undefined,
							limit: transferFilter.limit ? Number(transferFilter.limit) : undefined,
							offset: transferFilter.offset ? Number(transferFilter.offset) : undefined
						})}>fetch</button
				>
			</div>
			{#if transfers.loading}<p class="mono muted">loading…</p>{/if}
			{#if transfers.error}<p class="mono err-text">{transfers.error}</p>{/if}
			{#if transfers.data}<pre>{JSON.stringify(transfers.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- TRACES -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Traces</div>
			<div class="card-sub">GET /api/v1/traces</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim">tx hash <input bind:value={traceFilter.tx} placeholder="0x…" /></label>
				<label class="dim">from <input bind:value={traceFilter.from} placeholder="0x…" /></label>
				<label class="dim">to <input bind:value={traceFilter.to} placeholder="0x…" /></label>
				<label class="dim"
					>limit <input bind:value={traceFilter.limit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input
						bind:value={traceFilter.offset}
						placeholder="0"
						style="width:60px"
					/></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchTraces({
							tx: traceFilter.tx || undefined,
							from: traceFilter.from || undefined,
							to: traceFilter.to || undefined,
							limit: traceFilter.limit ? Number(traceFilter.limit) : undefined,
							offset: traceFilter.offset ? Number(traceFilter.offset) : undefined
						})}>fetch</button
				>
			</div>
			{#if traces.loading}<p class="mono muted">loading…</p>{/if}
			{#if traces.error}<p class="mono err-text">{traces.error}</p>{/if}
			{#if traces.data}<pre>{JSON.stringify(traces.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- CROSSCHAIN -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Crosschain</div>
			<div class="card-sub">GET /api/v1/crosschain</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>protocol
					<select bind:value={crosschainFilter.protocol}>
						<option value="">all</option><option>cctp</option><option>gateway</option>
					</select>
				</label>
				<label class="dim"
					>event_type
					<select bind:value={crosschainFilter.event_type}>
						<option value="">all</option><option>burn</option><option>mint</option><option
							>deposit</option
						><option>withdraw</option>
					</select>
				</label>
				<label class="dim"
					>direction
					<select bind:value={crosschainFilter.direction}>
						<option value="">all</option><option>inbound</option><option>outbound</option>
					</select>
				</label>
				<label class="dim"
					>sender <input bind:value={crosschainFilter.sender} placeholder="0x…" /></label
				>
				<label class="dim"
					>recipient <input bind:value={crosschainFilter.recipient} placeholder="0x…" /></label
				>
				<label class="dim"
					>limit <input
						bind:value={crosschainFilter.limit}
						placeholder="50"
						style="width:60px"
					/></label
				>
				<label class="dim"
					>offset <input
						bind:value={crosschainFilter.offset}
						placeholder="0"
						style="width:60px"
					/></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchCrosschain({
							protocol: (crosschainFilter.protocol as 'cctp' | 'gateway' | undefined) || undefined,
							event_type:
								(crosschainFilter.event_type as
									| 'burn'
									| 'mint'
									| 'deposit'
									| 'withdraw'
									| undefined) || undefined,
							direction:
								(crosschainFilter.direction as 'inbound' | 'outbound' | undefined) || undefined,
							sender: crosschainFilter.sender || undefined,
							recipient: crosschainFilter.recipient || undefined,
							limit: crosschainFilter.limit ? Number(crosschainFilter.limit) : undefined,
							offset: crosschainFilter.offset ? Number(crosschainFilter.offset) : undefined
						})}>fetch</button
				>
			</div>
			{#if crosschain.loading}<p class="mono muted">loading…</p>{/if}
			{#if crosschain.error}<p class="mono err-text">{crosschain.error}</p>{/if}
			{#if crosschain.data}<pre>{JSON.stringify(crosschain.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- FX -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">FX Trades</div>
			<div class="card-sub">GET /api/v1/fx</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>status
					<select bind:value={fxFilter.status}>
						<option value="">all</option><option>created</option><option>taker_funded</option
						><option>maker_funded</option><option>settled</option><option>cancelled</option>
					</select>
				</label>
				<label class="dim">maker <input bind:value={fxFilter.maker} placeholder="0x…" /></label>
				<label class="dim">taker <input bind:value={fxFilter.taker} placeholder="0x…" /></label>
				<label class="dim"
					>quote_id <input bind:value={fxFilter.quote_id} placeholder="bytes32" /></label
				>
				<label class="dim"
					>limit <input bind:value={fxFilter.limit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input bind:value={fxFilter.offset} placeholder="0" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchFx({
							status: fxFilter.status || undefined,
							maker: fxFilter.maker || undefined,
							taker: fxFilter.taker || undefined,
							quote_id: fxFilter.quote_id || undefined,
							limit: fxFilter.limit ? Number(fxFilter.limit) : undefined,
							offset: fxFilter.offset ? Number(fxFilter.offset) : undefined
						})}>fetch</button
				>
			</div>
			{#if fx.loading}<p class="mono muted">loading…</p>{/if}
			{#if fx.error}<p class="mono err-text">{fx.error}</p>{/if}
			{#if fx.data}<pre>{JSON.stringify(fx.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- AGENTS -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Agents</div>
			<div class="card-sub">GET /api/v1/agents</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>limit <input bind:value={agentsLimit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input bind:value={agentsOffset} placeholder="0" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchAgents(
							agentsLimit ? Number(agentsLimit) : 50,
							agentsOffset ? Number(agentsOffset) : 0
						)}>fetch</button
				>
			</div>
			{#if agents.loading}<p class="mono muted">loading…</p>{/if}
			{#if agents.error}<p class="mono err-text">{agents.error}</p>{/if}
			{#if agents.data}<pre>{JSON.stringify(agents.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- TOKENS -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Tokens</div>
			<div class="card-sub">GET /api/v1/tokens</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>search <input
						bind:value={tokensSearch}
						placeholder="symbol, name, or address"
						style="width:260px"
					/></label
				>
				<button class="btn acc" onclick={() => fetchTokens(50, 0, tokensSearch.trim() || undefined)}
					>fetch</button
				>
			</div>
			{#if tokens.loading}<p class="mono muted">loading…</p>{/if}
			{#if tokens.error}<p class="mono err-text">{tokens.error}</p>{/if}
			{#if tokens.data}<pre>{JSON.stringify(tokens.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- AGENT JOBS -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Agent Jobs</div>
			<div class="card-sub">GET /api/v1/jobs</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim"
					>status
					<select bind:value={jobsFilter.status}>
						<option value="">all</option><option>created</option><option>accepted</option><option
							>delivered</option
						><option>settled</option><option>disputed</option>
					</select>
				</label>
				<label class="dim"
					>employer <input bind:value={jobsFilter.employer} placeholder="0x…" /></label
				>
				<label class="dim">worker <input bind:value={jobsFilter.worker} placeholder="0x…" /></label>
				<label class="dim"
					>limit <input bind:value={jobsFilter.limit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input bind:value={jobsFilter.offset} placeholder="0" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchAgentJobs({
							status: jobsFilter.status || undefined,
							employer: jobsFilter.employer || undefined,
							worker: jobsFilter.worker || undefined,
							limit: jobsFilter.limit ? Number(jobsFilter.limit) : undefined,
							offset: jobsFilter.offset ? Number(jobsFilter.offset) : undefined
						})}>fetch</button
				>
			</div>
			{#if agentJobs.loading}<p class="mono muted">loading…</p>{/if}
			{#if agentJobs.error}<p class="mono err-text">{agentJobs.error}</p>{/if}
			{#if agentJobs.data}<pre>{JSON.stringify(agentJobs.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- GRAPH EDGES -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Graph Edges</div>
			<div class="card-sub">GET /api/v1/edges</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<label class="dim">wallet <input bind:value={edgesFilter.wallet} placeholder="0x…" /></label
				>
				<label class="dim"
					>limit <input bind:value={edgesFilter.limit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input
						bind:value={edgesFilter.offset}
						placeholder="0"
						style="width:60px"
					/></label
				>
				<button
					class="btn acc"
					onclick={() =>
						fetchEdges({
							wallet: edgesFilter.wallet || undefined,
							limit: edgesFilter.limit ? Number(edgesFilter.limit) : undefined,
							offset: edgesFilter.offset ? Number(edgesFilter.offset) : undefined
						})}>fetch</button
				>
			</div>
			{#if graph.loading}<p class="mono muted">loading…</p>{/if}
			{#if graph.error}<p class="mono err-text">{graph.error}</p>{/if}
			{#if graph.data}<pre>{JSON.stringify(graph.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- WALLET -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Wallet Profile</div>
			<div class="card-sub">GET /api/v1/wallet/{'{address}'}</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<input bind:value={walletAddress} placeholder="0x… wallet address" style="width:300px" />
				<label class="dim"
					>limit <input bind:value={walletLimit} placeholder="50" style="width:60px" /></label
				>
				<label class="dim"
					>offset <input bind:value={walletOffset} placeholder="0" style="width:60px" /></label
				>
				<button
					class="btn acc"
					onclick={() => {
						if (walletAddress.trim())
							fetchWallet(
								walletAddress.trim(),
								walletLimit ? Number(walletLimit) : 50,
								walletOffset ? Number(walletOffset) : 0
							);
					}}>lookup</button
				>
			</div>
			{#if wallet.loading}<p class="mono muted">loading…</p>{/if}
			{#if wallet.error}<p class="mono err-text">{wallet.error}</p>{/if}
			{#if wallet.data}<pre>{JSON.stringify(wallet.data, null, 2)}</pre>{/if}
		</div>
	</div>

	<!-- SINGLE AGENT -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Single Agent</div>
			<div class="card-sub">GET /api/v1/agents/{'{address}'}</div>
		</div>
		<div class="card-body">
			<div class="filter-bar">
				<input bind:value={agentAddress} placeholder="0x… agent address" style="width:300px" />
				<button
					class="btn acc"
					onclick={() => {
						if (agentAddress.trim()) fetchAgent(agentAddress.trim());
					}}>lookup</button
				>
			</div>
			{#if agent.loading}<p class="mono muted">loading…</p>{/if}
			{#if agent.error}<p class="mono err-text">{agent.error}</p>{/if}
			{#if agent.data}<pre>{JSON.stringify(agent.data, null, 2)}</pre>{/if}
		</div>
	</div>
</div>

<style>
	.card {
		margin-bottom: 12px;
	}

	pre {
		background: var(--bg-0);
		border: 1px solid var(--border-1);
		border-radius: 4px;
		padding: 10px 12px;
		font-family: var(--mono);
		font-size: var(--t-11);
		color: var(--fg-1);
		overflow: auto;
		max-height: 280px;
		white-space: pre-wrap;
		word-break: break-all;
		margin-top: 8px;
	}

	input,
	select {
		background: var(--bg-2);
		border: 1px solid var(--border-2);
		color: var(--fg-1);
		padding: 3px 8px;
		font-family: var(--mono);
		font-size: var(--t-11);
		border-radius: 4px;
		outline: none;
	}
	input::placeholder {
		color: var(--fg-4);
	}
	input:focus,
	select:focus {
		border-color: var(--border-3);
	}
	select option {
		background: var(--bg-2);
	}

	label {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-family: var(--mono);
		font-size: var(--t-10);
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}

	.err-text {
		color: var(--err);
		margin-top: 6px;
	}
</style>
