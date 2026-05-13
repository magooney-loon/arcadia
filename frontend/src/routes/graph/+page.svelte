<script lang="ts">
	import { onMount } from 'svelte';
	import { graph, fetchEdges } from '$lib/stores/graph.svelte';
	import ForceGraph from '$lib/components/ForceGraph.svelte';
	import * as fmt from '$lib/fmt.js';

	let walletInput = $state('');
	let offset = $state(0);
	const limit = 100;

	onMount(() => load());

	function load() {
		fetchEdges({
			wallet: walletInput.trim() || undefined,
			limit,
			offset
		});
	}

	const edges = $derived(graph.data?.edges ?? []);
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
		{:else if graph.loading}
			<div style="position:absolute;inset:0;display:grid;place-items:center">
				<span class="mono dim" style="font-size:11px">loading…</span>
			</div>
		{:else}
			<div style="position:absolute;inset:0;display:grid;place-items:center">
				<span class="mono dim" style="font-size:11px">no edges to display</span>
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
						<th>from</th>
						<th></th>
						<th>to</th>
						<th class="num">tx count</th>
						<th class="num">total vol</th>
						<th>from agent</th>
						<th>to agent</th>
					</tr>
				</thead>
				<tbody>
					{#if graph.loading}
						<tr
							><td colspan="7" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>loading…</td
							></tr
						>
					{:else if graph.error}
						<tr
							><td colspan="7" style="text-align:center;color:var(--err);padding:16px" class="mono"
								>{graph.error}</td
							></tr
						>
					{:else if edges.length}
						{#each edges as e (e.from_wallet + e.to_wallet)}
							<tr>
								<td class="addr"
									><a
										href={fmt.explorerAddr(e.from_wallet)}
										target="_blank"
										rel="external noopener noreferrer"
										style="text-decoration:none">{fmt.addr(e.from_wallet)}</a
									></td
								>
								<td class="acc">→</td>
								<td class="addr"
									><a
										href={fmt.explorerAddr(e.to_wallet)}
										target="_blank"
										rel="external noopener noreferrer"
										style="text-decoration:none">{fmt.addr(e.to_wallet)}</a
									></td
								>
								<td class="num">{fmt.num(e.tx_count)}</td>
								<td class="num">{fmt.usdc(e.total_usdc)}</td>
								<td>{e.from_is_agent ? '<span class="badge acc">agent</span>' : ''}</td>
								<td>{e.to_is_agent ? '<span class="badge acc">agent</span>' : ''}</td>
							</tr>
						{/each}
					{:else}
						<tr
							><td colspan="7" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>no edges found</td
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
