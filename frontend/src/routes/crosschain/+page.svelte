<script lang="ts">
	import { createSort } from '$lib/sort.svelte';
	import { crosschain, fetchCrosschain } from '$lib/stores/crosschain.svelte';
	import { analyticsBridgeFlow } from '$lib/stores/analytics.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	const sort = createSort('age', 'desc');

	const DIRECTIONS = ['all', 'inbound', 'outbound'];
	const PROTOCOLS = ['all', 'cctp', 'gateway'];

	let direction = $state('all');
	let protocol = $state('all');
	let offset = $state(0);
	const limit = 50;

	function load() {
		fetchCrosschain({
			direction: direction === 'all' ? undefined : (direction as 'inbound' | 'outbound'),
			protocol: protocol === 'all' ? undefined : (protocol as 'cctp' | 'gateway'),
			limit,
			offset
		});
	}

	const latestBlock = $derived(stats.data?.block_number ?? 0);
	const bf = $derived(analyticsBridgeFlow.data);

	const sortedEvents = $derived(
		sort.apply(crosschain.data?.events ?? [], {
			from_chain: (e) => fmt.domainName(e.source_domain) ?? '',
			to_chain: (e) => fmt.domainName(e.destination_domain) ?? '',
			event: (e) => e.event_type ?? '',
			protocol: (e) => e.protocol ?? '',
			amount: (e) => parseFloat(e.amount_usdc ?? '0') || 0,
			sender: (e) => e.sender ?? '',
			age: (e) => e.block_number ?? 0
		})
	);

	const EVENT_BADGE: Record<string, string> = {
		burn: 'err',
		mint: 'ok',
		deposit: 'info',
		withdraw: 'warn'
	};
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Cross-chain</div>
			<div class="view-sub">CCTP mints & burns · inbound and outbound · 24h</div>
		</div>
	</div>

	<!-- Summary stats -->
	<div class="grid grid-stats" style="grid-template-columns:repeat(4,1fr);margin-bottom:12px">
		<div class="stat">
			<div class="label">Inbound count</div>
			<div class="value">{fmt.num(bf?.inbound_count)}</div>
			<div class="delta up">↘ arriving on Arc</div>
		</div>
		<div class="stat">
			<div class="label">Inbound vol</div>
			<div class="value">{fmt.usdc(bf?.inbound_vol)}</div>
		</div>
		<div class="stat">
			<div class="label">Outbound count</div>
			<div class="value">{fmt.num(bf?.outbound_count)}</div>
			<div class="delta down">↗ leaving Arc</div>
		</div>
		<div class="stat">
			<div class="label">Outbound vol</div>
			<div class="value">{fmt.usdc(bf?.outbound_vol)}</div>
		</div>
	</div>

	<div class="filter-bar">
		{#each DIRECTIONS as d (d)}
			<button
				class="chip {direction === d ? 'on' : ''}"
				onclick={() => {
					direction = d;
					offset = 0;
					load();
				}}>{d}</button
			>
		{/each}
		<span class="mono dim" style="font-size:10px;margin-left:8px">protocol</span>
		{#each PROTOCOLS as p (p)}
			<button
				class="chip {protocol === p ? 'on' : ''}"
				onclick={() => {
					protocol = p;
					offset = 0;
					load();
				}}>{p}</button
			>
		{/each}
	</div>

	<div class="card">
		<div class="card-head">
			<div class="card-title">Messages</div>
			<div class="card-sub">CCTP · Gateway</div>
		</div>
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th
							class="sortable {sort.indicator('from_chain') || ''}"
							onclick={() => sort.toggle('from_chain')}>from chain</th
						>
						<th></th>
						<th
							class="sortable {sort.indicator('to_chain') || ''}"
							onclick={() => sort.toggle('to_chain')}>to chain</th
						>
						<th
							class="sortable {sort.indicator('event') || ''}"
							onclick={() => sort.toggle('event')}>event</th
						>
						<th
							class="sortable {sort.indicator('protocol') || ''}"
							onclick={() => sort.toggle('protocol')}>protocol</th
						>
						<th
							class="num sortable {sort.indicator('amount') || ''}"
							onclick={() => sort.toggle('amount')}>amount</th
						>
						<th
							class="sortable {sort.indicator('sender') || ''}"
							onclick={() => sort.toggle('sender')}>sender</th
						>
						<th
							class="num sortable {sort.indicator('age') || ''}"
							onclick={() => sort.toggle('age')}>age</th
						>
					</tr>
				</thead>
				<tbody>
					{#if crosschain.data?.events.length}
						{#each sortedEvents as e (e.id)}
							<tr>
								<td><span class="chain">{fmt.domainName(e.source_domain)}</span></td>
								<td class="acc">→</td>
								<td><span class="chain">{fmt.domainName(e.destination_domain)}</span></td>
								<td
									><span class="badge {EVENT_BADGE[e.event_type] ?? 'muted'}">{e.event_type}</span
									></td
								>
								<td class="muted">{e.protocol}</td>
								<td class="num">{fmt.usdc(e.amount_usdc)}</td>
								<td class="addr"><AddrLink address={e.sender ?? ''} /></td>
								<td class="num muted">{fmt.blockAge(e.block_number, latestBlock)}</td>
							</tr>
						{/each}
					{:else}
						<DataState
							loading={crosschain.loading}
							error={crosschain.error}
							colspan={8}
							label="cross-chain events"
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
