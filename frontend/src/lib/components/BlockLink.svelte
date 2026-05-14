<script lang="ts">
	import { resolve } from '$app/paths';
	import * as fmt from '$lib/fmt.js';

	interface Props {
		number?: number | null;
	}

	let { number }: Props = $props();
</script>

{#if number}
	<span class="block-link">
		<a class="mono" href={resolve(`/blocks/${number}/`)} style="text-decoration:none">#{number}</a>
		<a
			class="ext"
			href={fmt.explorerBlock(number)}
			target="_blank"
			rel="external noopener noreferrer"
			title="open in arcscan"
			aria-label="open in arcscan">↗</a
		>
	</span>
{:else}
	<span class="muted">—</span>
{/if}

<style>
	.block-link {
		display: inline-flex;
		align-items: baseline;
		gap: 3px;
	}
	.ext {
		font-size: 9px;
		color: var(--fg-4);
		text-decoration: none;
		opacity: 0.6;
	}
	.ext:hover {
		opacity: 1;
		color: var(--info);
	}
</style>
