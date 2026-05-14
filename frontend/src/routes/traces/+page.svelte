<script lang="ts">
	import { onMount } from 'svelte';
	import { traces, fetchTraces } from '$lib/stores/chain.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import { createSort } from '$lib/sort.svelte';
	import * as fmt from '$lib/fmt.js';

	let txFilter = $state('');
	let offset = $state(0);
	const limit = 50;

	onMount(() => load());

	function load() {
		fetchTraces({
			tx: txFilter.trim() || undefined,
			limit,
			offset
		});
	}

	const latestBlock = $derived(stats.data?.block_number ?? 0);

	const sort = createSort('age', 'desc');
	const sortedTraces = $derived(
		sort.apply(traces.data?.traces ?? [], {
			tx: (t) => t.tx_hash ?? '',
			type: (t) => t.call_type ?? t.trace_type ?? '',
			from: (t) => t.from_addr ?? '',
			to: (t) => t.to_addr ?? '',
			gas_used: (t) => t.gas_used ?? 0,
			age: (t) => t.block_number ?? 0
		})
	);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Traces</div>
			<div class="view-sub">Internal call traces · EVM execution</div>
		</div>
	</div>

	<div
		class="card"
		style="border-left:2px solid var(--warn);padding:10px 14px;margin-bottom:12px;background:var(--bg-2)"
	>
		<div class="mono" style="font-size:11px;color:var(--fg-2)">
			<span style="color:var(--warn);font-weight:600">⚠ no data available</span>
			— HyperSync does not expose traces for Arc Testnet. Traces are only served on select
			networks via dedicated <span class="dim">*-traces.hypersync.xyz</span> endpoints (eth-traces, base-traces).
			Backfilling from JSON-RPC <span class="dim">debug_traceBlockByNumber</span> against the Arc
			RPC pool is possible but not yet wired up.
		</div>
	</div>

	<div class="filter-bar">
		<input
			bind:value={txFilter}
			placeholder="filter by tx hash (0x…)"
			style="width:380px;background:var(--bg-2);border:1px solid var(--border-2);color:var(--fg-1);padding:4px 10px;font-family:var(--mono);font-size:11px;border-radius:4px;outline:none"
			onkeydown={(e) => e.key === 'Enter' && load()}
		/>
		<button class="btn acc" onclick={load}>fetch</button>
		{#if txFilter}
			<button
				class="btn ghost"
				onclick={() => {
					txFilter = '';
					offset = 0;
					load();
				}}>clear</button
			>
		{/if}
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th class="sortable {sort.indicator('tx') || ''}" onclick={() => sort.toggle('tx')}
							>tx</th
						>
						<th class="sortable {sort.indicator('type') || ''}" onclick={() => sort.toggle('type')}
							>type</th
						>
						<th class="sortable {sort.indicator('from') || ''}" onclick={() => sort.toggle('from')}
							>from</th
						>
						<th class="sortable {sort.indicator('to') || ''}" onclick={() => sort.toggle('to')}
							>to</th
						>
						<th
							class="num sortable {sort.indicator('gas_used') || ''}"
							onclick={() => sort.toggle('gas_used')}>gas used</th
						>
						<th
							class="num sortable {sort.indicator('age') || ''}"
							onclick={() => sort.toggle('age')}>age</th
						>
					</tr>
				</thead>
				<tbody>
					{#if traces.loading}
						<tr
							><td colspan="6" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>loading…</td
							></tr
						>
					{:else if traces.error}
						<tr
							><td colspan="6" style="text-align:center;color:var(--err);padding:16px" class="mono"
								>{traces.error}</td
							></tr
						>
					{:else if traces.data?.traces.length}
						{#each sortedTraces as t (t.tx_hash + '_' + t.block_number)}
							<tr>
								<td
									><a
										class="hash mono"
										href={fmt.explorerTx(t.tx_hash)}
										target="_blank"
										rel="external noopener noreferrer"
										style="text-decoration:none">{fmt.hash(t.tx_hash)}</a
									></td
								>
								<td><span class="badge muted">{t.call_type ?? t.trace_type ?? '—'}</span></td>
								<td class="addr"
									><a
										href={fmt.explorerAddr(t.from_addr)}
										target="_blank"
										rel="external noopener noreferrer"
										style="text-decoration:none">{fmt.addr(t.from_addr)}</a
									></td
								>
								<td class="addr"
									><a
										href={fmt.explorerAddr(t.to_addr)}
										target="_blank"
										rel="external noopener noreferrer"
										style="text-decoration:none">{fmt.addr(t.to_addr)}</a
									></td
								>
								<td class="num muted">{fmt.num(t.gas_used)}</td>
								<td class="num muted">{fmt.blockAge(t.block_number, latestBlock)}</td>
							</tr>
						{/each}
					{:else}
						<tr
							><td colspan="6" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>no results</td
							></tr
						>
					{/if}
				</tbody>
			</table>
		</div>
	</div>

	{#if !txFilter}
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
	{/if}
</div>
