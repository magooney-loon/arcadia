<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import favicon from '$lib/assets/favicon.svg';
	import { stats, fetchStats } from '$lib/stores/stats.svelte';
	import { health, fetchHealth } from '$lib/stores/health.svelte';
	import * as fmt from '$lib/fmt.js';
	import { search, runSearch, clearSearch } from '$lib/stores/search.svelte';
	import { getApiUrl } from '$lib/stores/config.svelte';

	let { children } = $props();
	let drawerOpen = $state(false);
	let searchQuery = $state('');
	let searchFocused = $state(false);
	let searchInput: HTMLInputElement | undefined = $state();

	function handleSearch() {
		const q = searchQuery.trim();
		if (q) runSearch(q);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') handleSearch();
		if (e.key === 'Escape') {
			searchFocused = false;
			searchInput?.blur();
		}
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
		fetchStats();
		fetchHealth();
		const id = setInterval(() => {
			fetchStats();
			fetchHealth();
		}, 8000);
		return () => clearInterval(id);
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
				{ id: 'traces', label: 'Traces', href: resolve('/traces/') }
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
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Arcadia Explorer</title>
</svelte:head>

<div class="app">
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

		<div class="search" class:open={searchFocused || search.data}>
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
				placeholder="Search address, tx, block, agent…"
				onfocus={() => (searchFocused = true)}
				onblur={() => setTimeout(() => (searchFocused = false), 200)}
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
			{#if searchFocused && (search.loading || search.error || search.data)}
				<div class="search-results">
					{#if search.loading}
						<div class="search-result-item muted">searching…</div>
					{:else if search.error}
						<div class="search-result-item err-text">{search.error}</div>
					{:else if search.data?.type === 'not_found'}
						<div class="search-result-item muted">no results found</div>
					{:else if search.data?.type === 'tx' && search.data.result}
						<a
							class="search-result-item"
							href={resolve(`/tx/${search.data.result.hash as string}/`)}
							onclick={() => {
								searchFocused = false;
								searchQuery = '';
								clearSearch();
							}}
						>
							<span class="badge info">tx</span>
							<span class="mono">{fmt.hash(search.data.result.hash as string)}</span>
						</a>
					{:else if search.data?.type === 'block' && search.data.result}
						<a
							class="search-result-item"
							href={fmt.explorerBlock(search.data.result.number as number)}
							target="_blank"
							rel="external noopener noreferrer"
						>
							<span class="badge ok">block</span>
							<span class="mono">#{search.data.result.number}</span>
						</a>
					{:else if search.data?.type === 'wallet' && search.data.result}
						<a
							class="search-result-item"
							href={resolve(`/wallet/${search.data.result.address as string}/`)}
							onclick={() => {
								searchFocused = false;
								searchQuery = '';
								clearSearch();
							}}
						>
							<span class="badge warn">wallet</span>
							<span class="mono">{fmt.addr(search.data.result.address as string)}</span>
						</a>
					{:else if search.data?.type === 'agent' && search.data.result}
						<a
							class="search-result-item"
							href={resolve(`/wallet/${search.data.result.address as string}/`)}
							onclick={() => {
								searchFocused = false;
								searchQuery = '';
								clearSearch();
							}}
						>
							<span class="badge acc">agent</span>
							<span class="mono">{fmt.addr(search.data.result.address as string)}</span>
						</a>
					{:else}
						<div class="search-result-item muted">unknown result</div>
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
		<span class="seg">
			<span class="dot {health.data?.syncing ? 'warn' : 'acc'}"></span>
			indexer
			<span class="v {health.data?.syncing ? '' : 'ok'}"
				>{health.data?.syncing ? 'syncing' : 'live'}</span
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
</style>
