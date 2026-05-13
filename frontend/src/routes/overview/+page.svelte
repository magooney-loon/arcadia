<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { stats, fetchStats } from '$lib/stores/stats.svelte';
	import { blocks, transactions, fetchBlocks, fetchTransactions } from '$lib/stores/chain.svelte';
	import { fx, fetchFx } from '$lib/stores/fx.svelte';
	import {
		analyticsFees,
		fetchAnalyticsFees,
		analyticsVolume,
		fetchAnalyticsVolume,
		analyticsBridgeFlow,
		fetchAnalyticsBridgeFlow,
		analyticsAgentLeaderboard,
		fetchAgentLeaderboard
	} from '$lib/stores/analytics.svelte';
	import * as fmt from '$lib/fmt.js';

	onMount(() => {
		const refresh = () => {
			fetchStats();
			fetchBlocks(10);
			fetchTransactions({ limit: 10 });
		};
		refresh();
		fetchFx({ limit: 5 });
		fetchAgentLeaderboard(5);
		fetchAnalyticsFees();
		fetchAnalyticsVolume();
		fetchAnalyticsBridgeFlow();
		const id = setInterval(refresh, 6000);
		return () => clearInterval(id);
	});

	const latestBlock = $derived(stats.data?.block_number ?? 0);
	const bridgeFlow = $derived(analyticsBridgeFlow.data);
	const chainEntries = $derived(
		Object.entries(bridgeFlow?.by_chain ?? {})
			.sort((a, b) => b[1].inbound_vol + b[1].outbound_vol - (a[1].inbound_vol + a[1].outbound_vol))
			.slice(0, 5)
	);
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Overview</div>
			<div class="view-sub">Live chain state · arc testnet</div>
		</div>
	</div>

	<!-- Stats row -->
	<div class="grid grid-stats">
		<div class="stat">
			<div class="label">TPS</div>
			<div class="value">{fmt.tps(stats.data?.tps)}<span class="unit">tx/s</span></div>
		</div>
		<div class="stat">
			<div class="label">Block time</div>
			<div class="value">
				{stats.data?.block_time_ms ? Math.round(stats.data.block_time_ms) : '—'}<span class="unit"
					>ms</span
				>
			</div>
		</div>
		<div class="stat">
			<div class="label">Transfers 24h</div>
			<div class="value">{fmt.num(analyticsVolume.data?.total_transfers)}</div>
		</div>
		<div class="stat">
			<div class="label">Fees 24h</div>
			<div class="value">{fmt.usdc(analyticsFees.data?.total_fees)}</div>
		</div>
		<div class="stat">
			<div class="label">Bridge net flow</div>
			<div class="value {(bridgeFlow?.net_flow ?? 0) >= 0 ? '' : 'err'}">
				{fmt.usdc(bridgeFlow?.net_flow)}
			</div>
			<div class="delta up">24h inbound − outbound</div>
		</div>
		<div class="stat">
			<div class="label">Agents</div>
			<div class="value">{analyticsAgentLeaderboard.data?.count ?? '—'}</div>
		</div>
	</div>

	<!-- Throughput + Cross-chain pulse -->
	<div class="grid grid-2" style="margin-top:12px">
		<div class="card">
			<div class="card-head">
				<div class="card-title">Fee analytics · 24h</div>
				<div class="card-sub">block_stats · percentiles</div>
			</div>
			<div class="card-body">
				{#if analyticsFees.data && analyticsFees.data.block_count > 0}
					<div class="detail-grid">
						<dt>Blocks sampled</dt>
						<dd>{fmt.num(analyticsFees.data.block_count)}</dd>
						<dt>Total fees</dt>
						<dd>{fmt.usdc(analyticsFees.data.total_fees, 4)}</dd>
						<dt>Fee p25</dt>
						<dd>{fmt.usdc(analyticsFees.data.avg_fee_p25, 6)}</dd>
						<dt>Fee p50</dt>
						<dd>{fmt.usdc(analyticsFees.data.avg_fee_p50, 6)}</dd>
						<dt>Fee p75</dt>
						<dd>{fmt.usdc(analyticsFees.data.avg_fee_p75, 6)}</dd>
						<dt>Fee p95</dt>
						<dd>{fmt.usdc(analyticsFees.data.avg_fee_p95, 6)}</dd>
						<dt>Avg block time</dt>
						<dd>{fmt.ms(analyticsFees.data.avg_block_time_ms)}</dd>
						<dt>Failed tx ratio</dt>
						<dd>{(analyticsFees.data.failed_tx_ratio * 100).toFixed(2)}%</dd>
					</div>
				{:else}
					<p class="mono muted" style="font-size:11px">loading…</p>
				{/if}
			</div>
		</div>

		<div class="card">
			<div class="card-head">
				<div class="card-title">Cross-chain pulse</div>
				<div class="card-sub">CCTP · 24h by chain</div>
			</div>
			<div class="card-body" style="padding:0">
				{#if bridgeFlow}
					<div class="flow" style="border-top:0;background:var(--bg-2);padding:8px 14px">
						<span class="mono dim" style="font-size:10px">in</span>
						<span class="mono fg0" style="margin-left:4px"
							>{fmt.usdc(bridgeFlow.inbound_vol)} ({bridgeFlow.inbound_count})</span
						>
						<span class="spacer"></span>
						<span class="mono dim" style="font-size:10px">out</span>
						<span class="mono" style="margin-left:4px"
							>{fmt.usdc(bridgeFlow.outbound_vol)} ({bridgeFlow.outbound_count})</span
						>
					</div>
					{#each chainEntries as [chain, flow] (chain)}
						<div class="flow">
							<span class="chain">{chain}</span>
							<span class="arrow">↔</span>
							<span class="mono" style="font-size:11px;color:var(--ok)"
								>↘ {fmt.usdc(flow.inbound_vol)}</span
							>
							<span class="spacer"></span>
							<span class="mono" style="font-size:11px;color:var(--err)"
								>↗ {fmt.usdc(flow.outbound_vol)}</span
							>
						</div>
					{/each}
				{:else}
					<p class="mono muted" style="padding:14px;font-size:11px">loading…</p>
				{/if}
			</div>
		</div>
	</div>

	<!-- Latest blocks + latest txs -->
	<div class="grid grid-2" style="margin-top:12px">
		<div class="card">
			<div class="card-head">
				<span class="dot acc"></span>
				<div class="card-title">Latest blocks</div>
				<div class="card-sub">live</div>
				<div class="card-actions">
					<a class="mono dim" style="font-size:10px" href={resolve('/blocks/')}>SEE ALL →</a>
				</div>
			</div>
			<div class="card-body" style="padding:0">
				{#if blocks.data?.blocks.length}
					{#each blocks.data.blocks as b (b.number)}
						<div class="live-row">
							<a
								class="num"
								href={fmt.explorerBlock(b.number)}
								target="_blank"
								rel="external noopener noreferrer"
								style="text-decoration:none">#{b.number}</a
							>
							<span class="age">{fmt.tsAge(b.timestamp)}</span>
							<span class="txs">{b.tx_count ?? 0} txs</span>
							<span class="fees">{fmt.pct(b.utilization_pct)}</span>
						</div>
					{/each}
				{:else}
					<p class="mono muted" style="padding:32px;text-align:center;font-size:11px">loading…</p>
				{/if}
			</div>
		</div>

		<div class="card">
			<div class="card-head">
				<span class="dot acc"></span>
				<div class="card-title">Latest transactions</div>
				<div class="card-sub">live</div>
				<div class="card-actions">
					<a class="mono dim" style="font-size:10px" href={resolve('/txs/')}>SEE ALL →</a>
				</div>
			</div>
			<div class="card-body" style="padding:0">
				{#if transactions.data?.transactions.length}
					{#each transactions.data.transactions as t (t.hash)}
						<div class="live-row" style="white-space:nowrap;overflow:hidden">
							<a
								class="hash mono"
								href={fmt.explorerTx(t.hash)}
								target="_blank"
								rel="external noopener noreferrer"
								style="font-size:11px;min-width:130px;overflow:hidden;text-overflow:ellipsis;text-decoration:none"
								>{fmt.hash(t.hash)}</a
							>
							<span
								class="mono"
								style="font-size:10px;color:var(--info);width:80px;overflow:hidden;text-overflow:ellipsis"
								>{fmt.methodName(t.sighash)}</span
							>
							<a
								class="addr mono"
								href={fmt.explorerAddr(t.from_addr)}
								target="_blank"
								rel="external noopener noreferrer"
								style="font-size:11px;overflow:hidden;text-overflow:ellipsis;text-decoration:none"
								>{fmt.addr(t.from_addr)}</a
							>
							<span class="arrow mono muted" style="margin-left:auto;font-size:10px"
								>{t.status === 1 ? '✓' : '✗'}</span
							>
						</div>
					{/each}
				{:else}
					<p class="mono muted" style="padding:32px;text-align:center;font-size:11px">loading…</p>
				{/if}
			</div>
		</div>
	</div>

	<!-- Top agents + FX -->
	<div class="grid grid-2" style="margin-top:12px">
		<div class="card">
			<div class="card-head">
				<div class="card-title">Top agents · 24h</div>
				<div class="card-sub">ERC-8004 · by tx count</div>
				<div class="card-actions">
					<a class="mono dim" style="font-size:10px" href={resolve('/agents/')}>REGISTRY →</a>
				</div>
			</div>
			<div class="card-body" style="padding:0">
				{#if analyticsAgentLeaderboard.data?.leaderboard.length}
					{#each analyticsAgentLeaderboard.data.leaderboard as a, i (a.agent_address)}
						<div class="agent-row">
							<div class="agent-avatar">{i + 1}</div>
							<div class="agent-meta">
								<a
									class="agent-name addr"
									href={fmt.explorerAddr(a.agent_address)}
									target="_blank"
									rel="external noopener noreferrer"
									style="text-decoration:none">{fmt.addr(a.agent_address)}</a
								>
								<div class="agent-sub">{a.tx_count ?? 0} txs · {a.job_count ?? 0} jobs</div>
							</div>
							<div class="agent-stats">
								<div>
									<span class="s-lbl">volume</span>
									{fmt.usdc(a.usdc_transferred)}
								</div>
							</div>
						</div>
					{/each}
				{:else}
					<p class="mono muted" style="padding:32px;text-align:center;font-size:11px">loading…</p>
				{/if}
			</div>
		</div>

		<div class="card">
			<div class="card-head">
				<div class="card-title">StableFX · live trades</div>
				<div class="card-sub">RFQ · recent</div>
				<div class="card-actions">
					<a class="mono dim" style="font-size:10px" href={resolve('/fx/')}>FX BOOK →</a>
				</div>
			</div>
			<div class="card-body" style="padding:0">
				{#if fx.data?.trades.length}
					{#each fx.data.trades as t, i (i)}
						<div class="flow">
							<span class="chain mono" style="font-size:10px"
								>{(t.input_token as string) ?? '?'}/{(t.output_token as string) ?? '?'}</span
							>
							<span class="mono" style="font-size:11px;margin-left:6px"
								>{fmt.usdc(t.input_amount as string)}</span
							>
							<span class="spacer"></span>
							<span class="badge {fmt.fxBadge(t.status)}">{t.status}</span>
							<span class="sub">{fmt.blockAge(t.block_number, latestBlock)}</span>
						</div>
					{/each}
				{:else}
					<p class="mono muted" style="padding:32px;text-align:center;font-size:11px">loading…</p>
				{/if}
			</div>
		</div>
	</div>
</div>
