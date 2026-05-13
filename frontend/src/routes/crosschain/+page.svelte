<script lang="ts">
	import { onMount } from 'svelte';
	import { crosschain, fetchCrosschain } from '$lib/stores/crosschain.svelte';
	import { analyticsBridgeFlow, fetchAnalyticsBridgeFlow } from '$lib/stores/analytics.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';

	const DIRECTIONS = ['all', 'inbound', 'outbound'];
	const PROTOCOLS = ['all', 'cctp', 'gateway'];

	let direction = $state('all');
	let protocol = $state('all');
	let offset = $state(0);
	const limit = 50;

	onMount(() => {
		load();
		fetchAnalyticsBridgeFlow();
	});

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
						<th>from chain</th>
						<th></th>
						<th>to chain</th>
						<th>event</th>
						<th>protocol</th>
						<th class="num">amount</th>
						<th>sender</th>
						<th class="num">age</th>
					</tr>
				</thead>
				<tbody>
					{#if crosschain.loading}
						<tr
							><td colspan="8" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>loading…</td
							></tr
						>
					{:else if crosschain.error}
						<tr
							><td colspan="8" style="text-align:center;color:var(--err);padding:16px" class="mono"
								>{crosschain.error}</td
							></tr
						>
					{:else if crosschain.data?.events.length}
						{#each crosschain.data.events as e (e.id)}
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
								<td class="addr"
									><a
										href={fmt.explorerAddr(e.sender ?? '')}
										target="_blank"
										rel="noopener noreferrer"
										style="text-decoration:none">{fmt.addr(e.sender)}</a
									></td
								>
								<td class="num muted">{fmt.blockAge(e.block_number, latestBlock)}</td>
							</tr>
						{/each}
					{:else}
						<tr
							><td colspan="8" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
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
