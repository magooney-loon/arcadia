<script lang="ts">
	import { resolve } from '$app/paths';

	const GITHUB = 'https://github.com/magooney-loon/arcadia';

	const sections = [
		{
			href: resolve('/overview/'),
			label: 'Overview',
			desc: 'Live dashboard with TPS, transfer volume, fee analytics, bridge net flow, top agents, and cross-chain pulse — all with switchable 1h / 24h / 7d windows.'
		},
		{
			href: resolve('/blocks/'),
			label: 'Blocks',
			desc: 'Every block on Arc with timestamp, transaction count, gas utilization, and fees. Click into any block for full details.'
		},
		{
			href: resolve('/txs/'),
			label: 'Transactions',
			desc: 'Real-time transaction feed with status, method signature, sender/receiver, and fee. Filter and sort by any column.'
		},
		{
			href: resolve('/transfers/'),
			label: 'Transfers',
			desc: 'Every USDC, EURC, and USYC token transfer on the network. Filter by token, spot whale transfers ($10K+), and trace capital movements.'
		},
		{
			href: resolve('/traces/'),
			label: 'Traces',
			desc: 'Internal transaction traces — see what really happens inside complex transactions: contract calls, delegate calls, and value transfers.'
		},
		{
			href: resolve('/crosschain/'),
			label: 'Cross-chain',
			desc: 'CCTP and Gateway bridge events between Arc and Ethereum, Base, Solana, and more. Track inbound/outbound flows by chain and protocol.'
		},
		{
			href: resolve('/fx/'),
			label: 'StableFX',
			desc: 'Onchain USDC↔EURC swap trades with implied exchange rate, maker/taker, and full lifecycle tracking from creation to settlement.'
		},
		{
			href: resolve('/agents/'),
			label: 'Agent registry',
			desc: 'All registered AI agents (ERC-8004) with transaction count, volume, fees, and job statistics. See who the most active agents are.'
		},
		{
			href: resolve('/jobs/'),
			label: 'Job market',
			desc: 'AI agent job postings and settlements (ERC-8183). Track escrow amounts, job lifecycle, and agent-to-agent service economy.'
		},
		{
			href: resolve('/graph/'),
			label: 'Wallet graph',
			desc: 'Interactive force-directed graph of wallet-to-wallet capital flows. Visualize who transacts with whom and how value moves through the network.'
		}
	];

	const dataPoints = [
		{ label: 'Blocks', detail: 'height, timestamp, tx count, gas used, utilization %, fees' },
		{ label: 'Transactions', detail: 'hash, status, from/to, value, gas price, method (sighash)' },
		{ label: 'Token transfers', detail: 'USDC, EURC, USYC — amount, sender, receiver, tx hash' },
		{ label: 'Internal traces', detail: 'call type, from/to, value, gas used, depth' },
		{
			label: 'Bridge events',
			detail: 'CCTP + Gateway, source/destination chain, direction, amount'
		},
		{ label: 'StableFX trades', detail: 'USDC/EURC pair, rate, maker/taker, status, escrow' },
		{ label: 'Agent profiles', detail: 'ERC-8004 registration, metadata, tx volume, job stats' },
		{ label: 'Job lifecycle', detail: 'ERC-8183 jobs — escrow, agent, status, settlement' },
		{ label: 'Wallet history', detail: 'per-address send/receive, net flow by token, graph edges' },
		{
			label: 'Analytics snapshots',
			detail: '5-min resolution: TPS, fees (p25–p95), volume, active addresses'
		}
	];

	const useCases = [
		{
			icon: '📈',
			label: 'Monitor capital flows',
			desc: 'Track net capital entering or leaving Arc via CCTP and Gateway bridges across all connected chains. The cross-chain page breaks down inbound vs outbound volume per chain in real time — useful for understanding liquidity trends and market sentiment.'
		},
		{
			icon: '🐋',
			label: 'Track whale movements',
			desc: 'Transfers of $10K or more are flagged throughout the app. Sort the transfers table by amount, or check the overview page for the largest transfer in any window. Navigate to any wallet to see its full send/receive history and who it interacts with.'
		},
		{
			icon: '🤖',
			label: 'Watch the AI agent economy',
			desc: 'Arc has native onchain AI agent identity (ERC-8004) and a job escrow system (ERC-8183). Browse the agent leaderboard by volume, check the job market for active and settled jobs, and see which agents are earning the most.'
		},
		{
			icon: '💱',
			label: 'Analyze stablecoin FX',
			desc: 'StableFX settles USDC↔EURC swaps onchain. Every trade is captured with implied exchange rate, maker/taker, and lifecycle status. Useful for monitoring FX basis risk, stablecoin peg health, and onchain DeFi activity.'
		},
		{
			icon: '🔍',
			label: 'Investigate any address',
			desc: 'Search for any wallet address, transaction hash, or block number using the search bar (⌘K). The wallet detail page shows full transfer history, net flow by token, graph connections, and agent status if the address is registered.'
		},
		{
			icon: '📊',
			label: 'Build quant models',
			desc: 'All analytics are pre-aggregated into 5-minute snapshots across 1h/24h/7d windows — transfer volume, fee percentiles (p25/p50/p75/p95), block time, and active address counts. Use the REST API to pull time-series data for volatility analysis or regime detection.'
		},
		{
			icon: '🕸️',
			label: 'Map wallet networks',
			desc: 'The interactive wallet graph renders capital flow edges between addresses. Filter by a specific wallet to see its direct connections and volume, or explore the full network to identify clusters and key actors.'
		},
		{
			icon: '⚡',
			label: 'Feed trading agents',
			desc: 'Use the REST API endpoints for live bridge flow, transfer volume, and analytics snapshots as data feeds for automated trading or agent decision loops. The OpenAPI docs describe every available endpoint.'
		}
	];
</script>

<svelte:head>
	<title>About · Arcadia Explorer</title>
</svelte:head>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">About Arcadia</div>
			<div class="view-sub">What this app tracks and how to use it</div>
		</div>
	</div>

	<div class="readme-body">
		<!-- Intro -->
		<section class="card">
			<div class="section-head">What is Arcadia Explorer?</div>
			<p class="prose">
				Arcadia is a <strong>real-time blockchain explorer and analytics dashboard</strong> for the Arc
				L1 testnet (chain ID 5042002). It indexes every block, transaction, token transfer, bridge event,
				FX trade, AI agent registration, and job settlement — then presents it through interactive pages
				you can search, sort, and filter.
			</p>
			<p class="prose">
				Think of it as a combination block explorer, analytics platform, and onchain data API
				purpose-built for the Arc ecosystem. Everything updates live — blocks stream in every few
				seconds, analytics refresh every 10 seconds, and you can drill into any address,
				transaction, or block instantly.
			</p>
		</section>

		<!-- Notice -->
		<section class="card notice-card">
			<div class="notice-icon">
				<svg
					viewBox="0 0 20 20"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
					style="width:20px;height:20px"
				>
					<circle cx="10" cy="10" r="8" />
					<path d="M10 6 V10.5 M10 13.5 V14" stroke-linecap="round" />
				</svg>
			</div>
			<div>
				<div class="notice-title">Demo instance</div>
				<div class="notice-body">
					This explorer runs on Envio's free HyperSync tier. Data is rate-limited and the free API
					can lag behind chain tip. For production use or real-time data, self-host with your own
					Envio API key.
				</div>
			</div>
		</section>

		<!-- What you can explore -->
		<section class="card">
			<div class="section-head">Pages</div>
			<div class="page-list">
				{#each sections as s (s.label)}
					<a href={s.href} class="page-item">
						<div class="page-label">{s.label}</div>
						<div class="page-desc">{s.desc}</div>
					</a>
				{/each}
			</div>
		</section>

		<!-- Data tracked -->
		<section class="card">
			<div class="section-head">Data coverage</div>
			<p class="prose">
				Arcadia indexes <strong>every onchain event</strong> on the Arc testnet and pre-aggregates analytics
				into 5-minute snapshots. Here's what's available:
			</p>
			<div class="data-grid">
				{#each dataPoints as d (d.label)}
					<div class="data-row">
						<div class="data-label">{d.label}</div>
						<div class="data-detail">{d.detail}</div>
					</div>
				{/each}
			</div>
		</section>

		<!-- Use cases -->
		<section class="card">
			<div class="section-head">Use cases</div>
			<div class="use-cases">
				{#each useCases as uc (uc.label)}
					<div class="use-case">
						<div class="uc-head">
							<span class="uc-icon">{uc.icon}</span>
							<span class="uc-label">{uc.label}</span>
						</div>
						<div class="uc-desc">{uc.desc}</div>
					</div>
				{/each}
			</div>
		</section>

		<!-- Source code -->
		<section class="card">
			<div class="section-head">Source code</div>
			<div class="github-row">
				<svg viewBox="0 0 20 20" fill="currentColor" style="width:18px;height:18px;flex-shrink:0">
					<path
						fill-rule="evenodd"
						d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 10 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0 0 20 10.017C20 4.484 15.522 0 10 0Z"
						clip-rule="evenodd"
					/>
				</svg>
				<a href={GITHUB} target="_blank" rel="external noopener noreferrer" class="github-link"
					>{GITHUB}</a
				>
			</div>
		</section>
	</div>
</div>

<style>
	.card {
		padding: 16px 18px;
	}

	.readme-body {
		display: flex;
		flex-direction: column;
		gap: 12px;
		max-width: 760px;
	}

	/* Notice */
	.notice-card {
		display: flex;
		gap: 14px;
		align-items: flex-start;
		border-color: var(--accent);
		background: color-mix(in srgb, var(--accent) 6%, var(--bg-2));
	}

	.notice-icon {
		color: var(--accent);
		flex-shrink: 0;
		margin-top: 2px;
	}

	.notice-title {
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--accent);
		margin-bottom: 6px;
	}

	.notice-body {
		font-size: 13px;
		line-height: 1.6;
		color: var(--fg-2);
	}

	/* Sections */
	.section-head {
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.07em;
		color: var(--fg-3);
		margin-bottom: 12px;
	}

	.prose {
		font-size: 13px;
		line-height: 1.7;
		color: var(--fg-2);
		margin: 0 0 10px 0;
	}

	.prose strong {
		color: var(--fg-1);
	}

	.prose:last-child {
		margin-bottom: 0;
	}

	/* Pages list */
	.page-list {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.page-item {
		display: block;
		padding: 10px 12px;
		border-radius: 6px;
		text-decoration: none;
		transition: background 0.15s;
	}

	.page-item:hover {
		background: var(--bg-3);
	}

	.page-label {
		font-size: 13px;
		font-weight: 600;
		color: var(--accent);
		margin-bottom: 2px;
	}

	.page-desc {
		font-size: 12.5px;
		line-height: 1.55;
		color: var(--fg-3);
	}

	.page-item:hover .page-desc {
		color: var(--fg-2);
	}

	/* Data grid */
	.data-grid {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.data-row {
		display: flex;
		gap: 12px;
		align-items: baseline;
	}

	.data-label {
		font-size: 12px;
		font-weight: 600;
		color: var(--fg-1);
		white-space: nowrap;
		min-width: 120px;
	}

	.data-detail {
		font-size: 12.5px;
		line-height: 1.55;
		color: var(--fg-3);
	}

	/* Use cases */
	.use-cases {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.use-case {
		display: flex;
		flex-direction: column;
		gap: 5px;
	}

	.uc-head {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.uc-icon {
		font-size: 14px;
		flex-shrink: 0;
	}

	.uc-label {
		font-size: 13px;
		font-weight: 600;
		color: var(--fg-1);
	}

	.uc-desc {
		font-size: 12.5px;
		line-height: 1.65;
		color: var(--fg-2);
		padding-left: 22px;
	}

	/* GitHub */
	.github-row {
		display: flex;
		align-items: center;
		gap: 10px;
		color: var(--fg-2);
	}

	.github-link {
		font-size: 13px;
		color: var(--accent);
		word-break: break-all;
	}
</style>
