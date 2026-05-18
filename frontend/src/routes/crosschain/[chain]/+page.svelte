<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { createSort } from '$lib/sort.svelte';
	import { crosschain, fetchCrosschain } from '$lib/stores/crosschain.svelte';
	import { analyticsBridgeFlow } from '$lib/stores/analytics.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import DataState from '$lib/components/DataState.svelte';
	import Pagination from '$lib/components/Pagination.svelte';

	const sort = createSort('age', 'desc');

	const DIRECTIONS = ['all', 'inbound', 'outbound'];
	const PROTOCOLS = ['all', 'cctp', 'gateway'];

	let direction = $state('all');
	let protocol = $state('all');
	let offset = $state(0);
	const limit = 50;

	const chainId = $derived(Number(page.params.chain ?? 0));
	const chainName = $derived(fmt.domainName(chainId));

	const latestBlock = $derived(stats.data?.block_number ?? 0);
	const bf = $derived(analyticsBridgeFlow.data);
	const chainFlow = $derived(bf?.by_chain?.[chainName] ?? null);

	const EVENT_BADGE: Record<string, string> = {
		burn: 'err',
		mint: 'ok',
		deposit: 'info',
		withdraw: 'warn'
	};

	function load() {
		fetchCrosschain({
			chain: chainId,
			direction: direction === 'all' ? undefined : (direction as 'inbound' | 'outbound'),
			protocol: protocol === 'all' ? undefined : (protocol as 'cctp' | 'gateway'),
			limit,
			offset
		});
	}

	const sortedEvents = $derived(
		sort.apply(crosschain.data?.events ?? [], {
			direction: (e) => (e.source_domain === chainId ? '→ Arc' : '← Arc'),
			event: (e) => e.event_type ?? '',
			protocol: (e) => e.protocol ?? '',
			amount: (e) => parseFloat(e.amount_usdc ?? '0') || 0,
			counterparty: (e) =>
				fmt.domainName(e.source_domain === chainId ? e.destination_domain : e.source_domain) ?? '',
			party: (e) => (e.source_domain === chainId ? (e.recipient ?? '') : (e.sender ?? '')),
			age: (e) => e.block_number ?? 0
		})
	);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">{chainName}</div>
			<div class="view-sub">Domain ID #{chainId}</div>
		</div>
	</div>

	<!-- Summary stats from by_chain data -->
	<div class="grid grid-stats" style="grid-template-columns:repeat(4,1fr);margin-bottom:12px">
		<div class="stat">
			<div class="label">Inbound vol</div>
			<div class="value">{fmt.usdc(chainFlow?.inbound_vol)}</div>
			<div class="delta up">↘ arriving on Arc</div>
		</div>
		<div class="stat">
			<div class="label">Inbound count</div>
			<div class="value">{fmt.num(chainFlow?.inbound_count)}</div>
		</div>
		<div class="stat">
			<div class="label">Outbound vol</div>
			<div class="value">{fmt.usdc(chainFlow?.outbound_vol)}</div>
			<div class="delta down">↗ leaving Arc</div>
		</div>
		<div class="stat">
			<div class="label">Outbound count</div>
			<div class="value">{fmt.num(chainFlow?.outbound_count)}</div>
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
			<div class="card-sub">{chainName} · CCTP · Gateway</div>
		</div>
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th></th>
						<th
							class="sortable {sort.indicator('direction') || ''}"
							onclick={() => sort.toggle('direction')}>dir</th
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
							class="sortable {sort.indicator('counterparty') || ''}"
							onclick={() => sort.toggle('counterparty')}>counterparty</th
						>
						<th
							class="sortable {sort.indicator('party') || ''}"
							onclick={() => sort.toggle('party')}>sender / recipient</th
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
							{@const isOutbound = e.source_domain === chainId}
							<tr>
								<td class="acc">{isOutbound ? '→' : '←'}</td>
								<td class="muted">{isOutbound ? 'out' : 'in'}</td>
								<td
									><span class="badge {EVENT_BADGE[e.event_type] ?? 'muted'}">{e.event_type}</span
									></td
								>
								<td class="muted">{e.protocol}</td>
								<td class="num">{fmt.usdc(e.amount_usdc)}</td>
								<td>
									<a
										href={resolve(
											`/crosschain/${(isOutbound ? e.destination_domain : e.source_domain) ?? 0}/`
										)}
										style="text-decoration:none;color:inherit"
										><span class="chain"
											>{fmt.domainName(isOutbound ? e.destination_domain : e.source_domain)}</span
										></a
									>
								</td>
								<td class="addr"
									><AddrLink address={isOutbound ? (e.recipient ?? '') : (e.sender ?? '')} /></td
								>
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

	<Pagination
		{offset}
		{limit}
		total={crosschain.data?.total ?? 0}
		onPrev={() => {
			offset = Math.max(0, offset - limit);
			load();
		}}
		onNext={() => {
			offset += limit;
			load();
		}}
	/>
</div>
