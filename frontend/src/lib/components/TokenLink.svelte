<script lang="ts">
	import { resolve } from '$app/paths';
	import * as fmt from '$lib/fmt.js';

	interface Props {
		address?: string | null;
		symbol?: string | null;
		name?: string | null;
		short?: boolean;
		long?: boolean;
	}

	let { address, symbol, name, short = true, long = false }: Props = $props();

	const label = $derived(
		symbol || name || (long ? (address ?? '—') : short ? fmt.addr(address) : (address ?? '—'))
	);
</script>

{#if address}
	<span class="token-link">
		<a class="label" href={resolve(`/tokens/${address}/`)} style="text-decoration:none">{label}</a>
		<a
			class="ext"
			href={fmt.explorerAddr(address)}
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
	.token-link {
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
