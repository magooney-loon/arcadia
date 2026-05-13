<script lang="ts">
	import { onMount } from 'svelte';
	import { fx, fetchFx } from '$lib/stores/fx.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';
	import { createSort } from '$lib/sort.svelte';

	const STATUSES = ['all', 'created', 'taker_funded', 'maker_funded', 'settled', 'cancelled'];

	let statusFilter = $state('all');
	let offset = $state(0);
	const limit = 50;

	onMount(() => load());

	function load() {
		fetchFx({
			status: statusFilter === 'all' ? undefined : statusFilter,
			limit,
			offset
		});
	}

	const latestBlock = $derived(stats.data?.block_number ?? 0);

	const sort = createSort('age', 'desc');

	const sortedTrades = $derived(
		sort.apply(fx.data?.trades ?? [], {
			pair: (t) => String(t.input_token ?? '') + '/' + String(t.output_token ?? ''),
			size: (t) => parseFloat(String(t.input_amount ?? '0')) || 0,
			rate: (t) => (typeof t.implied_rate === 'number' ? t.implied_rate : 0),
			maker: (t) => t.maker ?? '',
			taker: (t) => t.taker ?? '',
			status: (t) => t.status ?? '',
			age: (t) => t.block_number ?? 0
		})
	);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">StableFX</div>
			<div class="view-sub">RFQ market · on-chain FX trades</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost" onclick={load}>Refresh</button>
		</div>
	</div>

	<div class="filter-bar">
		{#each STATUSES as s (s)}
			<button
				class="chip {statusFilter === s ? 'on' : ''}"
				onclick={() => {
					statusFilter = s;
					offset = 0;
					load();
				}}>{s}</button
			>
		{/each}
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th class="sortable {sort.indicator('pair') || ''}" onclick={() => sort.toggle('pair')}
							>pair</th
						>
						<th
							class="sortable num {sort.indicator('size') || ''}"
							onclick={() => sort.toggle('size')}>size</th
						>
						<th
							class="sortable num {sort.indicator('rate') || ''}"
							onclick={() => sort.toggle('rate')}>rate</th
						>
						<th
							class="sortable {sort.indicator('maker') || ''}"
							onclick={() => sort.toggle('maker')}>maker</th
						>
						<th
							class="sortable {sort.indicator('taker') || ''}"
							onclick={() => sort.toggle('taker')}>taker</th
						>
						<th
							class="sortable {sort.indicator('status') || ''}"
							onclick={() => sort.toggle('status')}>status</th
						>
						<th
							class="sortable num {sort.indicator('age') || ''}"
							onclick={() => sort.toggle('age')}>age</th
						>
					</tr>
				</thead>
				<tbody>
					{#if fx.loading}
						<tr
							><td colspan="7" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>loading…</td
							></tr
						>
					{:else if fx.error}
						<tr
							><td colspan="7" style="text-align:center;color:var(--err);padding:16px" class="mono"
								>{fx.error}</td
							></tr
						>
					{:else if fx.data?.trades.length}
						{#each sortedTrades as t, i (i)}
							{@const inputTok = t.input_token as string}
							{@const outputTok = t.output_token as string}
							<tr>
								<td class="mono">{inputTok ?? '?'}/{outputTok ?? '?'}</td>
								<td class="num">{fmt.usdc(t.input_amount as string)}</td>
								<td class="num muted"
									>{typeof t.implied_rate === 'number'
										? (t.implied_rate as number).toFixed(4)
										: '—'}</td
								>
								<td class="addr"
									><a
										href={fmt.explorerAddr(t.maker)}
										target="_blank"
										rel="external noopener noreferrer"
										style="text-decoration:none">{fmt.addr(t.maker)}</a
									></td
								>
								<td class="addr"
									><a
										href={fmt.explorerAddr(t.taker)}
										target="_blank"
										rel="external noopener noreferrer"
										style="text-decoration:none">{fmt.addr(t.taker)}</a
									></td
								>
								<td><span class="badge {fmt.fxBadge(t.status)}">{t.status}</span></td>
								<td class="num muted">{fmt.blockAge(t.block_number, latestBlock)}</td>
							</tr>
						{/each}
					{:else}
						<tr
							><td colspan="7" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>no results</td
							></tr
						>
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
