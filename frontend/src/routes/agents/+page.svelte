<script lang="ts">
	import { onMount } from 'svelte';
	import { createSort } from '$lib/sort.svelte';
	import { analyticsAgentLeaderboard, fetchAgentLeaderboard } from '$lib/stores/analytics.svelte';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';

	let limit = $state(50);

	onMount(() => fetchAgentLeaderboard(limit));

	const board = $derived(analyticsAgentLeaderboard.data?.leaderboard ?? []);

	const sort = createSort('volume', 'desc');
	const sortedBoard = $derived(
		sort.apply(board, {
			address: (a) => a.agent_address ?? '',
			txs: (a) => a.tx_count ?? 0,
			volume: (a) => parseFloat(a.usdc_transferred_human ?? '0') || 0,
			fees_spent: (a) => parseFloat(a.usdc_spent_fees_human ?? '0') || 0,
			jobs: (a) => a.job_count ?? 0,
			paid: (a) => a.paid_jobs ?? 0,
			rejected: (a) => a.rejected_jobs ?? 0,
			escrow: (a) => a.total_escrow ?? 0
		})
	);

	const totalEscrow = $derived(board.reduce((s, a) => s + (a.total_escrow ?? 0), 0));
	const inFlightJobs = $derived(
		board.reduce((s, a) => s + (a.job_count - a.paid_jobs - a.rejected_jobs), 0)
	);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Agent registry</div>
			<div class="view-sub">ERC-8004 · registered agents on arc testnet</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost" onclick={() => fetchAgentLeaderboard(limit)}>Refresh</button>
		</div>
	</div>

	<div class="grid" style="grid-template-columns:repeat(4,1fr);margin-bottom:12px">
		<div class="stat">
			<div class="label">Total registered</div>
			<div class="value">{analyticsAgentLeaderboard.data?.count ?? '—'}</div>
		</div>
		<div class="stat">
			<div class="label">In leaderboard</div>
			<div class="value">{board.length}</div>
		</div>
		<div class="stat">
			<div class="label">Jobs in-flight</div>
			<div class="value">{inFlightJobs}</div>
		</div>
		<div class="stat">
			<div class="label">Total escrow</div>
			<div class="value">{fmt.usdc(totalEscrow)}</div>
		</div>
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th>#</th>
						<th
							class="sortable {sort.indicator('address') || ''}"
							onclick={() => sort.toggle('address')}>address</th
						>
						<th
							class="num sortable {sort.indicator('txs') || ''}"
							onclick={() => sort.toggle('txs')}>txs</th
						>
						<th
							class="num sortable {sort.indicator('volume') || ''}"
							onclick={() => sort.toggle('volume')}>volume</th
						>
						<th
							class="num sortable {sort.indicator('fees_spent') || ''}"
							onclick={() => sort.toggle('fees_spent')}>fees spent</th
						>
						<th
							class="num sortable {sort.indicator('jobs') || ''}"
							onclick={() => sort.toggle('jobs')}>jobs</th
						>
						<th
							class="num sortable {sort.indicator('paid') || ''}"
							onclick={() => sort.toggle('paid')}>paid</th
						>
						<th
							class="num sortable {sort.indicator('rejected') || ''}"
							onclick={() => sort.toggle('rejected')}>rejected</th
						>
						<th
							class="num sortable {sort.indicator('escrow') || ''}"
							onclick={() => sort.toggle('escrow')}>escrow</th
						>
					</tr>
				</thead>
				<tbody>
					{#if analyticsAgentLeaderboard.loading}
						<tr
							><td colspan="9" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>loading…</td
							></tr
						>
					{:else if analyticsAgentLeaderboard.error}
						<tr
							><td colspan="9" style="text-align:center;color:var(--err);padding:16px" class="mono"
								>{analyticsAgentLeaderboard.error}</td
							></tr
						>
					{:else if board.length}
						{#each sortedBoard as a, i (a.agent_address)}
							<tr>
								<td class="muted">{i + 1}</td>
								<td>
									<AddrLink address={a.agent_address} />
									{#if a.job_count > 0}
										<span class="badge acc" style="margin-left:6px">agent</span>
									{/if}
								</td>
								<td class="num">{fmt.num(a.tx_count)}</td>
								<td class="num">{fmt.usdc(a.usdc_transferred_human)}</td>
								<td class="num muted">{fmt.usdc(a.usdc_spent_fees_human, 4)}</td>
								<td class="num">{a.job_count}</td>
								<td class="num" style="color:var(--ok)">{a.paid_jobs}</td>
								<td class="num" style="color:{a.rejected_jobs > 0 ? 'var(--err)' : 'var(--fg-4)'}"
									>{a.rejected_jobs}</td
								>
								<td class="num">{fmt.usdc(a.total_escrow)}</td>
							</tr>
						{/each}
					{:else}
						<tr
							><td colspan="9" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>no agents found</td
							></tr
						>
					{/if}
				</tbody>
			</table>
		</div>
	</div>
</div>
