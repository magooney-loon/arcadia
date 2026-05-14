<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { blocks, fetchBlocks } from '$lib/stores/chain.svelte';
	import { blockStats, fetchBlockStats } from '$lib/stores/blockStats.svelte';
	import { createSort } from '$lib/sort.svelte';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';

	let limit = $state(50);
	let offset = $state(0);

	onMount(() => {
		fetchBlocks(limit, offset);
		fetchBlockStats(limit, offset);
	});

	function load() {
		fetchBlocks(limit, offset);
		fetchBlockStats(limit, offset);
	}

	const feesMap = $derived(new Map((blockStats.data?.stats ?? []).map((s) => [s.block_number, s])));

	const sort = createSort('number', 'desc');

	const sortedBlocks = $derived(
		sort.apply(blocks.data?.blocks ?? [], {
			number: (b) => b.number,
			age: (b) => b.timestamp ?? 0,
			txs: (b) => b.tx_count ?? 0,
			miner: (b) => b.miner ?? '',
			gas_util: (b) => b.utilization_pct ?? 0,
			fees: (b) => feesMap.get(b.number)?.total_fee_usdc ?? 0
		})
	);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Blocks</div>
			<div class="view-sub">Live block feed · arc testnet</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost" onclick={() => load()}>Refresh</button>
		</div>
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th
							class="sortable {sort.indicator('number') || ''}"
							onclick={() => sort.toggle('number')}>block</th
						>
						<th class="sortable {sort.indicator('age') || ''}" onclick={() => sort.toggle('age')}
							>age</th
						>
						<th class="sortable {sort.indicator('txs') || ''}" onclick={() => sort.toggle('txs')}
							>txs</th
						>
						<th
							class="sortable {sort.indicator('miner') || ''}"
							onclick={() => sort.toggle('miner')}>miner</th
						>
						<th
							class="num sortable {sort.indicator('gas_util') || ''}"
							onclick={() => sort.toggle('gas_util')}>gas util</th
						>
						<th
							class="num sortable {sort.indicator('fees') || ''}"
							onclick={() => sort.toggle('fees')}>fees</th
						>
					</tr>
				</thead>
				<tbody>
					{#if blocks.loading}
						<tr
							><td colspan="6" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>loading…</td
							></tr
						>
					{:else if blocks.error}
						<tr
							><td colspan="6" style="text-align:center;color:var(--err);padding:16px" class="mono"
								>{blocks.error}</td
							></tr
						>
					{:else if blocks.data?.blocks.length}
						{#each sortedBlocks as b (b.number)}
							{@const stat = feesMap.get(b.number)}
							<tr>
								<td
									><a
										class="acc mono"
										href={resolve(`/blocks/${b.number}/`)}
										style="text-decoration:none">#{b.number}</a
									></td
								>
								<td class="muted">{fmt.tsAge(b.timestamp)}</td>
								<td>{b.tx_count ?? 0}</td>
								<td class="addr"
									><AddrLink address={b.miner} /></td
								>
								<td class="num">{fmt.pct(b.utilization_pct)}</td>
								<td class="num">{stat ? fmt.usdc(stat.total_fee_usdc, 4) : '—'}</td>
							</tr>
						{/each}
					{:else}
						<tr
							><td colspan="6" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>no data</td
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
