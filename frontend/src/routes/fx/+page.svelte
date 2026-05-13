<script lang="ts">
	import { onMount } from 'svelte';
	import { fx, fetchFx } from '$lib/stores/fx.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';

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
						<th>pair</th>
						<th class="num">size</th>
						<th class="num">rate</th>
						<th>maker</th>
						<th>taker</th>
						<th>status</th>
						<th class="num">age</th>
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
						{#each fx.data.trades as t, i (i)}
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
										rel="noopener noreferrer"
										style="text-decoration:none">{fmt.addr(t.maker)}</a
									></td
								>
								<td class="addr"
									><a
										href={fmt.explorerAddr(t.taker)}
										target="_blank"
										rel="noopener noreferrer"
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
