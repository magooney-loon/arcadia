<script lang="ts">
	import { transactions, fetchTransactions } from '$lib/stores/chain.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';
	import { createSort } from '$lib/sort.svelte';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import TxLink from '$lib/components/TxLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	const METHODS = ['all', 'transfer', 'approve', 'swap', 'execute', 'multicall', 'deploy'];

	let methodFilter = $state('all');
	let offset = $state(0);
	const limit = 100;

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

	const sort = createSort('age', 'desc');

	const sortedTxs = $derived(
		sort.apply(filtered(), {
			hash: (t) => t.hash ?? '',
			method: (t) => fmt.methodName(t.sighash) ?? '',
			from: (t) => t.from_addr ?? '',
			to: (t) => t.to_addr ?? '',
			fee: (t) => parseFloat(t.fee_usdc ?? '0') || 0,
			status: (t) => t.status ?? 0,
			age: (t) => t.block_number ?? 0
		})
	);
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
						<th class="sortable {sort.indicator('hash') || ''}" onclick={() => sort.toggle('hash')}
							>hash</th
						>
						<th
							class="sortable {sort.indicator('method') || ''}"
							onclick={() => sort.toggle('method')}>method</th
						>
						<th class="sortable {sort.indicator('from') || ''}" onclick={() => sort.toggle('from')}
							>from</th
						>
						<th></th>
						<th class="sortable {sort.indicator('to') || ''}" onclick={() => sort.toggle('to')}
							>to</th
						>
						<th
							class="num sortable {sort.indicator('fee') || ''}"
							onclick={() => sort.toggle('fee')}>fee</th
						>
						<th
							class="num sortable {sort.indicator('status') || ''}"
							onclick={() => sort.toggle('status')}>status</th
						>
						<th
							class="num sortable {sort.indicator('age') || ''}"
							onclick={() => sort.toggle('age')}>age</th
						>
					</tr>
				</thead>
				<tbody>
					{#if filtered().length}
						{#each sortedTxs as t (t.hash)}
							<tr>
								<td><TxLink hash={t.hash} /></td>
								<td><span class="badge info">{fmt.methodName(t.sighash)}</span></td>
								<td class="addr"><AddrLink address={t.from_addr} /></td>
								<td class="muted">→</td>
								<td class="addr"
									>{#if t.is_contract_deploy}(new){:else}<AddrLink address={t.to_addr} />{/if}</td
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
						<DataState
							loading={transactions.loading}
							error={transactions.error}
							colspan={8}
							label="transactions"
						/>
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
