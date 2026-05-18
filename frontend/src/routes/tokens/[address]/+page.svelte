<script lang="ts">
	import { page } from '$app/state';
	import { tokenDetail } from '$lib/stores/tokens.svelte';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import TxLink from '$lib/components/TxLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	const address = $derived(page.params.address ?? '');

	const data = $derived(tokenDetail.data);
	const token = $derived(data?.token);
	const transfers = $derived(data?.transfers ?? []);

	function formatSupply(raw: string | undefined, human: string | undefined): string {
		if (human && human !== '0') return human;
		if (raw && raw !== '0') return raw;
		return '—';
	}
</script>

<svelte:head>
	<title>{token ? `${token.symbol || token.name || 'Token'} · Arcadia` : 'Token · Arcadia'}</title>
</svelte:head>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">{token ? token.symbol || token.name || 'Token' : 'Token'}</div>
			<div class="view-sub mono" style="font-size:11px">
				{#if token}{token.token_address}{/if}
			</div>
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

	{#if !token}
		<div class="card">
			<div class="card-body" style="padding:0">
				<DataState loading={tokenDetail.loading} error={tokenDetail.error} label="token" />
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
					<div class="label">Symbol</div>
					<div class="value" style="font-size:18px">{token.symbol || '—'}</div>
					{#if token.name}
						<div class="mono dim" style="font-size:10px">{token.name}</div>
					{/if}
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Transfers</div>
					<div class="value" style="font-size:18px">{fmt.num(token.transfer_count)}</div>
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Unique senders</div>
					<div class="value" style="font-size:18px">{fmt.num(token.unique_senders)}</div>
				</div>
				<div class="stat" style="padding:0;background:transparent;border:0">
					<div class="label">Unique receivers</div>
					<div class="value" style="font-size:18px">{fmt.num(token.unique_receivers)}</div>
				</div>
			</div>
		</div>

		<!-- Token details -->
		<div class="card" style="margin-bottom:12px">
			<div class="card-head">
				<span>Token details</span>
			</div>
			<div class="table-wrap">
				<table class="tbl">
					<tbody>
						<tr>
							<td class="label-cell">Address</td>
							<td><AddrLink address={token.token_address} /></td>
						</tr>
						<tr>
							<td class="label-cell">Name</td>
							<td>{token.name || '—'}</td>
						</tr>
						<tr>
							<td class="label-cell">Symbol</td>
							<td>{token.symbol || '—'}</td>
						</tr>
						<tr>
							<td class="label-cell">Decimals</td>
							<td class="mono">{token.decimals ?? '—'}</td>
						</tr>
						<tr>
							<td class="label-cell">Total supply</td>
							<td class="mono">{formatSupply(token.total_supply_raw, token.total_supply_human)}</td>
						</tr>
						<tr>
							<td class="label-cell">Token type</td>
							<td
								>{#if token.lookup_failed}<span style="color:var(--err)">lookup failed</span
									>{:else}<span style="color:var(--ok)">verified</span>{/if}</td
							>
						</tr>
						<tr>
							<td class="label-cell">First seen</td>
							<td class="mono">#{token.first_seen_block ?? '—'}</td>
						</tr>
						<tr>
							<td class="label-cell">Last seen</td>
							<td class="mono">#{token.last_seen_block ?? '—'}</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Recent transfers -->
		<div class="card">
			<div class="card-head">
				<span>Recent transfers</span>
				<span class="badge dim">{transfers.length}</span>
			</div>
			<div class="table-wrap">
				<table class="tbl">
					<thead>
						<tr>
							<th>Tx</th>
							<th>Block</th>
							<th>From</th>
							<th>To</th>
							<th>Amount</th>
						</tr>
					</thead>
					<tbody>
						{#if transfers.length}
							{#each transfers as t (t.id)}
								<tr>
									<td><TxLink hash={t.tx_hash} /></td>
									<td class="mono">#{t.block_number}</td>
									<td><AddrLink address={t.from_addr} /></td>
									<td><AddrLink address={t.to_addr} /></td>
									<td class="mono" style="font-size:11px">{t.amount_human || t.amount_raw}</td>
								</tr>
							{/each}
						{:else}
							<tr>
								<td colspan="5" class="muted" style="text-align:center;padding:20px"
									>No transfers found</td
								>
							</tr>
						{/if}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<style>
	.label-cell {
		color: var(--fg-3);
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.4px;
		white-space: nowrap;
		width: 140px;
	}
</style>
