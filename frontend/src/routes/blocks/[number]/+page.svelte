<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { blockDetail, fetchBlockDetail } from '$lib/stores/chain.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import TxLink from '$lib/components/TxLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	const number = $derived(Number(page.params.number ?? 0));
	const latestBlock = $derived(stats.data?.block_number ?? 0);

	$effect(() => {
		if (number) {
			blockDetail.data = null;
			fetchBlockDetail(number);
		}
	});

	const data = $derived(blockDetail.data);
	const block = $derived(data?.block);
	const txs = $derived(data?.transactions ?? []);
	const blockStats = $derived(data?.stats as Record<string, unknown> | undefined);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Block #{number}</div>
			<div class="view-sub mono" style="font-size:11px">
				{#if block?.hash}{block.hash}{/if}
			</div>
		</div>
		<div class="view-actions">
			<a
				class="btn ghost"
				href={fmt.explorerBlock(number)}
				target="_blank"
				rel="external noopener noreferrer">view on arcscan ↗</a
			>
		</div>
	</div>

	{#if !block}
		<div class="card">
			<div class="card-body" style="padding:0">
				<DataState loading={blockDetail.loading} error={blockDetail.error} label="block" />
			</div>
		</div>
	{:else}
		<!-- Summary banner -->
		<div
			class="card"
			style="padding:10px 14px;margin-bottom:12px;background:var(--bg-2);border-left:2px solid var(--accent)"
		>
			<div class="grid" style="grid-template-columns:repeat(4,1fr);gap:14px">
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Transactions</div>
					<div class="value" style="font-size:18px">{block.tx_count ?? txs.length}</div>
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Age</div>
					<div class="value" style="font-size:18px">
						{fmt.blockAge(block.number, Number(latestBlock))} ago
					</div>
					<div class="mono dim" style="font-size:10px">{fmt.tsAge(block.timestamp)}</div>
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Gas used</div>
					<div class="value" style="font-size:18px">{fmt.num(block.gas_used)}</div>
					{#if block.gas_limit}
						<div class="mono dim" style="font-size:10px">
							{fmt.pct(block.utilization_pct)} utilization
						</div>
					{/if}
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Block time</div>
					<div class="value" style="font-size:18px">{fmt.ms(block.block_time_ms)}</div>
				</div>
			</div>
		</div>

		<!-- Block fields -->
		<div class="card" style="margin-bottom:12px">
			<div class="card-head">
				<div class="card-title">Block</div>
				<div class="card-sub">raw fields</div>
			</div>
			<div class="card-body" style="padding:0">
				<table class="tbl">
					<tbody>
						<tr>
							<td class="mono muted lbl">Number</td>
							<td class="mono">#{block.number}</td>
						</tr>
						<tr>
							<td class="mono muted lbl">Hash</td>
							<td class="mono" style="word-break:break-all">{block.hash ?? '—'}</td>
						</tr>
						<tr>
							<td class="mono muted lbl">Parent hash</td>
							<td class="mono" style="word-break:break-all">
								{#if block.parent_hash}
									<a href={resolve(`/blocks/${block.number - 1}/`)} style="text-decoration:none"
										>{block.parent_hash}</a
									>
								{:else}
									—
								{/if}
							</td>
						</tr>
						<tr>
							<td class="mono muted lbl">Timestamp</td>
							<td>{block.timestamp ? new Date(block.timestamp * 1000).toISOString() : '—'}</td>
						</tr>
						<tr>
							<td class="mono muted lbl">Miner</td>
							<td class="addr"><AddrLink address={block.miner} /></td>
						</tr>
						<tr>
							<td class="mono muted lbl">Gas used</td>
							<td class="mono">{fmt.num(block.gas_used)}</td>
						</tr>
						<tr>
							<td class="mono muted lbl">Gas limit</td>
							<td class="mono">{fmt.num(block.gas_limit)}</td>
						</tr>
						<tr>
							<td class="mono muted lbl">Base fee/gas</td>
							<td class="mono">{block.base_fee_per_gas ?? '—'}</td>
						</tr>
						<tr>
							<td class="mono muted lbl">Size</td>
							<td class="mono">{block.size ? `${fmt.num(block.size)} bytes` : '—'}</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Block stats (from block_stats collection) -->
		{#if blockStats}
			<div class="card" style="margin-bottom:12px">
				<div class="card-head">
					<div class="card-title">Stats</div>
					<div class="card-sub">pre-aggregated</div>
				</div>
				<div class="card-body" style="padding:0">
					<table class="tbl">
						<tbody>
							<tr>
								<td class="mono muted lbl">TPS</td>
								<td class="mono">{(blockStats.tps as number | null) ?? '—'}</td>
							</tr>
							<tr>
								<td class="mono muted lbl">Total fees</td>
								<td class="mono">{fmt.usdc(blockStats.total_fee_usdc as string | number | null)}</td>
							</tr>
							<tr>
								<td class="mono muted lbl">Avg fee/tx</td>
								<td class="mono">{fmt.usdc(blockStats.avg_fee_usdc as string | number | null, 6)}</td>
							</tr>
							<tr>
								<td class="mono muted lbl">USDC transferred</td>
								<td class="mono">{fmt.usdc(blockStats.total_usdc_transferred as string | number | null)}</td>
							</tr>
							<tr>
								<td class="mono muted lbl">EURC transferred</td>
								<td class="mono">{fmt.usdc(blockStats.total_eurc_transferred as string | number | null)}</td>
							</tr>
							<tr>
								<td class="mono muted lbl">Unique senders</td>
								<td class="mono">{fmt.num(blockStats.unique_senders as number | null)}</td>
							</tr>
							<tr>
								<td class="mono muted lbl">Unique receivers</td>
								<td class="mono">{fmt.num(blockStats.unique_receivers as number | null)}</td>
							</tr>
							<tr>
								<td class="mono muted lbl">New contracts</td>
								<td class="mono">{(blockStats.new_contracts as number | null) ?? 0}</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		{/if}

		<!-- Transactions in this block -->
		{#if txs.length}
			<div class="card">
				<div class="card-head">
					<div class="card-title">Transactions</div>
					<div class="card-sub">{txs.length} tx{txs.length === 1 ? '' : 's'}</div>
				</div>
				<div class="card-body flush">
					<table class="tbl">
						<thead>
							<tr>
								<th>hash</th>
								<th>from</th>
								<th>to</th>
								<th>method</th>
								<th class="num">fee</th>
								<th class="num">status</th>
							</tr>
						</thead>
						<tbody>
							{#each txs as tx (tx.hash)}
								<tr>
									<td class="addr"><TxLink hash={tx.hash} /></td>
									<td class="addr"><AddrLink address={tx.from_addr} /></td>
									<td class="addr">
										<AddrLink address={tx.to_addr ?? tx.contract_address as string | undefined} />
									</td>
									<td class="mono muted" style="font-size:11px"
										>{fmt.methodName(tx.sighash)}</td
									>
									<td class="num">{fmt.usdc(tx.fee_usdc, 6)}</td>
									<td class="num">
										<span style="color:{tx.status === 1 ? 'var(--ok)' : 'var(--err)'}">
											{tx.status === 1 ? '✓' : '✗'}
										</span>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{:else}
			<div class="card"><div class="card-body mono muted">no transactions in this block</div></div>
		{/if}
	{/if}
</div>

<style>
	.lbl {
		width: 160px;
		font-size: 11px;
	}
</style>
