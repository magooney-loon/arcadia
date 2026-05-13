<script lang="ts">
	import { onMount } from 'svelte';
	import { transfers, fetchTransfers } from '$lib/stores/transfers.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';

	const TOKENS = ['all', 'USDC', 'EURC', 'USYC', 'OTHER'];
	const TOKEN_COLORS: Record<string, string> = {
		USDC: 'ok', EURC: 'info', USYC: 'warn', OTHER: 'muted',
	};

	let tokenFilter = $state('all');
	let offset = $state(0);
	const limit = 50;

	onMount(() => load());

	function load() {
		fetchTransfers({
			token: tokenFilter === 'all' ? undefined : (tokenFilter as any),
			limit,
			offset,
		});
	}

	const latestBlock = $derived(stats.data?.block_number ?? 0);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Transfers</div>
			<div class="view-sub">Token transfers · USDC, EURC, USYC and more</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost" onclick={load}>Refresh</button>
		</div>
	</div>

	<div class="filter-bar">
		{#each TOKENS as tok}
			<button
				class="chip {tokenFilter === tok ? 'on' : ''}"
				onclick={() => { tokenFilter = tok; offset = 0; load(); }}
			>{tok}</button>
		{/each}
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th>tx</th>
						<th>token</th>
						<th>from</th>
						<th></th>
						<th>to</th>
						<th class="num">amount</th>
						<th class="num">age</th>
					</tr>
				</thead>
				<tbody>
					{#if transfers.loading}
						<tr><td colspan="7" style="text-align:center;color:var(--fg-4);padding:32px" class="mono">loading…</td></tr>
					{:else if transfers.error}
						<tr><td colspan="7" style="text-align:center;color:var(--err);padding:16px" class="mono">{transfers.error}</td></tr>
					{:else if transfers.data?.transfers.length}
						{#each transfers.data.transfers as t}
							<tr>
								<td><span class="hash mono">{fmt.hash(t.tx_hash)}</span></td>
								<td><span class="badge {TOKEN_COLORS[t.token_symbol] ?? 'muted'}">{t.token_symbol}</span></td>
								<td class="addr">{fmt.addr(t.from_addr)}</td>
								<td class="muted">→</td>
								<td class="addr">{fmt.addr(t.to_addr)}</td>
								<td class="num">{fmt.usdc(t.amount_human)}</td>
								<td class="num muted">{fmt.blockAge(t.block_number, latestBlock)}</td>
							</tr>
						{/each}
					{:else}
						<tr><td colspan="7" style="text-align:center;color:var(--fg-4);padding:32px" class="mono">no results</td></tr>
					{/if}
				</tbody>
			</table>
		</div>
	</div>

	<div class="filter-bar" style="margin-top:10px;justify-content:flex-end">
		<button class="btn ghost" disabled={offset === 0} onclick={() => { offset = Math.max(0, offset - limit); load(); }}>← prev</button>
		<span class="mono dim" style="font-size:11px">offset {offset}</span>
		<button class="btn ghost" onclick={() => { offset += limit; load(); }}>next →</button>
	</div>
</div>
