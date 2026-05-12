<script lang="ts">
	import { onMount } from 'svelte';
	import { blocks, fetchBlocks } from '$lib/stores/chain.svelte';
	import BlockNodes from './BlockNodes.svelte';

	onMount(() => {
		fetchBlocks(50);
	});
</script>

{#if blocks.data?.blocks && blocks.data.blocks.length > 0}
	<BlockNodes blocks={blocks.data.blocks} />
{/if}
