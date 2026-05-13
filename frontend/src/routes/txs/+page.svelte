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
		{#each METHODS as m (m)}
			<button class="chip {methodFilter === m ? 'on' : ''}" onclick={() => (methodFilter = m)}
				>{m}</button
			>
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
						<tr
							><td colspan="8" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
								>loading…</td
							></tr
						>
					{:else if transactions.error}
						<tr
							><td colspan="8" style="text-align:center;color:var(--err);padding:16px" class="mono"
								>{transactions.error}</td
							></tr
						>
					{:else if filtered().length}
						{#each filtered() as t (t.hash)}
							<tr>
								<td
									><a
										class="hash mono"
										href={fmt.explorerTx(t.hash)}
										target="_blank"
										rel="noopener noreferrer"
										style="text-decoration:none">{fmt.hash(t.hash)}</a
									></td
								>
								<td><span class="badge info">{fmt.methodName(t.sighash)}</span></td>
								<td class="addr"
									><a
										href={fmt.explorerAddr(t.from_addr)}
										target="_blank"
										rel="noopener noreferrer"
										style="text-decoration:none">{fmt.addr(t.from_addr)}</a
									></td
								>
								<td class="muted">→</td>
								<td class="addr"
									>{#if t.is_contract_deploy}(new){:else}<a
											href={fmt.explorerAddr(t.to_addr)}
											target="_blank"
											rel="noopener noreferrer"
											style="text-decoration:none">{fmt.addr(t.to_addr)}</a
										>{/if}</td
								>
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
				loadPage();
			}}>← prev</button
		>
		<span class="mono dim" style="font-size:11px">offset {offset}</span>
		<button
			class="btn ghost"
			onclick={() => {
				offset += limit;
				loadPage();
			}}>next →</button
		>
	</div>
</div>
