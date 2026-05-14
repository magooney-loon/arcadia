<script lang="ts">
	import { resolve } from '$app/paths';
	import * as fmt from '$lib/fmt.js';

	interface Props {
		hash?: string | null;
		short?: boolean;
		long?: boolean;
	}

	let { hash, short = true, long = false }: Props = $props();
	const display = $derived(long ? (hash ?? '—') : short ? fmt.hash(hash) : (hash ?? '—'));
</script>

{#if hash}
	<span class="tx-link">
		<a class="hash mono" href={resolve(`/tx/${hash}/`)} style="text-decoration:none">{display}</a>
		<a
			class="ext"
			href={fmt.explorerTx(hash)}
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
	.tx-link {
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
