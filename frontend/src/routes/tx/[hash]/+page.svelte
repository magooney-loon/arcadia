<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { txDetail, fetchTxDetail } from '$lib/stores/chain.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';

	const hash = $derived(page.params.hash ?? '');
	const latestBlock = $derived(stats.data?.block_number ?? 0);

	$effect(() => {
		if (hash) fetchTxDetail(hash);
	});

	const data = $derived(txDetail.data);
	const tx = $derived(data?.transaction);
	const transfers = $derived(data?.transfers ?? []);
	const traces = $derived(data?.traces ?? []);

	function fieldRow(label: string, value: string | number | undefined | null, mono = true) {
		return { label, value: value ?? '—', mono };
	}

	const txRows = $derived(
		tx
			? [
					fieldRow('Hash', tx.hash),
					fieldRow('Block', tx.block_number, false),
					fieldRow('Status', tx.status === 1 ? '✓ success' : '✗ failed', false),
					fieldRow('From', tx.from_addr),
					fieldRow('To', tx.to_addr ?? (tx.contract_address as string | undefined) ?? '—'),
					fieldRow('Value (raw)', tx.value),
					fieldRow('Nonce', tx.nonce, false),
					fieldRow('Method', fmt.methodName(tx.sighash), false),
					fieldRow('Gas used', tx.gas_used, false),
					fieldRow('Gas limit', tx.gas_limit, false),
					fieldRow('Effective gas price', tx.effective_gas_price),
					fieldRow('Fee', tx.fee_usdc ? `${tx.fee_usdc} USDC` : '—', false),
					fieldRow(
						'Priority fee',
						tx.priority_fee_usdc ? `${tx.priority_fee_usdc} USDC` : '—',
						false
					),
					fieldRow('Tx type', tx.tx_type, false)
				]
			: []
	);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Transaction</div>
			<div class="view-sub mono" style="font-size:11px;word-break:break-all">{hash}</div>
		</div>
		<div class="view-actions">
			<a
				class="btn ghost"
				href={fmt.explorerTx(hash)}
				target="_blank"
				rel="external noopener noreferrer">view on arcscan ↗</a
			>
		</div>
	</div>

	{#if txDetail.loading}
		<div class="card"><div class="card-body mono muted">loading…</div></div>
	{:else if txDetail.error}
		<div class="card"><div class="card-body" style="color:var(--err)">{txDetail.error}</div></div>
	{:else if tx}
		<!-- Status banner -->
		<div
			class="card"
			style="border-left:2px solid {tx.status === 1
				? 'var(--ok)'
				: 'var(--err)'};padding:10px 14px;margin-bottom:12px;background:var(--bg-2)"
		>
			<div class="grid" style="grid-template-columns:repeat(4,1fr);gap:14px">
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Status</div>
					<div class="value" style="font-size:18px;color:{tx.status === 1 ? 'var(--ok)' : 'var(--err)'}">
						{tx.status === 1 ? '✓ success' : '✗ failed'}
					</div>
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Block</div>
					<div class="value" style="font-size:18px">#{tx.block_number}</div>
					<div class="mono dim" style="font-size:10px">
						{fmt.blockAge(tx.block_number, latestBlock)} ago
					</div>
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Fee</div>
					<div class="value" style="font-size:18px;color:var(--warn)">{fmt.usdc(tx.fee_usdc, 6)}</div>
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Gas used</div>
					<div class="value" style="font-size:18px">{fmt.num(tx.gas_used)}</div>
					{#if tx.gas_limit}
						<div class="mono dim" style="font-size:10px">
							of {fmt.num(tx.gas_limit)} ({Math.round(((tx.gas_used ?? 0) / tx.gas_limit) * 100)}%)
						</div>
					{/if}
				</div>
			</div>
		</div>

		<!-- Detail field grid -->
		<div class="card">
			<div class="card-head">
				<div class="card-title">Transaction</div>
				<div class="card-sub">raw fields</div>
			</div>
			<div class="card-body" style="padding:0">
				<table class="tbl">
					<tbody>
						{#each txRows as r (r.label)}
							<tr>
								<td class="mono muted" style="width:160px;font-size:11px">{r.label}</td>
								<td class={r.mono ? 'mono' : ''} style="font-size:11px;word-break:break-all">
									{#if r.label === 'From' || r.label === 'To'}
										{#if typeof r.value === 'string' && r.value.startsWith('0x') && r.value.length === 42}
											<a href={resolve(`/wallet/${r.value}/`)} style="text-decoration:none"
												>{r.value}</a
											>
										{:else}
											{r.value}
										{/if}
									{:else}
										{r.value}
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<!-- Transfers in this tx -->
		{#if transfers.length}
			<div class="card" style="margin-top:12px">
				<div class="card-head">
					<div class="card-title">Token transfers</div>
					<div class="card-sub">{transfers.length} event{transfers.length === 1 ? '' : 's'}</div>
				</div>
				<div class="card-body flush">
					<table class="tbl">
						<thead>
							<tr>
								<th>token</th>
								<th>from</th>
								<th>to</th>
								<th class="num">amount</th>
							</tr>
						</thead>
						<tbody>
							{#each transfers as t, i (i)}
								{@const tt = t as {
									token_symbol?: string;
									from_addr?: string;
									to_addr?: string;
									amount_human?: string;
									amount_raw?: string;
								}}
								<tr>
									<td><span class="badge muted">{tt.token_symbol ?? 'OTHER'}</span></td>
									<td class="addr">
										<AddrLink address={tt.from_addr} />
									</td>
									<td class="addr">
										<AddrLink address={tt.to_addr} />
									</td>
									<td class="num"
										>{tt.amount_human ? fmt.usdc(tt.amount_human) : (tt.amount_raw ?? '—')}</td
									>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}

		<!-- Traces (always empty on Arc — see /traces page note) -->
		{#if traces.length}
			<div class="card" style="margin-top:12px">
				<div class="card-head">
					<div class="card-title">Internal traces</div>
					<div class="card-sub">{traces.length} call{traces.length === 1 ? '' : 's'}</div>
				</div>
				<div class="card-body flush">
					<table class="tbl">
						<thead>
							<tr>
								<th>type</th>
								<th>from</th>
								<th>to</th>
								<th class="num">gas</th>
							</tr>
						</thead>
						<tbody>
							{#each traces as t, i (i)}
								<tr>
									<td><span class="badge muted">{t.call_type ?? t.trace_type ?? '—'}</span></td>
									<td class="addr"
										><AddrLink address={t.from_addr} /></td
									>
									<td class="addr"
										><AddrLink address={t.to_addr} /></td
									>
									<td class="num muted">{fmt.num(t.gas_used)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	{:else}
		<div class="card"><div class="card-body mono muted">transaction not found</div></div>
	{/if}
</div>
