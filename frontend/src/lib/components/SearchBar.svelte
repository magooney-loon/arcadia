<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { flip } from 'svelte/animate';
	import { fade, slide } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import * as fmt from '$lib/fmt.js';
	import { DOMAIN_NAMES } from '$lib/fmt.js';
	import { search, runSearch, clearSearch } from '$lib/stores/search.svelte';

	let searchQuery = $state('');
	let searchFocused = $state(false);
	let searchInput: HTMLInputElement | undefined = $state();
	let debounceTimer: ReturnType<typeof setTimeout> | undefined;
	let activeIndex = $state(0);
	let recents = $state<string[]>([]);

	const RECENTS_KEY = 'arcadia:search:recents';
	const RECENTS_MAX = 6;

	type FlatItem = {
		key: string;
		label: string;
		sublabel?: string;
		mono?: boolean;
		path: string; // unresolved path; pass through resolve() at use
		isRecent?: boolean;
		badge: string;
		badgeClass: string;
		section: string;
	};

	const PAGE_SUGGESTIONS: Array<{ keywords: string[]; label: string; path: string }> = [
		{ keywords: ['overview', 'dashboard', 'home', 'stats'], label: 'Overview', path: '/overview/' },
		{ keywords: ['block', 'blocks', 'height'], label: 'Blocks', path: '/blocks/' },
		{ keywords: ['transaction', 'transactions', 'tx', 'txs'], label: 'Transactions', path: '/txs/' },
		{
			keywords: ['transfer', 'transfers', 'usdc', 'eurc', 'usyc', 'token'],
			label: 'Transfers',
			path: '/transfers/'
		},
		{ keywords: ['trace', 'traces', 'internal'], label: 'Traces', path: '/traces/' },
		{
			keywords: ['crosschain', 'cross-chain', 'bridge', 'cctp', 'gateway'],
			label: 'Cross-chain',
			path: '/crosschain/'
		},
		{
			keywords: ['fx', 'stablefx', 'swap', 'eurc', 'exchange', 'rate'],
			label: 'StableFX',
			path: '/fx/'
		},
		{
			keywords: ['agent', 'agents', 'ai', 'erc-8004', 'erc8004', 'robot'],
			label: 'Agent registry',
			path: '/agents/'
		},
		{
			keywords: ['job', 'jobs', 'escrow', 'task', 'erc-8183', 'erc8183'],
			label: 'Job market',
			path: '/jobs/'
		},
		{
			keywords: ['graph', 'wallet graph', 'network', 'force'],
			label: 'Wallet graph',
			path: '/graph/'
		},
		{ keywords: ['token', 'tokens', 'erc20', 'erc-20', 'coin'], label: 'Tokens', path: '/tokens/' },
		{
			keywords: ['readme', 'about', 'help', 'docs', 'info'],
			label: 'About Arcadia',
			path: '/readme/'
		},
		{ keywords: ['debug', 'health', 'status'], label: 'Debug', path: '/debug/' }
	];

	function fuzzyScore(needle: string, hay: string): number {
		if (!needle) return 0;
		const n = needle.toLowerCase();
		const h = hay.toLowerCase();
		if (h === n) return 0;
		if (h.startsWith(n)) return 1;
		const idx = h.indexOf(n);
		if (idx >= 0) return 2 + idx;
		let hi = 0,
			gaps = 0,
			lastHit = -1;
		for (let i = 0; i < n.length; i++) {
			const ch = n[i];
			const found = h.indexOf(ch, hi);
			if (found < 0) return Infinity;
			if (lastHit >= 0) gaps += found - lastHit - 1;
			lastHit = found;
			hi = found + 1;
		}
		return 10 + gaps;
	}

	const pageSuggestions = $derived.by(() => {
		const q = searchQuery.trim().toLowerCase();
		if (!q) return [];
		return PAGE_SUGGESTIONS.map((p) => {
			const best = Math.min(fuzzyScore(q, p.label), ...p.keywords.map((k) => fuzzyScore(q, k)));
			return { p, score: best };
		})
			.filter((x) => isFinite(x.score))
			.sort((a, b) => a.score - b.score)
			.slice(0, 4)
			.map((x) => x.p);
	});

	const chainSuggestions = $derived.by(() => {
		const q = searchQuery.trim().toLowerCase();
		if (!q || q.length < 2) return [];
		const entries = Object.entries(DOMAIN_NAMES) as Array<[string, string]>;
		return entries
			.map(([id, name]) => ({ id: Number(id), name, score: fuzzyScore(q, name) }))
			.filter((x) => isFinite(x.score))
			.sort((a, b) => a.score - b.score)
			.slice(0, 4);
	});

	const items = $derived.by<FlatItem[]>(() => {
		const out: FlatItem[] = [];
		const d = search.data;
		if (d) {
			if (d.type === 'tx' && d.result) {
				out.push({
					key: 'tx',
					label: fmt.hash(d.result.hash as string),
					mono: true,
					path: `/tx/${d.result.hash as string}/`,
					badge: 'tx',
					badgeClass: 'info',
					section: 'Match'
				});
			} else if (d.type === 'block' && d.result) {
				out.push({
					key: 'block',
					label: `#${d.result.number}`,
					mono: true,
					path: `/blocks/${d.result.number as number}/`,
					badge: 'block',
					badgeClass: 'ok',
					section: 'Match'
				});
			} else if (d.type === 'wallet' && d.result) {
				out.push({
					key: 'wallet',
					label: fmt.addr(d.result.address as string),
					mono: true,
					path: `/wallet/${d.result.address as string}/`,
					badge: 'wallet',
					badgeClass: 'warn',
					section: 'Match'
				});
			} else if (d.type === 'agent' && d.result) {
				out.push({
					key: 'agent',
					label: fmt.addr(d.result.address as string),
					mono: true,
					path: `/wallet/${d.result.address as string}/`,
					badge: 'agent',
					badgeClass: 'acc',
					section: 'Match'
				});
			} else if (d.type === 'token' && d.result) {
				out.push({
					key: 'token',
					label: String(d.result.symbol || d.result.token_address || ''),
					sublabel: String(d.result.name || ''),
					path: `/tokens/${d.result.token_address as string}/`,
					badge: 'token',
					badgeClass: 'tok',
					section: 'Match'
				});
			} else if (d.type === 'multi') {
				(d.tokens || []).forEach((t) => {
					out.push({
						key: 'tok-' + t.id,
						label: t.symbol || '???',
						sublabel: t.name || '',
						path: `/tokens/${t.token_address}/`,
						badge: 'token',
						badgeClass: 'tok',
						section: 'Tokens'
					});
				});
				(d.agents || []).forEach((a) => {
					out.push({
						key: 'agt-' + a.id,
						label: fmt.addr(a.agent_address),
						mono: true,
						path: `/wallet/${a.agent_address}/`,
						badge: 'agent',
						badgeClass: 'acc',
						section: 'Agents'
					});
				});
			}
		}
		chainSuggestions.forEach((c) => {
			out.push({
				key: 'chain-' + c.id,
				label: c.name,
				sublabel: `domain ${c.id}`,
				path: `/crosschain/${c.id}/`,
				badge: 'chain',
				badgeClass: 'info',
				section: 'Chains'
			});
		});
		pageSuggestions.forEach((p) => {
			out.push({
				key: 'page-' + p.label,
				label: p.label,
				path: p.path,
				badge: 'page',
				badgeClass: 'dim',
				section: 'Pages'
			});
		});
		if (!searchQuery.trim() && recents.length > 0) {
			recents.forEach((r) => {
				out.push({
					key: 'recent-' + r,
					label: r,
					path: '',
					isRecent: true,
					badge: 'recent',
					badgeClass: 'dim',
					section: 'Recent'
				});
			});
		}
		return out;
	});

	const grouped = $derived.by(() => {
		const groups: Array<{ section: string; items: Array<FlatItem & { _idx: number }> }> = [];
		let i = 0;
		for (const it of items) {
			if (!groups.length || groups[groups.length - 1].section !== it.section) {
				groups.push({ section: it.section, items: [] });
			}
			groups[groups.length - 1].items.push({ ...it, _idx: i });
			i++;
		}
		return groups;
	});

	function loadRecents() {
		try {
			const raw = localStorage.getItem(RECENTS_KEY);
			recents = raw ? (JSON.parse(raw) as string[]).slice(0, RECENTS_MAX) : [];
		} catch {
			recents = [];
		}
	}

	function pushRecent(q: string) {
		if (!q) return;
		const next = [q, ...recents.filter((r) => r !== q)].slice(0, RECENTS_MAX);
		recents = next;
		try {
			localStorage.setItem(RECENTS_KEY, JSON.stringify(next));
		} catch {
			/* noop */
		}
	}

	function triggerSearch(q: string) {
		if (!q) return;
		runSearch(q);
	}

	function handleInput() {
		clearTimeout(debounceTimer);
		activeIndex = 0;
		const q = searchQuery.trim();
		if (!q) {
			clearSearch();
			return;
		}
		debounceTimer = setTimeout(() => triggerSearch(q), 300);
	}

	function activate(it: FlatItem) {
		if (it.isRecent) {
			searchQuery = it.label;
			handleInput();
			searchInput?.focus();
			return;
		}
		pushRecent(searchQuery.trim() || it.label);
		dismissSearch();
		goto(resolve(it.path as '/'));
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			if (items.length) activeIndex = (activeIndex + 1) % items.length;
			return;
		}
		if (e.key === 'ArrowUp') {
			e.preventDefault();
			if (items.length) activeIndex = (activeIndex - 1 + items.length) % items.length;
			return;
		}
		if (e.key === 'Enter') {
			clearTimeout(debounceTimer);
			if (items.length && items[activeIndex]) {
				e.preventDefault();
				activate(items[activeIndex]);
				return;
			}
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
		loadRecents();
		function globalShortcut(e: KeyboardEvent) {
			if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
				e.preventDefault();
				focus();
			}
			if (
				e.key === '/' &&
				document.activeElement?.tagName !== 'INPUT' &&
				document.activeElement?.tagName !== 'TEXTAREA'
			) {
				e.preventDefault();
				focus();
			}
		}
		window.addEventListener('keydown', globalShortcut);
		return () => window.removeEventListener('keydown', globalShortcut);
	});

	const showDropdown = $derived(
		searchFocused && (search.loading || !!search.error || items.length > 0 || !!searchQuery.trim())
	);
</script>

<div class="search" class:open={searchFocused || items.length > 0}>
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
		placeholder="Search tokens, agents, chains, txs, blocks…"
		onfocus={() => (searchFocused = true)}
		onblur={() => setTimeout(() => (searchFocused = false), 200)}
		oninput={handleInput}
		onkeydown={handleKeydown}
		aria-autocomplete="list"
		aria-expanded={showDropdown}
	/>
	{#if searchQuery}
		<button
			class="search-clear"
			onclick={() => {
				searchQuery = '';
				clearSearch();
				searchInput?.focus();
			}}
			aria-label="Clear search">✕</button
		>
	{:else}
		<span class="kbd" aria-hidden="true">⌘K</span>
	{/if}

	{#if showDropdown}
		<div
			class="search-results"
			role="listbox"
			transition:slide={{ duration: 160, easing: cubicOut }}
		>
			{#if search.loading && items.length === 0}
				<div class="search-result-item muted" transition:fade={{ duration: 120 }}>searching…</div>
			{:else if search.error}
				<div class="search-result-item err-text" transition:fade={{ duration: 120 }}>
					{search.error}
				</div>
			{/if}

			{#if !search.loading && items.length === 0 && searchQuery.trim()}
				<div class="search-result-item muted" transition:fade={{ duration: 120 }}>
					no matches — try a different query
				</div>
			{/if}

			<div class="results-list" class:loading={search.loading}>
				{#each grouped as g, gi (g.section)}
					<div
						class="search-group"
						animate:flip={{ duration: 220, easing: cubicOut }}
						in:fade={{ duration: 140 }}
						out:fade={{ duration: 80 }}
					>
						{#if gi > 0}
							<div class="search-divider"></div>
						{/if}
						<div class="search-section-label">{g.section}</div>
						{#each g.items as it (it.key)}
							<div
								class="result-row"
								animate:flip={{ duration: 220, easing: cubicOut }}
								in:fade={{ duration: 140 }}
								out:fade={{ duration: 80 }}
							>
								{#if it.isRecent}
									<button
										type="button"
										class="search-result-item"
										class:active={it._idx === activeIndex}
										onmousemove={() => (activeIndex = it._idx)}
										onmousedown={(e) => {
											e.preventDefault();
											activate(it);
										}}
									>
										<span class="badge {it.badgeClass}">{it.badge}</span>
										<span class:mono={it.mono}>{it.label}</span>
									</button>
								{:else}
									<a
										class="search-result-item"
										class:active={it._idx === activeIndex}
										href={resolve(it.path as '/')}
										onmousemove={() => (activeIndex = it._idx)}
										onclick={() => activate(it)}
									>
										<span class="badge {it.badgeClass}">{it.badge}</span>
										<span class:mono={it.mono}>{it.label}</span>
										{#if it.sublabel}
											<span class="dim">{it.sublabel}</span>
										{/if}
									</a>
								{/if}
							</div>
						{/each}
					</div>
				{/each}
			</div>
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
		opacity: 0.55;
		font-size: 11px;
		margin-left: 6px;
	}
	.search-result-item {
		position: relative;
		width: 100%;
		text-align: left;
		font: inherit;
		background: none;
		border: none;
	}
	.search-result-item:focus {
		outline: none;
	}
	.search-result-item.active {
		background: var(--bg-hover);
	}
	.search-result-item.active::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0;
		bottom: 0;
		width: 2px;
		background: var(--acc, #64dca0);
	}
	.results-list {
		transition: opacity 160ms ease;
	}
	.results-list.loading {
		opacity: 0.55;
	}
	.result-row,
	.search-group {
		will-change: transform;
	}
</style>
