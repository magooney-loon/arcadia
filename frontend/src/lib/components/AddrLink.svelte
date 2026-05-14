<script lang="ts">
	import { resolve } from '$app/paths';
	import * as fmt from '$lib/fmt.js';

	interface Props {
		address?: string | null;
		short?: boolean;
		long?: boolean; // render the full address instead of truncated
		muted?: boolean;
	}

	let { address, short = true, long = false, muted = false }: Props = $props();
	const display = $derived(long ? (address ?? '—') : short ? fmt.addr(address) : (address ?? '—'));
</script>

{#if address}
	<span class="addr-link" class:muted>
		<a class="addr" href={resolve(`/wallet/${address}/`)} style="text-decoration:none">{display}</a>
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
	.addr-link {
		display: inline-flex;
		align-items: baseline;
		gap: 3px;
	}
	.addr-link.muted .addr {
		color: var(--fg-3);
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
