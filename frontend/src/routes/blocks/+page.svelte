<script lang="ts">
	import { onMount } from 'svelte';
	import { blocks, fetchBlocks } from '$lib/stores/chain.svelte';
	import { blockStats, fetchBlockStats } from '$lib/stores/blockStats.svelte';
	import * as fmt from '$lib/fmt.js';

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

	const feesMap = $derived(
		new Map((blockStats.data?.stats ?? []).map((s) => [s.block_number, s]))
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
						<th>block</th>
						<th>age</th>
						<th>txs</th>
						<th>miner</th>
						<th class="num">gas util</th>
						<th class="num">fees</th>
					</tr>
				</thead>
				<tbody>
					{#if blocks.loading}
						<tr><td colspan="6" style="text-align:center;color:var(--fg-4);padding:32px" class="mono">loading…</td></tr>
					{:else if blocks.error}
						<tr><td colspan="6" style="text-align:center;color:var(--err);padding:16px" class="mono">{blocks.error}</td></tr>
					{:else if blocks.data?.blocks.length}
						{#each blocks.data.blocks as b}
							{@const stat = feesMap.get(b.number)}
							<tr>
								<td><span class="acc mono">#{b.number}</span></td>
								<td class="muted">{fmt.tsAge(b.timestamp)}</td>
								<td>{b.tx_count ?? 0}</td>
								<td class="addr">{fmt.addr(b.miner)}</td>
								<td class="num">{fmt.pct(b.utilization_pct)}</td>
								<td class="num">{stat ? fmt.usdc(stat.total_fee_usdc, 4) : '—'}</td>
							</tr>
						{/each}
					{:else}
						<tr><td colspan="6" style="text-align:center;color:var(--fg-4);padding:32px" class="mono">no data</td></tr>
					{/if}
				</tbody>
			</table>
		</div>
	</div>

	<div class="filter-bar" style="margin-top:10px;justify-content:flex-end">
		<button class="btn ghost" disabled={offset === 0} onclick={() => { offset = Math.max(0, offset - limit); load(); }}>← prev</button>
		<span class="mono dim" style="font-size:11px">offset {offset}</span>
		<button class="btn ghost" onclick={() => { offset += limit; load(); }}>next →</button>
	</div>
</div>
