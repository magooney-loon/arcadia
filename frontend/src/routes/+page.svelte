<script lang="ts">
	import { resolve } from '$app/paths';

	const REPO_URL = 'https://github.com/magooney-loon/arcadia';
</script>

<svelte:head>
	<title>Arcadia · Explorer</title>
</svelte:head>

<div class="landing">
	<div class="card">
		<div class="logo-row">
			<div class="logo-mark">
				<svg viewBox="0 0 32 32" fill="none">
					<path d="M16 2A14 14 0 0 0 4.1 22.5L7.8 21A10 10 0 0 1 16 6V2Z" fill="var(--accent)" />
					<path
						d="M4.1 22.5A14 14 0 0 0 28 20L24.8 17.5A10 10 0 0 1 7.8 21L4.1 22.5Z"
						fill="var(--accent)"
						opacity="0.5"
					/>
					<circle cx="16" cy="16" r="3" fill="var(--accent)" />
				</svg>
			</div>
			<div class="brand">
				<div class="brand-name">ARCADIA<span class="net">·explorer</span></div>
				<div class="brand-sub">Arc L1 indexer · real-time analytics</div>
			</div>
		</div>

		<div class="badge-row">
			<span class="badge">demo showcase</span>
			<span class="badge dim">arc testnet</span>
			<span class="badge dim">v0.4.2-rc1</span>
		</div>

		<p class="lead">
			A real-time blockchain indexer and analytics platform for
			<a href="https://docs.arc.network/" target="_blank" rel="external noopener noreferrer">Arc L1</a>.
			Indexes blocks, transactions, stablecoin transfers, internal traces, AI agent lifecycle events,
			cross-chain flows, FX swaps, and derived economic metrics — then serves them through a
			versioned REST API and this live SvelteKit dashboard.
		</p>

		<div class="callout">
			<div class="callout-label">heads up</div>
			<div class="callout-body">
				This is a public demo running on the Envio HyperSync free tier and a single SQLite writer.
				It's here to show what's possible — not to serve production traffic. For your own data,
				fork the repo, plug in your Envio API key, and run it locally or self-host. Extend the Go
				indexer to capture whatever events your protocol emits.
			</div>
		</div>

		<div class="features">
			<div class="feature">
				<div class="feature-title">Index everything</div>
				<div class="feature-body">
					Blocks, txs, ERC-20/721/1155 transfers, traces, ERC-8004 agents, ERC-8183 jobs, CCTP /
					Gateway bridges, StableFX swaps.
				</div>
			</div>
			<div class="feature">
				<div class="feature-title">Live dashboard</div>
				<div class="feature-body">
					SSE-backed Svelte 5 frontend. Charts, wallet graph, agent leaderboard, snapshot-backed
					analytics at 1h / 24h / 7d.
				</div>
			</div>
			<div class="feature">
				<div class="feature-title">Forkable</div>
				<div class="feature-body">
					Go + PocketBase + SQLite. One binary, one DB file, one config. Add a topic, add a
					handler, you're done.
				</div>
			</div>
		</div>

		<div class="actions">
			<a class="btn primary" href={resolve('/overview/')}>
				Enter explorer
				<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.6">
					<path d="M3 7 H11 M8 4 L11 7 L8 10" stroke-linecap="round" stroke-linejoin="round" />
				</svg>
			</a>
			<a class="btn ghost" href={REPO_URL} target="_blank" rel="external noopener noreferrer">
				<svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
					<path
						d="M8 0C3.58 0 0 3.58 0 8a8 8 0 0 0 5.47 7.59c.4.07.55-.17.55-.38v-1.3c-2.23.48-2.7-1.07-2.7-1.07-.36-.93-.89-1.18-.89-1.18-.73-.5.06-.49.06-.49.81.06 1.23.83 1.23.83.72 1.22 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.65-.89-3.65-3.97 0-.88.31-1.59.83-2.16-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.83a7.7 7.7 0 0 1 4 0c1.53-1.04 2.2-.83 2.2-.83.44 1.1.16 1.92.08 2.12.52.57.83 1.28.83 2.16 0 3.09-1.87 3.77-3.66 3.97.29.25.54.73.54 1.48v2.2c0 .21.15.46.55.38A8 8 0 0 0 16 8c0-4.42-3.58-8-8-8z"
					/>
				</svg>
				View on GitHub
			</a>
			<a class="btn ghost" href={resolve('/readme/')}>README</a>
		</div>

		<div class="footnote">
			<span class="dim">stack ·</span> Go · PocketBase · SvelteKit · HyperSync · SQLite (WAL)
		</div>
	</div>
</div>

<style>
	.landing {
		min-height: 100dvh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 32px 20px;
		background:
			radial-gradient(circle at 20% 20%, rgba(80, 150, 255, 0.06), transparent 50%),
			radial-gradient(circle at 80% 80%, rgba(60, 200, 180, 0.05), transparent 50%),
			var(--bg-0);
	}

	.card {
		max-width: 720px;
		width: 100%;
		background: var(--bg-1);
		border: 1px solid var(--border-1);
		border-radius: 10px;
		padding: 40px 38px 32px;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
	}

	.logo-row {
		display: flex;
		align-items: center;
		gap: 14px;
		margin-bottom: 14px;
	}
	.logo-mark {
		width: 40px;
		height: 40px;
		flex-shrink: 0;
	}
	.logo-mark svg {
		width: 100%;
		height: 100%;
	}
	.brand-name {
		font-size: 18px;
		font-weight: 700;
		letter-spacing: 1px;
		color: var(--fg-0);
	}
	.brand-name .net {
		color: var(--accent);
		font-weight: 500;
		letter-spacing: 0;
		margin-left: 2px;
	}
	.brand-sub {
		font-size: 11px;
		color: var(--fg-3);
		margin-top: 2px;
	}

	.badge-row {
		display: flex;
		gap: 6px;
		margin-bottom: 20px;
		flex-wrap: wrap;
	}
	.badge {
		font-size: 9px;
		text-transform: uppercase;
		letter-spacing: 0.6px;
		padding: 3px 8px;
		border-radius: 3px;
		background: rgba(120, 200, 255, 0.12);
		color: var(--accent);
		border: 1px solid rgba(120, 200, 255, 0.25);
	}
	.badge.dim {
		background: var(--bg-2);
		color: var(--fg-3);
		border-color: var(--border-1);
	}

	.lead {
		font-size: 13px;
		line-height: 1.65;
		color: var(--fg-3);
		margin: 0 0 18px;
	}
	.lead a {
		color: var(--accent);
		text-decoration: none;
		border-bottom: 1px dashed currentColor;
	}
	.lead a:hover {
		opacity: 0.8;
	}

	.callout {
		background: var(--bg-2);
		border-left: 2px solid var(--warn);
		padding: 12px 14px;
		border-radius: 4px;
		margin-bottom: 22px;
	}
	.callout-label {
		font-size: 9px;
		text-transform: uppercase;
		letter-spacing: 0.6px;
		color: var(--warn);
		margin-bottom: 4px;
		font-weight: 600;
	}
	.callout-body {
		font-size: 12px;
		line-height: 1.6;
		color: var(--fg-3);
	}

	.features {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 10px;
		margin-bottom: 24px;
	}
	@media (max-width: 600px) {
		.features {
			grid-template-columns: 1fr;
		}
	}
	.feature {
		background: var(--bg-2);
		border: 1px solid var(--border-1);
		border-radius: 5px;
		padding: 12px 14px;
	}
	.feature-title {
		font-size: 11px;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		color: var(--accent);
		margin-bottom: 6px;
		font-weight: 600;
	}
	.feature-body {
		font-size: 11px;
		line-height: 1.5;
		color: var(--fg-3);
	}

	.actions {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
		align-items: center;
		margin-bottom: 20px;
	}
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 9px 14px;
		border-radius: 5px;
		font-size: 12px;
		font-weight: 500;
		letter-spacing: 0.2px;
		text-decoration: none;
		border: 1px solid transparent;
		transition: background 120ms ease;
	}
	.btn svg {
		width: 12px;
		height: 12px;
		flex-shrink: 0;
	}
	.btn.primary {
		background: var(--accent);
		color: var(--bg-0);
	}
	.btn.primary:hover {
		filter: brightness(1.1);
	}
	.btn.ghost {
		background: var(--bg-2);
		color: var(--fg-0);
		border-color: var(--border-1);
	}
	.btn.ghost:hover {
		background: var(--border-1);
	}

	.footnote {
		font-size: 10px;
		color: var(--fg-3);
		opacity: 0.7;
		padding-top: 14px;
		border-top: 1px solid var(--border-1);
	}
	.dim {
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}
</style>
