<script lang="ts">
	import { onMount } from 'svelte';
	import { stats, fetchStats } from '$lib/stores/stats.svelte';
	import { blocks, transactions, traces, fetchBlocks, fetchTransactions, fetchTraces } from '$lib/stores/chain.svelte';
	import { transfers, fetchTransfers } from '$lib/stores/transfers.svelte';
	import { wallet, fetchWallet } from '$lib/stores/wallet.svelte';
	import { crosschain, fetchCrosschain } from '$lib/stores/crosschain.svelte';
	import { fx, fetchFx } from '$lib/stores/fx.svelte';
	import { agents, agentJobs, fetchAgents, fetchAgentJobs } from '$lib/stores/agents.svelte';
	import { graph, fetchEdges } from '$lib/stores/graph.svelte';

	let walletAddress = $state('');

	onMount(() => {
		fetchStats();
		fetchBlocks();
		fetchTransactions();
		fetchTraces();
		fetchTransfers();
		fetchCrosschain();
		fetchFx();
		fetchAgents();
		fetchAgentJobs();
		fetchEdges();
	});

	function lookupWallet() {
		if (walletAddress.trim()) fetchWallet(walletAddress.trim());
	}
</script>

<h1>Arcadia API Explorer</h1>

<section>
	<h2>Stats <code>GET /api/v1/stats</code></h2>
	{#if stats.loading}<p>loading…</p>{/if}
	{#if stats.error}<p>error: {stats.error}</p>{/if}
	{#if stats.data}<pre>{JSON.stringify(stats.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Blocks <code>GET /api/v1/blocks</code></h2>
	{#if blocks.loading}<p>loading…</p>{/if}
	{#if blocks.error}<p>error: {blocks.error}</p>{/if}
	{#if blocks.data}<pre>{JSON.stringify(blocks.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Transactions <code>GET /api/v1/transactions</code></h2>
	{#if transactions.loading}<p>loading…</p>{/if}
	{#if transactions.error}<p>error: {transactions.error}</p>{/if}
	{#if transactions.data}<pre>{JSON.stringify(transactions.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Traces <code>GET /api/v1/traces</code></h2>
	{#if traces.loading}<p>loading…</p>{/if}
	{#if traces.error}<p>error: {traces.error}</p>{/if}
	{#if traces.data}<pre>{JSON.stringify(traces.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Transfers <code>GET /api/v1/transfers</code></h2>
	{#if transfers.loading}<p>loading…</p>{/if}
	{#if transfers.error}<p>error: {transfers.error}</p>{/if}
	{#if transfers.data}<pre>{JSON.stringify(transfers.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Crosschain <code>GET /api/v1/crosschain</code></h2>
	{#if crosschain.loading}<p>loading…</p>{/if}
	{#if crosschain.error}<p>error: {crosschain.error}</p>{/if}
	{#if crosschain.data}<pre>{JSON.stringify(crosschain.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>FX Trades <code>GET /api/v1/fx</code></h2>
	{#if fx.loading}<p>loading…</p>{/if}
	{#if fx.error}<p>error: {fx.error}</p>{/if}
	{#if fx.data}<pre>{JSON.stringify(fx.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Agents <code>GET /api/v1/agents</code></h2>
	{#if agents.loading}<p>loading…</p>{/if}
	{#if agents.error}<p>error: {agents.error}</p>{/if}
	{#if agents.data}<pre>{JSON.stringify(agents.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Agent Jobs <code>GET /api/v1/jobs</code></h2>
	{#if agentJobs.loading}<p>loading…</p>{/if}
	{#if agentJobs.error}<p>error: {agentJobs.error}</p>{/if}
	{#if agentJobs.data}<pre>{JSON.stringify(agentJobs.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Graph Edges <code>GET /api/v1/edges</code></h2>
	{#if graph.loading}<p>loading…</p>{/if}
	{#if graph.error}<p>error: {graph.error}</p>{/if}
	{#if graph.data}<pre>{JSON.stringify(graph.data, null, 2)}</pre>{/if}
</section>

<section>
	<h2>Wallet <code>GET /api/v1/wallet/{'{address}'}</code></h2>
	<input bind:value={walletAddress} placeholder="0x..." />
	<button onclick={lookupWallet}>lookup</button>
	{#if wallet.loading}<p>loading…</p>{/if}
	{#if wallet.error}<p>error: {wallet.error}</p>{/if}
	{#if wallet.data}<pre>{JSON.stringify(wallet.data, null, 2)}</pre>{/if}
</section>
