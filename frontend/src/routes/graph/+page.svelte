<script lang="ts">
	import { graph, fetchEdges } from '$lib/stores/graph.svelte';
	import ForceGraph from '$lib/components/ForceGraph.svelte';
	import * as fmt from '$lib/fmt.js';
	import { createSort } from '$lib/sort.svelte';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	let walletInput = $state('');
	let offset = $state(0);
	const limit = 500;

	function load() {
		fetchEdges({
			wallet: walletInput.trim() || undefined,
			limit,
			offset
		});
	}

	const edges = $derived(graph.data?.edges ?? []);

	const sort = createSort('total_vol', 'desc');
	const sortedEdges = $derived(
		sort.apply(edges, {
			from: (e) => e.from_wallet ?? '',
			to: (e) => e.to_wallet ?? '',
			tx_count: (e) => e.tx_count ?? 0,
			total_vol: (e) => parseFloat(e.total_usdc_human ?? '0') || 0,
			from_agent: (e) => (e.from_is_agent ? '1' : '0'),
			to_agent: (e) => (e.to_is_agent ? '1' : '0')
		})
	);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Wallet graph</div>
			<div class="view-sub">Address relationship network · directional edges</div>
		</div>
		<div class="view-actions">
			<button
				class="btn ghost"
				onclick={() => {
					walletInput = '';
					offset = 0;
					load();
				}}>Reset</button
			>
		</div>
	</div>

	<div class="filter-bar">
		<input
			bind:value={walletInput}
			placeholder="filter by wallet address (0x…)"
			style="width:380px;background:var(--bg-2);border:1px solid var(--border-2);color:var(--fg-1);padding:4px 10px;font-family:var(--mono);font-size:11px;border-radius:4px;outline:none"
			onkeydown={(e) => e.key === 'Enter' && ((offset = 0), load())}
		/>
		<button
			class="btn acc"
			onclick={() => {
				offset = 0;
				load();
			}}>search</button
		>
	</div>

	<!-- Force graph -->
	<div class="graph-stage" style="margin-bottom:12px">
		{#if edges.length}
			<ForceGraph {edges} />
		{:else}
			<div style="position:absolute;inset:0;display:grid;place-items:center">
				<DataState loading={graph.loading} error={graph.error} label="edges" compact />
			</div>
		{/if}
	</div>

	<!-- Edge table -->
	<div class="card">
		<div class="card-head">
			<div class="card-title">Edges</div>
			<div class="card-sub">{graph.data?.count ?? 0} total</div>
		</div>
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th class="sortable {sort.indicator('from') || ''}" onclick={() => sort.toggle('from')}
							>from</th
						>
						<th></th>
						<th class="sortable {sort.indicator('to') || ''}" onclick={() => sort.toggle('to')}
							>to</th
						>
						<th
							class="num sortable {sort.indicator('tx_count') || ''}"
							onclick={() => sort.toggle('tx_count')}>tx count</th
						>
						<th
							class="num sortable {sort.indicator('total_vol') || ''}"
							onclick={() => sort.toggle('total_vol')}>total vol</th
						>
						<th
							class="sortable {sort.indicator('from_agent') || ''}"
							onclick={() => sort.toggle('from_agent')}>from agent</th
						>
						<th
							class="sortable {sort.indicator('to_agent') || ''}"
							onclick={() => sort.toggle('to_agent')}>to agent</th
						>
					</tr>
				</thead>
				<tbody>
					{#if edges.length}
						{#each sortedEdges as e (e.from_wallet + e.to_wallet)}
							<tr>
								<td class="addr"><AddrLink address={e.from_wallet} /></td>
								<td class="acc">→</td>
								<td class="addr"><AddrLink address={e.to_wallet} /></td>
								<td class="num">{fmt.num(e.tx_count)}</td>
								<td class="num">{fmt.usdc(e.total_usdc_human)}</td>
								<td>
									{#if e.from_is_agent}<span class="badge acc">agent</span>{/if}
								</td>
								<td>
									{#if e.to_is_agent}<span class="badge acc">agent</span>{/if}
								</td>
							</tr>
						{/each}
					{:else}
						<DataState loading={graph.loading} error={graph.error} colspan={7} label="edges" />
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
