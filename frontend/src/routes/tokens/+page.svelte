<script lang="ts">
	import { onMount } from 'svelte';
	import { tokens, fetchTokens } from '$lib/stores/tokens.svelte';
	import { createSort } from '$lib/sort.svelte';
	import { resolve } from '$app/paths';
	import * as fmt from '$lib/fmt.js';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	let searchQuery = $state('');
	let statusFilter = $state('all');
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;
	const sort = createSort('transfers', 'desc');

	onMount(() => {
		tokens.data = null;
		fetchTokens(500);
	});

	function handleSearchInput() {
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => {
			fetchTokens(500, 0, searchQuery.trim() || undefined);
		}, 300);
	}

	const STATUSES = ['all', 'active', 'failed'];

	const filteredTokens = $derived.by(() => {
		let list = tokens.data?.tokens ?? [];
		if (statusFilter === 'active') list = list.filter((t) => !t.lookup_failed);
		if (statusFilter === 'failed') list = list.filter((t) => t.lookup_failed);
		return list;
	});

	const sortedTokens = $derived(
		sort.apply(filteredTokens, {
			symbol: (t) => (t.symbol || t.name || '').toLowerCase(),
			address: (t) => t.token_address ?? '',
			decimals: (t) => t.decimals ?? 0,
			transfers: (t) => t.transfer_count ?? 0,
			senders: (t) => t.unique_senders ?? 0,
			receivers: (t) => t.unique_receivers ?? 0,
			first_seen: (t) => t.first_seen_block ?? 0,
			status: (t) => (t.lookup_failed ? 1 : 0)
		})
	);

	const totalTransfers = $derived(
		(tokens.data?.tokens ?? []).reduce((sum, t) => sum + (t.transfer_count ?? 0), 0)
	);

	function formatSupply(raw: string | undefined, human: string | undefined): string {
		if (human && human !== '0') return human;
		if (raw && raw !== '0') return raw;
		return '—';
	}
</script>

<svelte:head>
	<title>Tokens · Arcadia Explorer</title>
</svelte:head>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Tokens</div>
			<div class="view-sub">Discovered ERC-20 tokens on Arc testnet</div>
		</div>
	</div>

	<!-- Summary stats -->
	<div class="grid grid-stats">
		<div class="stat">
			<div class="label">Tokens</div>
			<div class="value">{tokens.data?.count ?? '—'}</div>
		</div>
		<div class="stat">
			<div class="label">Total transfers</div>
			<div class="value">{fmt.num(totalTransfers)}</div>
		</div>
		<div class="stat">
			<div class="label">Active</div>
			<div class="value" style="color:var(--ok)">
				{(tokens.data?.tokens ?? []).filter((t) => !t.lookup_failed).length}
			</div>
		</div>
		<div class="stat">
			<div class="label">Failed lookup</div>
			<div class="value" style="color:var(--err)">
				{(tokens.data?.tokens ?? []).filter((t) => t.lookup_failed).length}
			</div>
		</div>
	</div>

	<!-- Filters -->
	<div class="card" style="padding:10px 14px">
		<div class="filter-bar" style="gap:10px;flex-wrap:wrap">
			<input
				bind:value={searchQuery}
				oninput={handleSearchInput}
				placeholder="Search symbol, name, or address…"
				style="flex:1;min-width:200px"
			/>
			{#each STATUSES as s (s)}
				<button
					class="btn ghost {statusFilter === s ? 'active' : ''}"
					onclick={() => (statusFilter = s)}
				>
					{#if s === 'all'}All
					{:else if s === 'active'}✓ Active
					{:else}✗ Failed{/if}
				</button>
			{/each}
		</div>
	</div>

	<!-- Table -->
	<div class="card">
		<div class="table-wrap">
			<table class="tbl">
				<thead>
					<tr>
						<th
							class="sortable {sort.indicator('symbol') || ''}"
							onclick={() => sort.toggle('symbol')}>Token</th
						>
						<th
							class="sortable {sort.indicator('address') || ''}"
							onclick={() => sort.toggle('address')}>Address</th
						>
						<th
							class="sortable {sort.indicator('decimals') || ''}"
							onclick={() => sort.toggle('decimals')}>Dec</th
						>
						<th>Supply</th>
						<th
							class="sortable {sort.indicator('transfers') || ''}"
							onclick={() => sort.toggle('transfers')}>Transfers</th
						>
						<th
							class="sortable {sort.indicator('senders') || ''}"
							onclick={() => sort.toggle('senders')}>Senders</th
						>
						<th
							class="sortable {sort.indicator('receivers') || ''}"
							onclick={() => sort.toggle('receivers')}>Receivers</th
						>
						<th
							class="sortable {sort.indicator('first_seen') || ''}"
							onclick={() => sort.toggle('first_seen')}>First seen</th
						>
						<th
							class="sortable {sort.indicator('status') || ''}"
							onclick={() => sort.toggle('status')}>Status</th
						>
					</tr>
				</thead>
				<tbody>
					{#if sortedTokens.length}
						{#each sortedTokens as token (token.id)}
							<tr>
								<td>
									{#if token.symbol}
										<span class="mono" style="font-weight:600;color:var(--fg-1)"
											>{token.symbol}</span
										>
										{#if token.name}
											<br /><span class="dim" style="font-size:10px">{token.name}</span>
										{/if}
									{:else if token.name}
										<span class="mono" style="font-weight:600;color:var(--fg-1)">{token.name}</span>
									{:else}
										<span class="muted">—</span>
									{/if}
								</td>
								<td><AddrLink address={token.token_address} /></td>
								<td class="mono">{token.decimals ?? '—'}</td>
								<td class="mono" style="font-size:11px"
									>{formatSupply(token.total_supply_raw, token.total_supply_human)}</td
								>
								<td class="mono">{fmt.num(token.transfer_count)}</td>
								<td class="mono">{fmt.num(token.unique_senders)}</td>
								<td class="mono">{fmt.num(token.unique_receivers)}</td>
								<td class="mono">
									{#if token.first_seen_block}
										<a
											href={resolve(`/blocks/${token.first_seen_block}/`)}
											style="text-decoration:none;color:inherit">#{token.first_seen_block}</a
										>
									{:else}—{/if}
								</td>
								<td>
									{#if token.lookup_failed}
										<span style="color:var(--err)">✗</span>
									{:else}
										<span style="color:var(--ok)">✓</span>
									{/if}
								</td>
							</tr>
						{/each}
					{:else}
						<DataState
							loading={tokens.loading}
							error={tokens.error}
							colspan={9}
							label="tokens"
						/>
					{/if}
				</tbody>
			</table>
		</div>
		<div class="table-foot">
			<span class="mono dim" style="font-size:10px"
				>{sortedTokens.length} of {tokens.data?.count ?? 0} tokens</span
			>
		</div>
	</div>
</div>
