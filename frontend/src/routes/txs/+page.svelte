<script lang="ts">
	import { onMount } from 'svelte';
	import { transactions, fetchTransactions } from '$lib/stores/chain.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';

	const METHODS = ['all', 'transfer', 'approve', 'swap', 'execute', 'multicall', 'deploy'];

	let methodFilter = $state('all');
	let offset = $state(0);
	const limit = 100;

	onMount(() => fetchTransactions({ limit, offset }));

	function loadPage() {
		fetchTransactions({ limit, offset });
	}

	const latestBlock = $derived(stats.data?.block_number ?? 0);

	const filtered = $derived(() => {
		const txs = transactions.data?.transactions ?? [];
		if (methodFilter === 'all') return txs;
		if (methodFilter === 'deploy') return txs.filter((t) => t.is_contract_deploy === true);
		return txs.filter((t) => fmt.methodName(t.sighash) === methodFilter);
	});
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Transactions</div>
			<div class="view-sub">All transaction types · arc testnet</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost" onclick={loadPage}>Refresh</button>
		</div>
	</div>

	<div class="filter-bar">
		{#each METHODS as m}
			<button class="chip {methodFilter === m ? 'on' : ''}" onclick={() => (methodFilter = m)}>{m}</button>
		{/each}
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th>hash</th>
						<th>method</th>
						<th>from</th>
						<th></th>
						<th>to</th>
						<th class="num">fee</th>
						<th class="num">status</th>
						<th class="num">age</th>
					</tr>
				</thead>
				<tbody>
					{#if transactions.loading}
						<tr><td colspan="8" style="text-align:center;color:var(--fg-4);padding:32px" class="mono">loading…</td></tr>
					{:else if transactions.error}
						<tr><td colspan="8" style="text-align:center;color:var(--err);padding:16px" class="mono">{transactions.error}</td></tr>
					{:else if filtered().length}
						{#each filtered() as t}
							<tr>
								<td><span class="hash mono">{fmt.hash(t.hash)}</span></td>
								<td><span class="badge info">{fmt.methodName(t.sighash)}</span></td>
								<td class="addr">{fmt.addr(t.from_addr)}</td>
								<td class="muted">→</td>
								<td class="addr">{t.is_contract_deploy ? '(new)' : fmt.addr(t.to_addr)}</td>
								<td class="num muted">{fmt.usdc(t.fee_usdc, 5)}</td>
								<td class="num">
									{#if t.status === 1}
										<span class="badge ok">✓</span>
									{:else}
										<span class="badge err">✗</span>
									{/if}
								</td>
								<td class="num muted">{fmt.blockAge(t.block_number, latestBlock)}</td>
							</tr>
						{/each}
					{:else}
						<tr><td colspan="8" style="text-align:center;color:var(--fg-4);padding:32px" class="mono">no results</td></tr>
					{/if}
				</tbody>
			</table>
		</div>
	</div>

	<div class="filter-bar" style="margin-top:10px;justify-content:flex-end">
		<button class="btn ghost" disabled={offset === 0} onclick={() => { offset = Math.max(0, offset - limit); loadPage(); }}>← prev</button>
		<span class="mono dim" style="font-size:11px">offset {offset}</span>
		<button class="btn ghost" onclick={() => { offset += limit; loadPage(); }}>next →</button>
	</div>
</div>
