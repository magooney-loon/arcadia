<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import * as fmt from '$lib/fmt.js';
	import { search, runSearch, clearSearch } from '$lib/stores/search.svelte';

	let searchQuery = $state('');
	let searchFocused = $state(false);
	let searchInput: HTMLInputElement | undefined = $state();
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;

	const PAGE_SUGGESTIONS = [
		{
			keywords: ['overview', 'dashboard', 'home', 'stats'],
			label: 'Overview',
			href: resolve('/overview/')
		},
		{ keywords: ['block', 'blocks', 'height'], label: 'Blocks', href: resolve('/blocks/') },
		{
			keywords: ['transaction', 'transactions', 'tx', 'txs'],
			label: 'Transactions',
			href: resolve('/txs/')
		},
		{
			keywords: ['transfer', 'transfers', 'usdc', 'eurc', 'usyc', 'token'],
			label: 'Transfers',
			href: resolve('/transfers/')
		},
		{ keywords: ['trace', 'traces', 'internal'], label: 'Traces', href: resolve('/traces/') },
		{
			keywords: ['crosschain', 'cross-chain', 'bridge', 'cctp', 'gateway'],
			label: 'Cross-chain',
			href: resolve('/crosschain/')
		},
		{
			keywords: ['fx', 'stablefx', 'swap', 'eurc', 'exchange', 'rate'],
			label: 'StableFX',
			href: resolve('/fx/')
		},
		{
			keywords: ['agent', 'agents', 'ai', 'erc-8004', 'erc8004', 'robot'],
			label: 'Agent registry',
			href: resolve('/agents/')
		},
		{
			keywords: ['job', 'jobs', 'escrow', 'task', 'erc-8183', 'erc8183'],
			label: 'Job market',
			href: resolve('/jobs/')
		},
		{
			keywords: ['graph', 'wallet graph', 'network', 'force'],
			label: 'Wallet graph',
			href: resolve('/graph/')
		},
		{
			keywords: ['token', 'tokens', 'erc20', 'erc-20', 'coin'],
			label: 'Tokens',
			href: resolve('/tokens/')
		},
		{
			keywords: ['readme', 'about', 'help', 'docs', 'info'],
			label: 'About Arcadia',
			href: resolve('/readme/')
		},
		{ keywords: ['debug', 'health', 'status'], label: 'Debug', href: resolve('/debug/') }
	];

	const pageSuggestions = $derived.by(() => {
		const q = searchQuery.trim().toLowerCase();
		if (!q) return [];
		const matches = PAGE_SUGGESTIONS.filter((p) =>
			p.keywords.some((k) => k.includes(q) || q.includes(k))
		);
		return matches.slice(0, 3);
	});

	function triggerSearch(q: string) {
		if (!q) return;
		runSearch(q);
	}

	function handleInput() {
		clearTimeout(debounceTimer);
		const q = searchQuery.trim();
		if (!q) {
			clearSearch();
			return;
		}
		debounceTimer = setTimeout(() => triggerSearch(q), 300);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			clearTimeout(debounceTimer);
			triggerSearch(searchQuery.trim());
		}
		if (e.key === 'Escape') {
			searchFocused = false;
			searchInput?.blur();
		}
	}

	function dismissSearch() {
		searchFocused = false;
		searchQuery = '';
		clearSearch();
	}

	export function focus() {
		searchInput?.focus();
		searchFocused = true;
	}

	onMount(() => {
		function globalShortcut(e: KeyboardEvent) {
			if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
				e.preventDefault();
				focus();
			}
		}
		window.addEventListener('keydown', globalShortcut);
		return () => window.removeEventListener('keydown', globalShortcut);
	});
</script>

<div class="search" class:open={searchFocused || search.data || pageSuggestions.length > 0}>
	<svg
		viewBox="0 0 14 14"
		fill="none"
		class="ico"
		stroke="currentColor"
		stroke-width="1.4"
		style="width:14px;height:14px;flex-shrink:0"
	>
		<circle cx="6" cy="6" r="4" /><path d="M9 9 L12 12" />
	</svg>
	<input
		bind:this={searchInput}
		bind:value={searchQuery}
		placeholder="Search tokens, agents, txs, blocks…"
		onfocus={() => (searchFocused = true)}
		onblur={() => setTimeout(() => (searchFocused = false), 200)}
		oninput={handleInput}
		onkeydown={handleKeydown}
	/>
	{#if searchQuery}
		<button
			class="search-clear"
			onclick={() => {
				searchQuery = '';
				clearSearch();
			}}>✕</button
		>
	{:else}
		<span class="kbd" aria-hidden="true">⌘K</span>
	{/if}

	<!-- Search results dropdown -->
	{#if searchFocused && (search.loading || search.error || search.data || pageSuggestions.length > 0)}
		<div class="search-results">
			{#if search.loading}
				<div class="search-result-item muted">searching…</div>
			{:else if search.error}
				<div class="search-result-item err-text">{search.error}</div>
			{:else if search.data?.type === 'not_found'}
				<div class="search-result-item muted">no onchain match — try a page name below</div>
			{:else if search.data?.type === 'tx' && search.data.result}
				<a
					class="search-result-item"
					href={resolve(`/tx/${search.data.result.hash as string}/`)}
					onclick={dismissSearch}
				>
					<span class="badge info">tx</span>
					<span class="mono">{fmt.hash(search.data.result.hash as string)}</span>
				</a>
			{:else if search.data?.type === 'block' && search.data.result}
				<a
					class="search-result-item"
					href={resolve(`/blocks/${search.data.result.number as number}/`)}
					onclick={dismissSearch}
				>
					<span class="badge ok">block</span>
					<span class="mono">#{search.data.result.number}</span>
				</a>
			{:else if search.data?.type === 'wallet' && search.data.result}
				<a
					class="search-result-item"
					href={resolve(`/wallet/${search.data.result.address as string}/`)}
					onclick={dismissSearch}
				>
					<span class="badge warn">wallet</span>
					<span class="mono">{fmt.addr(search.data.result.address as string)}</span>
				</a>
			{:else if search.data?.type === 'agent' && search.data.result}
				<a
					class="search-result-item"
					href={resolve(`/wallet/${search.data.result.address as string}/`)}
					onclick={dismissSearch}
				>
					<span class="badge acc">agent</span>
					<span class="mono">{fmt.addr(search.data.result.address as string)}</span>
				</a>
			{:else if search.data?.type === 'token' && search.data.result}
				<a
					class="search-result-item"
					href={resolve(`/tokens/${search.data.result.token_address as string}/`)}
					onclick={dismissSearch}
				>
					<span class="badge tok">token</span>
					<span class="mono"
						>{fmt.truncate(
							String(search.data.result.symbol || search.data.result.token_address),
							20
						)}</span
					>
				</a>
			{:else if search.data?.type === 'multi'}
				{#if search.data.tokens && search.data.tokens.length > 0}
					<div class="search-section-label">Tokens</div>
					{#each search.data.tokens as t (t.id)}
						<a
							class="search-result-item"
							href={resolve(`/tokens/${t.token_address}/`)}
							onclick={dismissSearch}
						>
							<span class="badge tok">token</span>
							<span>{t.symbol || '???'}</span>
							<span class="dim">{t.name || ''}</span>
						</a>
					{/each}
				{/if}
				{#if search.data.agents && search.data.agents.length > 0}
					{#if search.data.tokens && search.data.tokens.length > 0}
						<div class="search-divider"></div>
					{/if}
					<div class="search-section-label">Agents</div>
					{#each search.data.agents as a (a.id)}
						<a
							class="search-result-item"
							href={resolve(`/wallet/${a.agent_address}/`)}
							onclick={dismissSearch}
						>
							<span class="badge acc">agent</span>
							<span class="mono">{fmt.addr(a.agent_address)}</span>
						</a>
					{/each}
				{/if}
			{:else if search.data?.type === 'unknown'}
				<div class="search-result-item muted">no onchain match — try a page name below</div>
			{/if}

			{#if pageSuggestions.length > 0}
				{#if search.data?.type !== 'not_found' && search.data}
					<div class="search-divider"></div>
				{/if}
				<div class="search-section-label">Pages</div>
				{#each pageSuggestions as p (p.label)}
					<a class="search-result-item" href={p.href} onclick={dismissSearch}>
						<span class="badge dim">page</span>
						<span>{p.label}</span>
					</a>
				{/each}
			{/if}
		</div>
	{/if}
</div>

<style>
	.badge.tok {
		background: rgba(100, 220, 160, 0.12);
		color: #64dca0;
		border-color: rgba(100, 220, 160, 0.25);
	}
	.dim {
		opacity: 0.5;
		font-size: 11px;
		margin-left: 6px;
	}
</style>
