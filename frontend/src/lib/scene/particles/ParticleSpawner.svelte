<script lang="ts">
	import { transfers } from '$lib/stores/transfers.svelte';
	import { spawnParticles } from '$lib/scene-state/particles.svelte';
	import { getNodePositions } from '$lib/scene-state/layout.svelte';

	let spawned = false;

	// Spawn particles once the graph layout and transfers are both ready
	$effect(() => {
		if (spawned) return;
		const positions = getNodePositions();
		const txData = transfers.data?.transfers;
		if (Object.keys(positions).length === 0 || !txData || txData.length === 0) return;

		spawned = true;
		spawnParticles(txData);
	});
</script>
