<script lang="ts">
	import { onMount } from 'svelte';
	import { agentJobs, fetchAgentJobs } from '$lib/stores/agents.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';

	const TABS = [
		{ label: 'All', status: '' },
		{ label: 'Created', status: 'created' },
		{ label: 'Accepted', status: 'accepted' },
		{ label: 'Delivered', status: 'delivered' },
		{ label: 'Settled', status: 'settled' },
		{ label: 'Disputed', status: 'disputed' }
	];

	let activeStatus = $state('');
	let offset = $state(0);
	const limit = 50;

	onMount(() => load());

	function load() {
		fetchAgentJobs({
			status: activeStatus || undefined,
			limit,
			offset
		});
	}

	function setTab(status: string) {
		activeStatus = status;
		offset = 0;
		load();
	}

	const latestBlock = $derived(stats.data?.block_number ?? 0);
	const jobs = $derived(agentJobs.data?.jobs ?? []);

	const totalEscrow = $derived(jobs.reduce((s, j) => s + parseFloat(j.payment_usdc ?? '0'), 0));
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Job market</div>
			<div class="view-sub">ERC-8183 · agent job lifecycle</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost" onclick={load}>Refresh</button>
		</div>
	</div>

	{#if jobs.length > 0}
		<div class="grid" style="grid-template-columns:repeat(3,1fr);margin-bottom:12px">
			<div class="stat">
				<div class="label">Jobs shown</div>
				<div class="value">{jobs.length}</div>
			</div>
			<div class="stat">
				<div class="label">Total escrow</div>
				<div class="value">{fmt.usdc(totalEscrow)}</div>
			</div>
			<div class="stat">
				<div class="label">Avg reward</div>
				<div class="value">{jobs.length ? fmt.usdc(totalEscrow / jobs.length) : '—'}</div>
			</div>
		</div>
	{/if}

	<div class="tabs">
		{#each TABS as tab (tab.status)}
			<button
				class="tab {activeStatus === tab.status ? 'active' : ''}"
				onclick={() => setTab(tab.status)}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th>job id</th>
						<th>employer</th>
						<th>worker</th>
						<th>status</th>
						<th class="num">reward</th>
						<th class="num">created</th>
						<th class="num">settled</th>
					</tr>
				</thead>
				<tbody>
					{#if agentJobs.loading}
						<tr
							><td colspan="7" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>loading…</td
							></tr
						>
					{:else if agentJobs.error}
						<tr
							><td colspan="7" style="text-align:center;color:var(--err);padding:16px" class="mono"
								>{agentJobs.error}</td
							></tr
						>
					{:else if jobs.length}
						{#each jobs as j (j.job_id)}
							<tr>
								<td><span class="hash mono">{fmt.hash(j.job_id)}</span></td>
								<td class="addr">{fmt.addr(j.employer_address)}</td>
								<td class="addr">{fmt.addr(j.worker_address)}</td>
								<td><span class="badge {fmt.jobBadge(j.status)}">{j.status}</span></td>
								<td class="num">{fmt.usdc(j.payment_usdc)}</td>
								<td class="num muted">{fmt.blockAge(j.created_at_block, latestBlock)}</td>
								<td class="num muted"
									>{j.settled_at_block ? fmt.blockAge(j.settled_at_block, latestBlock) : '—'}</td
								>
							</tr>
						{/each}
					{:else}
						<tr
							><td colspan="7" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>no jobs found</td
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
