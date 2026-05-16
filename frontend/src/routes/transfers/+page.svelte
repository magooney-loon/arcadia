<script lang="ts">
	import { onMount } from 'svelte';
	import { createSort } from '$lib/sort.svelte';
	import { transfers, fetchTransfers } from '$lib/stores/transfers.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import TxLink from '$lib/components/TxLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	const TOKENS = ['all', 'USDC', 'EURC', 'USYC', 'OTHER'];
	const TOKEN_COLORS: Record<string, string> = {
		USDC: 'ok',
		EURC: 'info',
		USYC: 'warn',
		OTHER: 'muted'
	};

	let tokenFilter = $state('all');
	let offset = $state(0);
	const limit = 50;

	onMount(() => {
		transfers.data = null;
		load();
	});

	function load() {
		fetchTransfers({
			token:
				tokenFilter === 'all' ? undefined : (tokenFilter as 'USDC' | 'EURC' | 'USYC' | 'OTHER'),
			limit,
			offset
		});
	}

	const sort = createSort('age', 'desc');

	const sortedTransfers = $derived(
		sort.apply(transfers.data?.transfers ?? [], {
			tx: (t) => t.tx_hash ?? '',
			token: (t) => t.token_symbol ?? '',
			from: (t) => t.from_addr ?? '',
			to: (t) => t.to_addr ?? '',
			amount: (t) => parseFloat(t.amount_human ?? '0') || 0,
			age: (t) => t.block_number ?? 0
		})
	);

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
		{#each TOKENS as tok (tok)}
			<button
				class="chip {tokenFilter === tok ? 'on' : ''}"
				onclick={() => {
					tokenFilter = tok;
					offset = 0;
					load();
				}}>{tok}</button
			>
		{/each}
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th class="sortable {sort.indicator('tx') || ''}" onclick={() => sort.toggle('tx')}
							>tx</th
						>
						<th
							class="sortable {sort.indicator('token') || ''}"
							onclick={() => sort.toggle('token')}>token</th
						>
						<th class="sortable {sort.indicator('from') || ''}" onclick={() => sort.toggle('from')}
							>from</th
						>
						<th></th>
						<th class="sortable {sort.indicator('to') || ''}" onclick={() => sort.toggle('to')}
							>to</th
						>
						<th
							class="sortable num {sort.indicator('amount') || ''}"
							onclick={() => sort.toggle('amount')}>amount</th
						>
						<th
							class="sortable num {sort.indicator('age') || ''}"
							onclick={() => sort.toggle('age')}>age</th
						>
					</tr>
				</thead>
				<tbody>
					{#if transfers.data?.transfers.length}
						{#each sortedTransfers as t (t.id)}
							<tr>
								<td><TxLink hash={t.tx_hash} /></td>
								<td
									><span class="badge {TOKEN_COLORS[t.token_symbol] ?? 'muted'}"
										>{t.token_symbol}</span
									></td
								>
								<td class="addr"><AddrLink address={t.from_addr} /></td>
								<td class="muted">→</td>
								<td class="addr"><AddrLink address={t.to_addr} /></td>
								<td class="num">{fmt.usdc(t.amount_human)}</td>
								<td class="num muted">{fmt.blockAge(t.block_number, latestBlock)}</td>
							</tr>
						{/each}
					{:else}
						<DataState
							loading={transfers.loading}
							error={transfers.error}
							colspan={7}
							label="transfers"
						/>
					{/if}
				</tbody>
			</table>
		</div>
	</div>

	<div class="filter-bar" style="margin-top:10px;justify-content:flex-end">
		<button
			class="btn ghost"
			disabled={offset === 0}
			onclick={() => {
				offset = Math.max(0, offset - limit);
				load();
			}}>← prev</button
		>
		<span class="mono dim" style="font-size:11px">offset {offset}</span>
		<button
			class="btn ghost"
			onclick={() => {
				offset += limit;
				load();
			}}>next →</button
		>
	</div>
</div>
