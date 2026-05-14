<script lang="ts">
	const GITHUB = 'https://github.com/magooney-loon/arcadia';
	const ENVIO = 'https://envio.dev';
</script>

<svelte:head>
	<title>README · Arcadia Explorer</title>
</svelte:head>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">README</div>
			<div class="view-sub">Demo · self-hosting · use cases</div>
		</div>
	</div>

	<div class="readme-body">
		<section class="card notice-card">
			<div class="notice-icon">
				<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5" style="width:20px;height:20px">
					<circle cx="10" cy="10" r="8" />
					<path d="M10 6 V10.5 M10 13.5 V14" stroke-linecap="round" />
				</svg>
			</div>
			<div>
				<div class="notice-title">Demo instance · free tier</div>
				<div class="notice-body">
					This explorer runs on the <strong>Envio HyperSync free API tier</strong>. Throughput is
					rate-limited and the Arc testnet HyperSync endpoint lags ~55 minutes behind chain tip by
					design (Envio's ingestion delay on this network). For production use or real-time data,
					self-host with your own API key.
				</div>
			</div>
		</section>

		<section class="card">
			<div class="section-head">What is Arcadia?</div>
			<p class="prose">
				Arcadia is a full-stack, real-time blockchain indexer and analytics dashboard for the
				<strong>Arc L1 testnet</strong> (chain ID 5042002). It streams every layer of onchain
				activity — blocks, transactions, USDC/EURC/USYC token transfers, internal traces, AI agent
				registrations (ERC-8004), job settlements (ERC-8183), cross-chain CCTP/Gateway flows, and
				StableFX swaps — into a local PocketBase database and exposes it through a REST API and live
				SvelteKit frontend.
			</p>
			<p class="prose">
				All data is pre-aggregated into 5-minute snapshots across 1h / 24h / 7d windows so
				dashboards are instant reads rather than live SQL aggregations.
			</p>
		</section>

		<section class="card">
			<div class="section-head">Use cases</div>
			<div class="use-cases">
				<div class="use-case">
					<div class="uc-label">Trading agents</div>
					<div class="uc-desc">
						Feed live inflow/outflow data and bridge flow direction into agent decision loops.
						Monitor net capital entering Arc via CCTP from Ethereum/Base/Solana and adjust
						positions ahead of liquidity movements. The <code>/analytics/bridge_flow</code> endpoint
						gives per-chain directional volume in real time.
					</div>
				</div>
				<div class="use-case">
					<div class="uc-label">Quant analytics</div>
					<div class="uc-desc">
						The <code>analytics_snapshots</code> collection stores a complete time-series of
						transfer volume, fee percentiles (p25/p50/p75/p95), block time, and active address
						counts at 5-minute resolution. Use the <code>/analytics/history</code> endpoint to pull
						rolling windows for volatility, autocorrelation, or regime-detection models.
					</div>
				</div>
				<div class="use-case">
					<div class="uc-label">Whale tracking · copy trading</div>
					<div class="uc-desc">
						Whale transfers ($10K+) are tracked in the <code>transfers</code> collection and
						flagged in snapshots. The <code>/wallet/{"{address}"}</code> endpoint returns full
						send/receive history and graph edges per address. Combine with the wallet graph to map
						capital flows between large wallets and identify lead actors to follow.
					</div>
				</div>
				<div class="use-case">
					<div class="uc-label">Agent economy monitoring</div>
					<div class="uc-desc">
						Arc has native onchain AI agent identity (ERC-8004) and a job escrow system
						(ERC-8183). Arcadia indexes every agent registration, job lifecycle event, and
						agent-to-agent capital flow. Use the agent leaderboard and job market to monitor the
						health and growth rate of the AI agent economy on Arc.
					</div>
				</div>
				<div class="use-case">
					<div class="uc-label">FX and stablecoin research</div>
					<div class="uc-desc">
						StableFX settles USDC↔EURC swaps onchain. Every trade is indexed with implied rate,
						maker/taker, and settlement status. Cross-chain USDC mint/burn events give a full
						picture of stablecoin supply dynamics. Useful for FX basis research and stablecoin
						peg health monitoring.
					</div>
				</div>
			</div>
		</section>

		<section class="card">
			<div class="section-head">Self-hosting</div>
			<p class="prose">Run your own instance with an Envio API key for full throughput and no lag.</p>
			<div class="steps">
				<div class="step">
					<div class="step-n">1</div>
					<div>
						Get a free API key at
						<a href={ENVIO} target="_blank" rel="external noopener noreferrer">{ENVIO}</a>
					</div>
				</div>
				<div class="step">
					<div class="step-n">2</div>
					<div>
						Clone the repo and install the toolchain:
						<pre><code>git clone {GITHUB}.git
cd arcadia
go install github.com/magooney-loon/pb-ext/cmd/pb-cli@latest</code></pre>
					</div>
				</div>
				<div class="step">
					<div class="step-n">3</div>
					<div>
						Set your API token and run:
						<pre><code>export ENVIO_API_TOKEN=your_token_here
go run ./cmd/server --dev</code></pre>
					</div>
				</div>
				<div class="step">
					<div class="step-n">4</div>
					<div>
						Open <code>http://127.0.0.1:8090</code> — the frontend and API are served from the same
						process. Admin UI at <code>/_/</code>.
					</div>
				</div>
			</div>
		</section>

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

	.notice-body strong {
		color: var(--fg-1);
	}

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

	.use-cases {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.use-case {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.uc-label {
		font-size: 12px;
		font-weight: 600;
		color: var(--fg-1);
	}

	.uc-desc {
		font-size: 13px;
		line-height: 1.65;
		color: var(--fg-2);
	}

	.uc-desc code {
		font-family: var(--font-mono, monospace);
		font-size: 11px;
		background: var(--bg-3);
		padding: 1px 5px;
		border-radius: 3px;
		color: var(--accent);
	}

	.steps {
		display: flex;
		flex-direction: column;
		gap: 14px;
		margin-top: 4px;
	}

	.step {
		display: flex;
		gap: 14px;
		align-items: flex-start;
		font-size: 13px;
		color: var(--fg-2);
		line-height: 1.6;
	}

	.step-n {
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: var(--bg-3);
		border: 1px solid var(--border-1);
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 11px;
		font-weight: 600;
		color: var(--accent);
		flex-shrink: 0;
		margin-top: 1px;
	}

	.step pre {
		margin: 8px 0 0 0;
		background: var(--bg-3);
		border: 1px solid var(--border-1);
		border-radius: 4px;
		padding: 10px 12px;
		overflow-x: auto;
	}

	.step code {
		font-family: var(--font-mono, monospace);
		font-size: 12px;
		color: var(--fg-1);
	}

	.step a {
		color: var(--accent);
	}

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
