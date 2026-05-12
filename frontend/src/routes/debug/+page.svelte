<script lang="ts">
	import { onMount } from 'svelte';
	import { stats, fetchStats } from '$lib/stores/stats.svelte';
	import { blockStats, fetchBlockStats } from '$lib/stores/blockStats.svelte';
	import { blocks, transactions, traces, fetchBlocks, fetchTransactions, fetchTraces } from '$lib/stores/chain.svelte';
	import { transfers, fetchTransfers } from '$lib/stores/transfers.svelte';
	import { wallet, fetchWallet } from '$lib/stores/wallet.svelte';
	import { crosschain, fetchCrosschain } from '$lib/stores/crosschain.svelte';
	import { fx, fetchFx } from '$lib/stores/fx.svelte';
	import { agents, agentJobs, fetchAgents, fetchAgentJobs } from '$lib/stores/agents.svelte';
	import { graph, fetchEdges } from '$lib/stores/graph.svelte';

	// per-section filter state
	let txFilter = $state({ block: '', from: '', to: '' });
	let transferFilter = $state({ token: '', from: '', to: '' });
	let traceFilter = $state({ tx: '', from: '', to: '' });
	let crosschainFilter = $state({ protocol: '', event_type: '', sender: '', recipient: '', direction: '' });
	let fxFilter = $state({ status: '', maker: '', taker: '', quote_id: '' });
	let jobsFilter = $state({ status: '', employer: '', worker: '' });
	let edgesFilter = $state({ wallet: '' });
	let walletAddress = $state('');
	let agentAddress = $state('');

	onMount(() => {
		fetchStats();
		fetchBlockStats(50);
		fetchBlocks(50);
		fetchTransactions();
		fetchTraces();
		fetchTransfers();
		fetchCrosschain();
		fetchFx();
		fetchAgents();
		fetchAgentJobs();
		fetchEdges();
	});
</script>

<style>
	:global(body) { font-family: monospace; font-size: 13px; background: #0d0d0d; color: #ccc; padding: 1rem; }
	h1 { color: #e6e6ee; margin-bottom: 1.5rem; }
	section { margin-bottom: 2rem; border-top: 1px solid #222; padding-top: 1rem; }
	h2 { color: #7ee5a8; margin-bottom: 0.5rem; }
	code { color: #6be3ff; }
	pre { background: #111; padding: 0.75rem; overflow: auto; max-height: 300px; white-space: pre-wrap; word-break: break-all; }
	.filters { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 0.5rem; align-items: center; }
	input, select { background: #1a1a2e; border: 1px solid #333; color: #ccc; padding: 3px 7px; font-family: monospace; font-size: 12px; border-radius: 3px; }
	button { background: #1e3a2e; border: 1px solid #7ee5a8; color: #7ee5a8; padding: 3px 10px; font-family: monospace; font-size: 12px; cursor: pointer; border-radius: 3px; }
	.err { color: #ff6b6b; }
	.loading { color: #666; }
	label { color: #888; font-size: 11px; }
</style>

<h1>Arcadia Debug · API Explorer</h1>

<!-- STATS -->
<section>
	<h2>Stats <code>GET /api/v1/stats</code></h2>
	<button onclick={() => fetchStats()}>refetch</button>
	{#if stats.loading}<p class="loading">loading…</p>{/if}
	{#if stats.error}<p class="err">{stats.error}</p>{/if}
	{#if stats.data}<pre>{JSON.stringify(stats.data, null, 2)}</pre>{/if}
</section>

<!-- BLOCK STATS HISTORY -->
<section>
	<h2>Block Stats History <code>GET /api/v1/block_stats</code></h2>
	<button onclick={() => fetchBlockStats(50)}>refetch</button>
	{#if blockStats.loading}<p class="loading">loading…</p>{/if}
	{#if blockStats.error}<p class="err">{blockStats.error}</p>{/if}
	{#if blockStats.data}<pre>{JSON.stringify(blockStats.data, null, 2)}</pre>{/if}
</section>

<!-- BLOCKS -->
<section>
	<h2>Blocks <code>GET /api/v1/blocks</code></h2>
	<button onclick={() => fetchBlocks(50)}>refetch</button>
	{#if blocks.loading}<p class="loading">loading…</p>{/if}
	{#if blocks.error}<p class="err">{blocks.error}</p>{/if}
	{#if blocks.data}<pre>{JSON.stringify(blocks.data, null, 2)}</pre>{/if}
</section>

<!-- TRANSACTIONS -->
<section>
	<h2>Transactions <code>GET /api/v1/transactions</code></h2>
	<div class="filters">
		<label>block <input bind:value={txFilter.block} placeholder="block number" /></label>
		<label>from <input bind:value={txFilter.from} placeholder="0x…" /></label>
		<label>to <input bind:value={txFilter.to} placeholder="0x…" /></label>
		<button onclick={() => fetchTransactions({ block: txFilter.block || undefined, from: txFilter.from || undefined, to: txFilter.to || undefined })}>fetch</button>
	</div>
	{#if transactions.loading}<p class="loading">loading…</p>{/if}
	{#if transactions.error}<p class="err">{transactions.error}</p>{/if}
	{#if transactions.data}<pre>{JSON.stringify(transactions.data, null, 2)}</pre>{/if}
</section>

<!-- TRANSFERS -->
<section>
	<h2>Transfers <code>GET /api/v1/transfers</code></h2>
	<div class="filters">
		<label>token
			<select bind:value={transferFilter.token}>
				<option value="">all</option>
				<option>USDC</option><option>EURC</option><option>USYC</option><option>OTHER</option>
			</select>
		</label>
		<label>from <input bind:value={transferFilter.from} placeholder="0x…" /></label>
		<label>to <input bind:value={transferFilter.to} placeholder="0x…" /></label>
		<button onclick={() => fetchTransfers({ token: (transferFilter.token as 'USDC'|'EURC'|'USYC'|'OTHER'|undefined) || undefined, from: transferFilter.from || undefined, to: transferFilter.to || undefined })}>fetch</button>
	</div>
	{#if transfers.loading}<p class="loading">loading…</p>{/if}
	{#if transfers.error}<p class="err">{transfers.error}</p>{/if}
	{#if transfers.data}<pre>{JSON.stringify(transfers.data, null, 2)}</pre>{/if}
</section>

<!-- TRACES -->
<section>
	<h2>Traces <code>GET /api/v1/traces</code></h2>
	<div class="filters">
		<label>tx hash <input bind:value={traceFilter.tx} placeholder="0x…" /></label>
		<label>from <input bind:value={traceFilter.from} placeholder="0x…" /></label>
		<label>to <input bind:value={traceFilter.to} placeholder="0x…" /></label>
		<button onclick={() => fetchTraces({ tx: traceFilter.tx || undefined, from: traceFilter.from || undefined, to: traceFilter.to || undefined })}>fetch</button>
	</div>
	{#if traces.loading}<p class="loading">loading…</p>{/if}
	{#if traces.error}<p class="err">{traces.error}</p>{/if}
	{#if traces.data}<pre>{JSON.stringify(traces.data, null, 2)}</pre>{/if}
</section>

<!-- CROSSCHAIN -->
<section>
	<h2>Crosschain <code>GET /api/v1/crosschain</code></h2>
	<div class="filters">
		<label>protocol
			<select bind:value={crosschainFilter.protocol}>
				<option value="">all</option><option>cctp</option><option>gateway</option>
			</select>
		</label>
		<label>event_type
			<select bind:value={crosschainFilter.event_type}>
				<option value="">all</option><option>burn</option><option>mint</option><option>deposit</option><option>withdraw</option>
			</select>
		</label>
		<label>direction
			<select bind:value={crosschainFilter.direction}>
				<option value="">all</option><option>inbound</option><option>outbound</option>
			</select>
		</label>
		<label>sender <input bind:value={crosschainFilter.sender} placeholder="0x…" /></label>
		<label>recipient <input bind:value={crosschainFilter.recipient} placeholder="0x…" /></label>
		<button onclick={() => fetchCrosschain({
			protocol: (crosschainFilter.protocol as 'cctp'|'gateway'|undefined) || undefined,
			event_type: (crosschainFilter.event_type as 'burn'|'mint'|'deposit'|'withdraw'|undefined) || undefined,
			direction: (crosschainFilter.direction as 'inbound'|'outbound'|undefined) || undefined,
			sender: crosschainFilter.sender || undefined,
			recipient: crosschainFilter.recipient || undefined
		})}>fetch</button>
	</div>
	{#if crosschain.loading}<p class="loading">loading…</p>{/if}
	{#if crosschain.error}<p class="err">{crosschain.error}</p>{/if}
	{#if crosschain.data}<pre>{JSON.stringify(crosschain.data, null, 2)}</pre>{/if}
</section>

<!-- FX -->
<section>
	<h2>FX Trades <code>GET /api/v1/fx</code></h2>
	<div class="filters">
		<label>status
			<select bind:value={fxFilter.status}>
				<option value="">all</option><option>created</option><option>taker_funded</option><option>maker_funded</option><option>settled</option><option>cancelled</option>
			</select>
		</label>
		<label>maker <input bind:value={fxFilter.maker} placeholder="0x…" /></label>
		<label>taker <input bind:value={fxFilter.taker} placeholder="0x…" /></label>
		<label>quote_id <input bind:value={fxFilter.quote_id} placeholder="bytes32" /></label>
		<button onclick={() => fetchFx({ status: fxFilter.status || undefined, maker: fxFilter.maker || undefined, taker: fxFilter.taker || undefined, quote_id: fxFilter.quote_id || undefined })}>fetch</button>
	</div>
	{#if fx.loading}<p class="loading">loading…</p>{/if}
	{#if fx.error}<p class="err">{fx.error}</p>{/if}
	{#if fx.data}<pre>{JSON.stringify(fx.data, null, 2)}</pre>{/if}
</section>

<!-- AGENTS -->
<section>
	<h2>Agents <code>GET /api/v1/agents</code></h2>
	<button onclick={() => fetchAgents()}>refetch</button>
	{#if agents.loading}<p class="loading">loading…</p>{/if}
	{#if agents.error}<p class="err">{agents.error}</p>{/if}
	{#if agents.data}<pre>{JSON.stringify(agents.data, null, 2)}</pre>{/if}
</section>

<!-- AGENT JOBS -->
<section>
	<h2>Agent Jobs <code>GET /api/v1/jobs</code></h2>
	<div class="filters">
		<label>status
			<select bind:value={jobsFilter.status}>
				<option value="">all</option><option>created</option><option>accepted</option><option>delivered</option><option>settled</option><option>disputed</option>
			</select>
		</label>
		<label>employer <input bind:value={jobsFilter.employer} placeholder="0x…" /></label>
		<label>worker <input bind:value={jobsFilter.worker} placeholder="0x…" /></label>
		<button onclick={() => fetchAgentJobs({ status: jobsFilter.status || undefined, employer: jobsFilter.employer || undefined, worker: jobsFilter.worker || undefined })}>fetch</button>
	</div>
	{#if agentJobs.loading}<p class="loading">loading…</p>{/if}
	{#if agentJobs.error}<p class="err">{agentJobs.error}</p>{/if}
	{#if agentJobs.data}<pre>{JSON.stringify(agentJobs.data, null, 2)}</pre>{/if}
</section>

<!-- GRAPH EDGES -->
<section>
	<h2>Graph Edges <code>GET /api/v1/edges</code></h2>
	<div class="filters">
		<label>wallet <input bind:value={edgesFilter.wallet} placeholder="0x…" /></label>
		<button onclick={() => fetchEdges({ wallet: edgesFilter.wallet || undefined })}>fetch</button>
	</div>
	{#if graph.loading}<p class="loading">loading…</p>{/if}
	{#if graph.error}<p class="err">{graph.error}</p>{/if}
	{#if graph.data}<pre>{JSON.stringify(graph.data, null, 2)}</pre>{/if}
</section>

<!-- WALLET -->
<section>
	<h2>Wallet Profile <code>GET /api/v1/wallet/{'{address}'}</code></h2>
	<div class="filters">
		<input bind:value={walletAddress} placeholder="0x…" style="width:340px" />
		<button onclick={() => { if (walletAddress.trim()) fetchWallet(walletAddress.trim()); }}>lookup</button>
	</div>
	{#if wallet.loading}<p class="loading">loading…</p>{/if}
	{#if wallet.error}<p class="err">{wallet.error}</p>{/if}
	{#if wallet.data}<pre>{JSON.stringify(wallet.data, null, 2)}</pre>{/if}
</section>

<!-- SINGLE AGENT -->
<section>
	<h2>Single Agent <code>GET /api/v1/agents/{'{address}'}</code></h2>
	<div class="filters">
		<input bind:value={agentAddress} placeholder="0x…" style="width:340px" />
		<button onclick={() => {
			if (!agentAddress.trim()) return;
			import('$lib/stores/agents.svelte').then(m => m.fetchAgent(agentAddress.trim()));
		}}>lookup</button>
	</div>
	<p style="color:#555;font-size:11px">result shows in Agents section above (agent + jobs)</p>
</section>
