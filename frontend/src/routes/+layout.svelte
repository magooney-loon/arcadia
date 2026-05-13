<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();

	let drawerOpen = $state(false);

	const NAV = [
		{
			group: 'Live',
			items: [
				{ id: 'overview', label: 'Overview', href: resolve('/overview/') },
				{ id: 'blocks', label: 'Blocks', href: resolve('/blocks/'), live: true },
				{ id: 'txs', label: 'Transactions', href: resolve('/txs/'), live: true },
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
			group: 'Agents · ERC-8004',
			items: [
				{ id: 'agents', label: 'Agent registry', href: resolve('/agents/') },
				{ id: 'jobs', label: 'Job market', href: resolve('/jobs/') },
				{ id: 'graph', label: 'Wallet graph', href: resolve('/graph/') }
			]
		},
		{
			group: 'Dev',
			items: [{ id: 'debug', label: 'Debug', href: resolve('/debug/') }]
		}
	];

	function isActive(href: string) {
		return $page.url.pathname === href || $page.url.pathname === href.replace(/\/$/, '');
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Arcadia Explorer</title>
</svelte:head>

<div class="app">
	<!-- Logo (desktop) -->
	<a class="logo" href={resolve('/overview/')}>
		<div class="logo-mark"></div>
		<div class="logo-text">ARCADIA<span class="net">·explorer</span></div>
	</a>

	<!-- Topbar -->
	<header class="topbar">
		<!-- Mobile: logo inline -->
		<a class="logo-inline" href={resolve('/overview/')} aria-label="Home">
			<div class="logo-mark" style="width:18px;height:18px"></div>
			<span class="logo-text" style="font-size:12px">ARCADIA</span>
		</a>

		<div class="search">
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
			<input placeholder="Search address, tx, block, agent…" />
			<span class="kbd" aria-hidden="true">⌘K</span>
		</div>

		<div class="topbar-meta">
			<span class="pill"><span class="pulse-dot"></span> arc testnet</span>
			<span class="pill">head <span class="val">#—</span></span>
			<span class="pill">tps <span class="val">—</span></span>
			<span class="pill">block <span class="val">—ms</span></span>
			<span class="pill">indexer <span class="val acc">·</span> 0 lag</span>
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
					<a class="nav-item {isActive(item.href) ? 'active' : ''}" href={item.href}>
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
		<span class="seg"><span class="dot acc"></span> indexer <span class="v ok">live</span></span>
		<span class="seg">rpc <span class="v">arc.rpc.circle.com</span></span>
		<span class="seg">finality <span class="v">single-slot</span></span>
		<span class="seg right">v0.4.2-rc1</span>
		<span class="seg">ws ↔ <span class="v ok">connected</span></span>
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
