<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { page, navigating } from '$app/state';
	import { resolve } from '$app/paths';
	import favicon from '$lib/assets/favicon.svg';
	import { stats, fetchStats } from '$lib/stores/stats.svelte';
	import { health, fetchHealth } from '$lib/stores/health.svelte';
	import * as fmt from '$lib/fmt.js';
	import { search, runSearch, clearSearch } from '$lib/stores/search.svelte';
	import { getApiUrl } from '$lib/stores/config.svelte';
	import { connectRealtime, disconnectRealtime } from '$lib/realtime';

	let { children } = $props();
	let drawerOpen = $state(false);
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
		// Only hit the API if the query could be an exact entity
		const couldBeEntity =
			(q.startsWith('0x') && (q.length === 42 || q.length === 66)) || /^\d+$/.test(q);
		if (couldBeEntity) {
			runSearch(q);
		} else {
			// Non-entity queries: clear previous API results so we just show page suggestions
			search.data = null;
			search.error = null;
			search.loading = false;
		}
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

	onMount(() => {
		function globalShortcut(e: KeyboardEvent) {
			if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
				e.preventDefault();
				searchInput?.focus();
				searchFocused = true;
			}
		}
		window.addEventListener('keydown', globalShortcut);
		return () => window.removeEventListener('keydown', globalShortcut);
	});

	onMount(() => {
		// One-shot REST fetch for instant first paint; SSE takes over after.
		fetchStats();
		fetchHealth();
		connectRealtime();
		return () => {
			disconnectRealtime();
		};
	});

	interface NavItem {
		id: string;
		label: string;
		href: string;
		live?: boolean;
		external?: boolean;
	}

	const NAV: { group: string; items: NavItem[] }[] = [
		{
			group: 'Data',
			items: [
				{ id: 'overview', label: 'Overview', href: resolve('/overview/') },
				{ id: 'blocks', label: 'Blocks', href: resolve('/blocks/'), live: false },
				{ id: 'txs', label: 'Transactions', href: resolve('/txs/'), live: false },
				{ id: 'transfers', label: 'Transfers', href: resolve('/transfers/') },
				{ id: 'traces', label: 'Traces', href: resolve('/traces/') },
				{ id: 'tokens', label: 'Tokens', href: resolve('/tokens/') }
			]
		},
		{
			group: 'Flows',
			items: [
				{ id: 'crosschain', label: 'Cross-chain', href: resolve('/crosschain/') },
				{ id: 'fx', label: 'StableFX', href: resolve('/fx/') }
			]
		},
		{
			group: 'Agents',
			items: [
				{ id: 'agents', label: 'Agent registry', href: resolve('/agents/') },
				{ id: 'jobs', label: 'Job market', href: resolve('/jobs/') },
				{ id: 'graph', label: 'Wallet graph', href: resolve('/graph/') }
			]
		},
		{
			group: 'Dev',
			items: [
				{ id: 'readme', label: 'README', href: resolve('/readme/') },
				{ id: 'debug', label: 'Debug', href: resolve('/debug/') },
				{
					id: 'openapi',
					label: 'OpenAPI',
					href: getApiUrl() + '/api/docs/v1/swagger',
					external: true
				}
			]
		}
	];

	function isActive(href: string) {
		return page.url.pathname === href || page.url.pathname === href.replace(/\/$/, '');
	}

	// Fullscreen "indexer is syncing" gate. Shown when the indexer is far
	// enough behind that the writer is sprinting and SQLite reads stall —
	// during that window any clicks the user makes either fail outright or
	// queue up handlers that pile pressure on the lock.
	//
	// Hysteresis: show at lag > 100, hide at lag <= 20. Without it the
	// modal would flicker on/off as lag oscillates around a single
	// threshold during the final approach to tip.
	const SYNC_MODAL_SHOW = 100;
	const SYNC_MODAL_HIDE = 20;
	let syncModalOpen = $state(false);
	$effect(() => {
		const lag = health.data?.lag_blocks ?? 0;
		if (!health.data) return;
		if (syncModalOpen) {
			if (lag <= SYNC_MODAL_HIDE) syncModalOpen = false;
		} else {
			if (lag > SYNC_MODAL_SHOW) syncModalOpen = true;
		}
	});

	const syncProgressPct = $derived.by(() => {
		const tip = health.data?.chain_tip ?? 0;
		const head = health.data?.last_indexed_block ?? 0;
		if (!tip || head > tip) return 0;
		// Show how far through the visible sync window we are. We don't know
		// the original gap, so anchor on the modal threshold: 100 blocks
		// behind = 0%, 20 blocks behind (close enough to dismiss) = 100%.
		const lag = health.data?.lag_blocks ?? 0;
		const span = SYNC_MODAL_SHOW - SYNC_MODAL_HIDE;
		const remaining = Math.max(0, Math.min(span, lag - SYNC_MODAL_HIDE));
		return Math.round(((span - remaining) / span) * 100);
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Arcadia Explorer</title>
</svelte:head>

<div class="app">
	<!-- Navigation progress bar — visible while SvelteKit is loading
	     the target route's +page.ts load function (or JS chunk). -->
	{#if navigating}
		<div class="nav-bar" aria-hidden="true"></div>
	{/if}

	<!-- Logo (desktop) -->
	<a class="logo" href={resolve('/overview/')}>
		<div class="logo-mark">
			<svg viewBox="0 0 32 32" fill="none" style="width:100%;height:100%"
				><path
					d="M16 2A14 14 0 0 0 4.1 22.5L7.8 21A10 10 0 0 1 16 6V2Z"
					fill="var(--accent)"
				/><path
					d="M4.1 22.5A14 14 0 0 0 28 20L24.8 17.5A10 10 0 0 1 7.8 21L4.1 22.5Z"
					fill="var(--accent)"
					opacity="0.5"
				/><circle cx="16" cy="16" r="3" fill="var(--accent)" /></svg
			>
		</div>
		<div class="logo-text">ARCADIA<span class="net">·explorer</span></div>
	</a>

	<!-- Topbar -->
	<header class="topbar">
		<!-- Mobile: logo inline -->
		<a class="logo-inline" href={resolve('/overview/')} aria-label="Home">
			<div class="logo-mark" style="width:18px;height:18px">
				<svg viewBox="0 0 32 32" fill="none" style="width:100%;height:100%"
					><path
						d="M16 2A14 14 0 0 0 4.1 22.5L7.8 21A10 10 0 0 1 16 6V2Z"
						fill="var(--accent)"
					/><path
						d="M4.1 22.5A14 14 0 0 0 28 20L24.8 17.5A10 10 0 0 1 7.8 21L4.1 22.5Z"
						fill="var(--accent)"
						opacity="0.5"
					/><circle cx="16" cy="16" r="3" fill="var(--accent)" /></svg
				>
			</div>
			<span class="logo-text" style="font-size:12px">ARCADIA</span>
		</a>

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
				placeholder="Search address, tx, block, page…"
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

		<div class="topbar-meta">
			<span class="pill"><span class="pulse-dot"></span> arc testnet</span>
			<span class="pill">head <span class="val">#{stats.data?.indexed_block ?? '—'}</span></span>
			<span class="pill">tps <span class="val">{fmt.tps(stats.data?.tps)}</span></span>
			<span class="pill">block <span class="val">{fmt.ms(stats.data?.block_time_ms)}</span></span>
			<span class="pill"
				>lag <span class="val {(health.data?.lag_blocks ?? 0) > 50 ? 'warn' : 'acc'}"
					>{health.data?.lag_blocks ?? '—'}</span
				></span
			>
		</div>

		<button class="hamburger" onclick={() => (drawerOpen = true)} aria-label="Open navigation">
			<span></span><span></span><span></span>
		</button>
	</header>

	<!-- Sidebar (desktop) -->
	<aside class="sidebar">
		{#each NAV as group (group.group)}
			<div class="nav-group">
				<div class="nav-label">{group.group}</div>
				{#each group.items as item (item.id)}
					<a
						class="nav-item {isActive(item.href) ? 'active' : ''}"
						href={item.href}
						target={item.external ? '_blank' : undefined}
						rel="external noopener noreferrer"
					>
						{@render NavIcon({ id: item.id })}
						<span>{item.label}</span>
						{#if item.live}
							<span class="count">live</span>
						{/if}
					</a>
				{/each}
			</div>
		{/each}
	</aside>

	<!-- Main -->
	<main class="main">
		{@render children()}
	</main>

	<!-- Status bar (desktop) -->
	<footer class="statusbar">
		<span
			class="seg"
			title={health.data?.syncing
				? `Indexer is catching up · ${health.data?.lag_blocks ?? '?'} blocks behind. Some data may refresh slowly until sync completes.`
				: 'Indexer is at chain tip'}
		>
			<span
				class="dot {health.data?.syncing ? 'warn' : 'acc'}"
				class:syncing-pulse={health.data?.syncing}
			></span>
			indexer
			<span class="v {health.data?.syncing ? 'warn' : 'ok'}"
				>{health.data?.syncing
					? `syncing · ${health.data?.lag_blocks ?? '?'} behind`
					: 'live'}</span
			>
		</span>
		<span class="seg">errors/h <span class="v">{health.data?.errors_1h ?? '—'}</span></span>
		<span class="seg"
			>batch <span class="v"
				>{health.data?.avg_batch_ms ? Math.round(health.data.avg_batch_ms) + 'ms' : '—'}</span
			></span
		>
		<span class="seg right">v0.4.2-rc1</span>
		<span class="seg"
			><a
				href="https://envio.dev/"
				target="_blank"
				rel="external noopener noreferrer"
				style="text-decoration:none;color:inherit">HyperSync ↗</a
			></span
		>
	</footer>
</div>

<!-- Fullscreen indexer-syncing gate. Blocks all input until the indexer
     catches up enough (lag <= SYNC_MODAL_HIDE blocks). aria-modal + focus
     trap left intentionally simple: this is a transient state and the
     user can't actually do anything productive underneath. -->
{#if syncModalOpen}
	<div
		class="sync-modal"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="sync-modal-title"
		aria-describedby="sync-modal-desc"
	>
		<div class="sync-card">
			<div class="sync-pulse-ring">
				<svg viewBox="0 0 40 40" fill="none" aria-hidden="true">
					<circle cx="20" cy="20" r="14" stroke="var(--accent)" stroke-width="2" opacity="0.3" />
					<circle cx="20" cy="20" r="14" stroke="var(--accent)" stroke-width="2"
						stroke-dasharray="22 88" stroke-linecap="round" class="sync-spin" />
				</svg>
			</div>
			<div class="sync-title" id="sync-modal-title">Indexer is catching up</div>
			<div class="sync-desc" id="sync-modal-desc">
				Arcadia is replaying recent blocks. The dashboard is paused so the indexer
				can finish without read contention. This usually clears within a minute.
			</div>

			<div class="sync-stats">
				<div class="sync-stat">
					<div class="sync-stat-label">behind by</div>
					<div class="sync-stat-val warn">{health.data?.lag_blocks ?? '—'}<span class="sync-unit"> blocks</span></div>
				</div>
				<div class="sync-stat">
					<div class="sync-stat-label">indexed</div>
					<div class="sync-stat-val mono">#{health.data?.last_indexed_block ?? '—'}</div>
				</div>
				<div class="sync-stat">
					<div class="sync-stat-label">chain tip</div>
					<div class="sync-stat-val mono">#{health.data?.chain_tip ?? '—'}</div>
				</div>
			</div>

			<div class="sync-progress" aria-hidden="true">
				<div class="sync-progress-fill" style="width:{syncProgressPct}%"></div>
			</div>

			<div class="sync-footnote">
				Live updates over SSE — the page will resume automatically.
			</div>
		</div>
	</div>
{/if}

<!-- Mobile drawer overlay -->
{#if drawerOpen}
	<button
		class="sidebar-overlay"
		onclick={() => (drawerOpen = false)}
		aria-label="Close navigation"
		tabindex="-1"
	></button>
{/if}

<aside class="sidebar-drawer {drawerOpen ? 'open' : ''}" aria-hidden={!drawerOpen}>
	<div style="padding:14px 18px 10px;border-bottom:1px solid var(--border-1);margin-bottom:4px">
		<div class="logo-text">ARCADIA<span class="net">·explorer</span></div>
	</div>
	{#each NAV as group (group.group)}
		<div class="nav-group">
			<div class="nav-label">{group.group}</div>
			{#each group.items as item (item.id)}
				<a
					class="nav-item {isActive(item.href) ? 'active' : ''}"
					href={item.href}
					onclick={() => (drawerOpen = false)}
					target={item.external ? '_blank' : undefined}
					rel="external noopener noreferrer"
				>
					{@render NavIcon({ id: item.id })}
					<span>{item.label}</span>
					{#if item.live}
						<span class="count">live</span>
					{/if}
				</a>
			{/each}
		</div>
	{/each}
</aside>

{#snippet NavIcon({ id }: { id: string })}
	{#if id === 'overview'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><rect x="2" y="2" width="4" height="4" /><rect x="8" y="2" width="4" height="4" /><rect
				x="2"
				y="8"
				width="4"
				height="4"
			/><rect x="8" y="8" width="4" height="4" /></svg
		>
	{:else if id === 'blocks'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><path d="M7 1 L13 4 L7 7 L1 4 Z" /><path d="M1 4 L1 10 L7 13 L13 10 L13 4" /><path
				d="M7 7 L7 13"
			/></svg
		>
	{:else if id === 'txs'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><path d="M2 4 H10 M8 2 L10 4 L8 6 M12 10 H4 M6 8 L4 10 L6 12" /></svg
		>
	{:else if id === 'transfers'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><circle cx="7" cy="7" r="5" /><path d="M5 6 L7 4 L9 6 M7 4 L7 10" /></svg
		>
	{:else if id === 'traces'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><path d="M2 2 V12 H12 M4 9 L7 6 L9 8 L12 4" /></svg
		>
	{:else if id === 'tokens'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><circle cx="7" cy="7" r="5" /><path
				d="M5 7.5 L6.5 9 L9 5.5"
				stroke-linecap="round"
				stroke-linejoin="round"
			/></svg
		>
	{:else if id === 'crosschain'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><circle cx="3.5" cy="7" r="2" /><circle cx="10.5" cy="7" r="2" /><path
				d="M5.5 7 H8.5"
			/></svg
		>
	{:else if id === 'fx'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><path d="M2 3 L11 3 M9 1 L11 3 L9 5 M12 11 L3 11 M5 9 L3 11 L5 13" /></svg
		>
	{:else if id === 'agents'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><rect x="3" y="3" width="8" height="8" rx="1" /><circle
				cx="5.5"
				cy="6"
				r="0.6"
				fill="currentColor"
			/><circle cx="8.5" cy="6" r="0.6" fill="currentColor" /><path d="M5 9 H9" /><path
				d="M7 1 V3 M7 11 V13 M1 7 H3 M11 7 H13"
			/></svg
		>
	{:else if id === 'jobs'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><rect x="2" y="4" width="10" height="8" rx="0.5" /><path
				d="M5 4 V2.5 Q5 2 5.5 2 H8.5 Q9 2 9 2.5 V4"
			/></svg
		>
	{:else if id === 'graph'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><circle cx="3" cy="3" r="1.5" /><circle cx="11" cy="3" r="1.5" /><circle
				cx="7"
				cy="11"
				r="1.5"
			/><circle cx="11" cy="9" r="1.2" /><path d="M4 4 L6 10 M10 4 L8 10 M11 4.5 L11 7.5" /></svg
		>
	{:else if id === 'readme'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><rect x="2" y="1" width="10" height="12" rx="1" /><path
				d="M4 4 H10 M4 6.5 H10 M4 9 H7.5"
			/></svg
		>
	{:else if id === 'debug'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><path
				d="M5 2 H9 M4 5 H10 M4 9 H10 M7 5 V9 M3 3 L1 5 M11 3 L13 5 M1 9 L3 11 M13 9 L11 11 M4 11 Q4 13 7 13 Q10 13 10 11"
			/></svg
		>
	{:else if id === 'openapi'}
		<svg viewBox="0 0 14 14" fill="none" class="ico" stroke="currentColor" stroke-width="1.4"
			><circle cx="7" cy="4" r="2" /><path d="M3 12 L5 6 M9 6 L11 12 M5 8 H9" /></svg
		>
	{/if}
{/snippet}

<style>
	/* logo inline — mobile only */
	.logo-inline {
		display: none;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	@media (max-width: 767px) {
		.logo-inline {
			display: flex;
		}
	}

	.ico {
		width: 14px;
		height: 14px;
		flex-shrink: 0;
		color: var(--fg-3);
	}
	.nav-item.active .ico {
		color: var(--accent);
	}
	.syncing-pulse {
		animation: sync-pulse 1.4s ease-in-out infinite;
	}
	@keyframes sync-pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.35;
		}
	}

	/* Navigation progress bar — thin bar at the top of the viewport
	   that animates while a route is transitioning. Grows quickly
	   to 60% then slows down; disappears when navigation completes. */
	.nav-bar {
		position: fixed;
		top: 0;
		left: 0;
		height: 2px;
		width: 100%;
		z-index: 9999;
		background: linear-gradient(90deg, var(--accent), var(--info));
		transform-origin: left;
		animation: nav-bar-grow 1.2s ease-out forwards;
		pointer-events: none;
	}
	@keyframes nav-bar-grow {
		0% {
			transform: scaleX(0);
		}
		40% {
			transform: scaleX(0.6);
		}
		100% {
			transform: scaleX(1);
		}
	}

	/* Fullscreen indexer-sync gate. z-index sits above .nav-bar (9999)
	   and the mobile drawer so it always wins. */
	.sync-modal {
		position: fixed;
		inset: 0;
		z-index: 10000;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 24px;
		background: rgba(8, 12, 18, 0.78);
		backdrop-filter: blur(6px);
		-webkit-backdrop-filter: blur(6px);
		animation: sync-modal-in 180ms ease-out;
	}
	@keyframes sync-modal-in {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}
	.sync-card {
		max-width: 440px;
		width: 100%;
		background: var(--bg-1);
		border: 1px solid var(--border-1);
		border-radius: 8px;
		padding: 28px 28px 22px;
		box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
		text-align: center;
	}
	.sync-pulse-ring {
		width: 56px;
		height: 56px;
		margin: 0 auto 18px;
	}
	.sync-pulse-ring svg {
		width: 100%;
		height: 100%;
	}
	.sync-spin {
		transform-origin: center;
		animation: sync-spin 1.1s linear infinite;
	}
	@keyframes sync-spin {
		to {
			transform: rotate(360deg);
		}
	}
	.sync-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--fg-0);
		letter-spacing: 0.2px;
		margin-bottom: 6px;
	}
	.sync-desc {
		font-size: 12px;
		line-height: 1.55;
		color: var(--fg-3);
		margin: 0 auto 18px;
		max-width: 360px;
	}
	.sync-stats {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 8px;
		padding: 12px 0;
		border-top: 1px solid var(--border-1);
		border-bottom: 1px solid var(--border-1);
		margin-bottom: 14px;
	}
	.sync-stat-label {
		font-size: 9px;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		color: var(--fg-3);
		margin-bottom: 3px;
	}
	.sync-stat-val {
		font-size: 14px;
		font-weight: 600;
		color: var(--fg-0);
		font-variant-numeric: tabular-nums;
	}
	.sync-stat-val.warn {
		color: var(--warn);
	}
	.sync-stat-val.mono {
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 12px;
		font-weight: 500;
	}
	.sync-unit {
		font-size: 10px;
		font-weight: 400;
		color: var(--fg-3);
		margin-left: 2px;
	}
	.sync-progress {
		height: 3px;
		background: var(--border-1);
		border-radius: 2px;
		overflow: hidden;
		margin-bottom: 10px;
	}
	.sync-progress-fill {
		height: 100%;
		background: linear-gradient(90deg, var(--accent), var(--info));
		transition: width 400ms ease-out;
	}
	.sync-footnote {
		font-size: 10px;
		color: var(--fg-3);
		opacity: 0.7;
	}
</style>
