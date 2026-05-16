<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { wallet, fetchWallet } from '$lib/stores/wallet.svelte';
	import { stats } from '$lib/stores/stats.svelte';
	import * as fmt from '$lib/fmt.js';
	import TxLink from '$lib/components/TxLink.svelte';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	const address = $derived(page.params.address ?? '');
	const latestBlock = $derived(stats.data?.block_number ?? 0);

	let tab = $state<'transfers' | 'txs' | 'edges'>('transfers');

	$effect(() => {
		if (address) {
			wallet.data = null;
			fetchWallet(address, 50, 0);
		}
	});

	const data = $derived(wallet.data);
	const isAgent = $derived(data?.is_agent === true);
	const agent = $derived(data?.agent ?? null);

	// Net flow per token from sent + received
	const tokenFlow = $derived(() => {
		const map: Record<string, { sent: number; received: number; net: number }> = {};
		for (const t of data?.sent ?? []) {
			const sym = (t as { token_symbol?: string }).token_symbol ?? 'OTHER';
			const amt = parseFloat((t as { amount_human?: string }).amount_human ?? '0') || 0;
			map[sym] ??= { sent: 0, received: 0, net: 0 };
			map[sym].sent += amt;
			map[sym].net -= amt;
		}
		for (const t of data?.received ?? []) {
			const sym = (t as { token_symbol?: string }).token_symbol ?? 'OTHER';
			const amt = parseFloat((t as { amount_human?: string }).amount_human ?? '0') || 0;
			map[sym] ??= { sent: 0, received: 0, net: 0 };
			map[sym].received += amt;
			map[sym].net += amt;
		}
		return Object.entries(map).sort((a, b) => Math.abs(b[1].net) - Math.abs(a[1].net));
	});
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">
				{isAgent ? 'Agent' : 'Wallet'}
				<span class="mono" style="font-size:13px;color:var(--fg-2)">
					{fmt.addr(address)}
				</span>
			</div>
			<div class="view-sub mono" style="font-size:11px;word-break:break-all">{address}</div>
		</div>
		<div class="view-actions">
			<a
				class="btn ghost"
				href={fmt.explorerAddr(address)}
				target="_blank"
				rel="external noopener noreferrer">view on arcscan ↗</a
			>
		</div>
	</div>

	{#if !data}
		<div class="card">
			<div class="card-body" style="padding:0">
				<DataState loading={wallet.loading} error={wallet.error} label="wallet" />
			</div>
		</div>
	{:else}
		<!-- Agent stats banner (only if registered ERC-8004 agent) -->
		{#if isAgent && agent}
			<div
				class="card"
				style="border-left:2px solid var(--acc);padding:10px 14px;margin-bottom:12px;background:var(--bg-2)"
			>
				<div class="grid" style="grid-template-columns:repeat(4,1fr);gap:14px">
					<div class="stat" style="padding:0;background:transparent;border:0">
						<div class="label">Registered</div>
						<div class="value" style="font-size:18px">#{agent.registered_at_block ?? '—'}</div>
					</div>
					<div class="stat" style="padding:0;background:transparent;border:0">
						<div class="label">Txs</div>
						<div class="value" style="font-size:18px">{fmt.num(agent.tx_count)}</div>
					</div>
					<div class="stat" style="padding:0;background:transparent;border:0">
						<div class="label">Volume transferred</div>
						<div class="value" style="font-size:18px;color:var(--ok)">
							{fmt.usdc(agent.usdc_transferred_human as string | undefined)}
						</div>
					</div>
					<div class="stat" style="padding:0;background:transparent;border:0">
						<div class="label">Fees spent</div>
						<div class="value" style="font-size:18px;color:var(--warn)">
							{fmt.usdc(agent.usdc_spent_fees_human as string | undefined, 4)}
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- Per-token flow summary -->
		{#if tokenFlow().length}
			<div class="grid" style="grid-template-columns:repeat({Math.min(tokenFlow().length, 4)},1fr);margin-bottom:12px">
				{#each tokenFlow().slice(0, 4) as [sym, flow] (sym)}
					<div class="stat">
						<div class="label">{sym} net · recent 50</div>
						<div class="value" style="color:{flow.net >= 0 ? 'var(--ok)' : 'var(--err)'}">
							{flow.net >= 0 ? '+' : ''}{fmt.usdc(flow.net)}
						</div>
						<div class="mono dim" style="font-size:10px">
							in {fmt.usdc(flow.received)} · out {fmt.usdc(flow.sent)}
						</div>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Tabs -->
		<div class="filter-bar" style="margin-bottom:10px">
			<button
				class="btn {tab === 'transfers' ? 'acc' : 'ghost'}"
				onclick={() => (tab = 'transfers')}
				>transfers ({(data.sent?.length ?? 0) + (data.received?.length ?? 0)})</button
			>
			<button class="btn {tab === 'txs' ? 'acc' : 'ghost'}" onclick={() => (tab = 'txs')}
				>txs ({(data.txs_sent?.length ?? 0) + (data.txs_received?.length ?? 0)})</button
			>
			<button class="btn {tab === 'edges' ? 'acc' : 'ghost'}" onclick={() => (tab = 'edges')}
				>edges ({(data.outgoing_edges?.length ?? 0) + (data.incoming_edges?.length ?? 0)})</button
			>
		</div>

		{#if tab === 'transfers'}
			<div class="card">
				<div class="card-body flush">
					<table class="tbl">
						<thead>
							<tr>
								<th>dir</th>
								<th>token</th>
								<th>counterparty</th>
								<th class="num">amount</th>
								<th>tx</th>
								<th class="num">age</th>
							</tr>
						</thead>
						<tbody>
							{#each [...(data.received ?? []).map((t) => ({ ...t, dir: 'in' as const })), ...(data.sent ?? []).map((t) => ({ ...t, dir: 'out' as const }))].sort((a, b) => ((b as { block_number?: number }).block_number ?? 0) - ((a as { block_number?: number }).block_number ?? 0)) as t (((t as { id?: string }).id ?? '') + (t as { dir: string }).dir)}
								{@const tt = t as {
									dir: 'in' | 'out';
									tx_hash?: string;
									token_symbol?: string;
									from_addr?: string;
									to_addr?: string;
									amount_human?: string;
									block_number?: number;
								}}
								<tr>
									<td
										><span class="badge {tt.dir === 'in' ? 'ok' : 'err'}"
											>{tt.dir === 'in' ? '↘ in' : '↗ out'}</span
										></td
									>
									<td><span class="badge muted">{tt.token_symbol ?? 'OTHER'}</span></td>
									<td class="addr">
										<a
											href={resolve(
												`/wallet/${tt.dir === 'in' ? tt.from_addr : tt.to_addr}/`
											)}
											style="text-decoration:none"
										>
											{fmt.addr(tt.dir === 'in' ? tt.from_addr : tt.to_addr)}
										</a>
									</td>
									<td class="num">{fmt.usdc(tt.amount_human)}</td>
									<td>
										<TxLink hash={tt.tx_hash} />
									</td>
									<td class="num muted">{fmt.blockAge(tt.block_number, latestBlock)}</td>
								</tr>
							{/each}
							{#if !(data.sent?.length || data.received?.length)}
								<tr><td colspan="6" class="mono muted" style="text-align:center;padding:32px">no transfers</td></tr>
							{/if}
						</tbody>
					</table>
				</div>
			</div>
		{:else if tab === 'txs'}
			<div class="card">
				<div class="card-body flush">
					<table class="tbl">
						<thead>
							<tr>
								<th>dir</th>
								<th>tx</th>
								<th>method</th>
								<th>counterparty</th>
								<th class="num">fee</th>
								<th class="num">age</th>
							</tr>
						</thead>
						<tbody>
							{#each [...(data.txs_received ?? []).map((t) => ({ ...t, dir: 'in' as const })), ...(data.txs_sent ?? []).map((t) => ({ ...t, dir: 'out' as const }))].sort((a, b) => ((b as { block_number?: number }).block_number ?? 0) - ((a as { block_number?: number }).block_number ?? 0)) as t (((t as { id?: string }).id ?? '') + (t as { dir: string }).dir)}
								{@const tt = t as {
									dir: 'in' | 'out';
									hash?: string;
									sighash?: string;
									from_addr?: string;
									to_addr?: string;
									fee_usdc?: string;
									block_number?: number;
								}}
								<tr>
									<td
										><span class="badge {tt.dir === 'in' ? 'ok' : 'err'}"
											>{tt.dir === 'in' ? '↘ in' : '↗ out'}</span
										></td
									>
									<td>
										<TxLink hash={tt.hash} />
									</td>
									<td><span class="mono" style="font-size:11px">{fmt.methodName(tt.sighash)}</span></td>
									<td class="addr">
										<AddrLink address={tt.dir === 'in' ? tt.from_addr : tt.to_addr} />
									</td>
									<td class="num muted">{fmt.usdc(tt.fee_usdc, 5)}</td>
									<td class="num muted">{fmt.blockAge(tt.block_number, latestBlock)}</td>
								</tr>
							{/each}
							{#if !(data.txs_sent?.length || data.txs_received?.length)}
								<tr><td colspan="6" class="mono muted" style="text-align:center;padding:32px">no transactions</td></tr>
							{/if}
						</tbody>
					</table>
				</div>
			</div>
		{:else}
			<div class="card">
				<div class="card-body flush">
					<table class="tbl">
						<thead>
							<tr>
								<th>dir</th>
								<th>counterparty</th>
								<th class="num">total USDC</th>
								<th class="num">txs</th>
								<th class="num">last seen</th>
							</tr>
						</thead>
						<tbody>
							{#each [...(data.outgoing_edges ?? []).map((e) => ({ ...e, dir: 'out' as const })), ...(data.incoming_edges ?? []).map((e) => ({ ...e, dir: 'in' as const }))].sort((a, b) => (b.tx_count ?? 0) - (a.tx_count ?? 0)) as e (e.id + e.dir)}
								<tr>
									<td
										><span class="badge {e.dir === 'in' ? 'ok' : 'err'}"
											>{e.dir === 'in' ? '↘ in' : '↗ out'}</span
										></td
									>
									<td class="addr">
										<a
											href={resolve(
												`/wallet/${e.dir === 'in' ? e.from_wallet : e.to_wallet}/`
											)}
											style="text-decoration:none"
										>
											{fmt.addr(e.dir === 'in' ? e.from_wallet : e.to_wallet)}
										</a>
									</td>
									<td class="num">{fmt.usdc(e.total_usdc_human)}</td>
									<td class="num muted">{fmt.num(e.tx_count)}</td>
									<td class="num muted">{fmt.blockAge(e.last_seen_block, latestBlock)}</td>
								</tr>
							{/each}
							{#if !(data.outgoing_edges?.length || data.incoming_edges?.length)}
								<tr><td colspan="5" class="mono muted" style="text-align:center;padding:32px">no edges</td></tr>
							{/if}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	{/if}
</div>
