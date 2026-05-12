<script lang="ts">
	import { onMount } from 'svelte';
	import { graph, fetchEdges } from '$lib/stores/graph.svelte';
	import { agents, fetchAgents } from '$lib/stores/agents.svelte';
	import { fetchTransfers } from '$lib/stores/transfers.svelte';
	import { runSimulation, getNodePositions } from '$lib/scene-state/layout.svelte';
	import TransferEdges from './TransferEdges.svelte';
	import WalletNodes from './WalletNodes.svelte';
	import AgentNodes from './AgentNodes.svelte';

	onMount(async () => {
		await Promise.all([
			fetchEdges({ limit: 500 }),
			fetchAgents(200),
			fetchTransfers({ limit: 200 })
		]);

		const edges = graph.data?.edges;
		if (edges && edges.length > 0) {
			const agentAddrs = new Set((agents.data?.agents ?? []).map((a) => a.agent_address));
			runSimulation(edges, agentAddrs);
		}
	});
</script>

{#if Object.keys(getNodePositions()).length > 0}
	<TransferEdges />
	<WalletNodes />
	<AgentNodes />
{/if}
