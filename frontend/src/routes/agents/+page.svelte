<script lang="ts">
	import { onMount } from 'svelte';
	import { analyticsAgentLeaderboard, fetchAgentLeaderboard } from '$lib/stores/analytics.svelte';
	import * as fmt from '$lib/fmt.js';

	let limit = $state(50);

	onMount(() => fetchAgentLeaderboard(limit));

	const board = $derived(analyticsAgentLeaderboard.data?.leaderboard ?? []);

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
						<th>address</th>
						<th class="num">txs</th>
						<th class="num">volume</th>
						<th class="num">fees spent</th>
						<th class="num">jobs</th>
						<th class="num">paid</th>
						<th class="num">rejected</th>
						<th class="num">escrow</th>
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
						{#each board as a, i (a.agent_address)}
							<tr>
								<td class="muted">{i + 1}</td>
								<td>
									<a
										class="addr"
										href={fmt.explorerAddr(a.agent_address)}
										target="_blank"
										rel="external noopener noreferrer"
										style="text-decoration:none">{fmt.addr(a.agent_address)}</a
									>
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
