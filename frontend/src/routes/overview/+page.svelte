<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { stats, fetchStats } from '$lib/stores/stats.svelte';
	import { blocks, transactions, fetchBlocks, fetchTransactions } from '$lib/stores/chain.svelte';
	import { blockStats, fetchBlockStats } from '$lib/stores/blockStats.svelte';
	import {
		analyticsOverview,
		fetchAnalyticsOverview,
		analyticsBridgeFlow,
		fetchAnalyticsBridgeFlow,
		analyticsVolume,
		fetchAnalyticsVolume,
		analyticsAgentLeaderboard,
		fetchAgentLeaderboard
	} from '$lib/stores/analytics.svelte';
	import * as fmt from '$lib/fmt.js';
	import Chart from '$lib/components/Chart.svelte';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import TxLink from '$lib/components/TxLink.svelte';

	onMount(() => {
		const refresh = () => {
			fetchStats();
			fetchBlocks(10);
			fetchTransactions({ limit: 10 });
		};
		refresh();
		fetchBlockStats(200);
		fetchAgentLeaderboard(5);
		fetchAnalyticsOverview();
		fetchAnalyticsBridgeFlow();
		fetchAnalyticsVolume();
		const id = setInterval(refresh, 6000);
		return () => clearInterval(id);
	});

	const bridgeFlow = $derived(analyticsBridgeFlow.data);
	const chainEntries = $derived(
		Object.entries(bridgeFlow?.by_chain ?? {})
			.sort((a, b) => b[1].inbound_vol + b[1].outbound_vol - (a[1].inbound_vol + a[1].outbound_vol))
			.slice(0, 7)
	);

	const volume = $derived(analyticsVolume.data);
	const tokenStats = $derived({
		USDC: volume?.by_token?.USDC ?? { volume: 0, count: 0, whale_count: 0 },
		EURC: volume?.by_token?.EURC ?? { volume: 0, count: 0, whale_count: 0 },
		USYC: volume?.by_token?.USYC ?? { volume: 0, count: 0, whale_count: 0 }
	});
	// Latest block_stats row carries the largest single USDC transfer in that block.
	// Show the max across the loaded 200-block window.
	const largestTransfer = $derived(
		(blockStats.data?.stats ?? []).reduce(
			(max, s) => {
				const v = parseFloat(s.largest_usdc_transfer ?? '0');
				return v > max.v ? { v, block: s.block_number ?? 0 } : max;
			},
			{ v: 0, block: 0 }
		)
	);

	// Chart data from block stats (sorted oldest → newest)
	const chartStats = $derived((blockStats.data?.stats ?? []).slice().reverse());
	const chartLabels = $derived(chartStats.map((s) => s.timestamp ?? 0));
	const tpsSeries = $derived([
		{
			label: 'TPS',
			data: chartStats.map((s) => s.tps ?? null),
			stroke: '#6dd5fa',
			fill: 'rgba(109,213,250,0.1)'
		}
	]);
	const feeSeries = $derived([
		{
			label: 'Avg fee',
			data: chartStats.map((s) => (s.avg_fee_usdc ? parseFloat(s.avg_fee_usdc) : null)),
			stroke: '#f0b429',
			fill: 'rgba(240,180,41,0.08)'
		}
	]);
	const txCountSeries = $derived([
		{
			label: 'Tx count',
			data: chartStats.map((s) => s.tx_count ?? null),
			stroke: '#47d16c',
			fill: 'rgba(71,209,108,0.08)'
		}
	]);
	const utilizationSeries = $derived([
		{
			label: 'Utilization %',
			data: chartStats.map((s) => s.utilization_pct ?? null),
			stroke: '#e05252',
			fill: 'rgba(224,82,82,0.08)'
		}
	]);
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
			<div class="label">Transfers 24h</div>
			<div class="value">{fmt.num(analyticsOverview.data?.transfers_count)}</div>
		</div>
		<div class="stat">
			<div class="label">Volume 24h</div>
			<div class="value">{fmt.usdc(analyticsOverview.data?.transfer_volume)}</div>
		</div>
		<div class="stat">
			<div class="label">Fees 24h</div>
			<div class="value">{fmt.usdc(analyticsOverview.data?.fees_total)}</div>
		</div>
		<div class="stat">
			<div class="label">Bridge net flow 24h</div>
			<div class="value {(analyticsOverview.data?.bridge_net_flow ?? 0) >= 0 ? '' : 'err'}">
				{fmt.usdc(analyticsOverview.data?.bridge_net_flow)}
			</div>
		</div>
		<div class="stat">
			<div class="label">Agents</div>
			<div class="value">{analyticsOverview.data?.agent_count ?? '—'}</div>
		</div>
	</div>

	<!-- Fee analytics · stat cards -->
	{#if analyticsOverview.data}
		<div class="grid" style="grid-template-columns:repeat(3,1fr);margin-top:12px">
			<div class="stat">
				<div class="label">Fee p50</div>
				<div class="value" style="color:var(--info)">
					{fmt.usdc(analyticsOverview.data.fee_p50, 6)}
				</div>
			</div>
			<div class="stat">
				<div class="label">Fee p95</div>
				<div class="value" style="color:var(--warn)">
					{fmt.usdc(analyticsOverview.data.fee_p95, 6)}
				</div>
			</div>
			<div class="stat hide-mobile">
				<div class="label">Failed tx ratio</div>
				<div
					class="value"
					style="color:{analyticsOverview.data.failed_tx_ratio > 0.05 ? 'var(--err)' : 'var(--ok)'}"
				>
					{(analyticsOverview.data.failed_tx_ratio * 100).toFixed(2)}%
				</div>
			</div>
		</div>
	{/if}

	<!-- Stablecoin volume breakdown + whales -->
	{#if volume}
		<div class="grid" style="grid-template-columns:repeat(5,1fr);margin-top:12px">
			<div class="stat">
				<div class="label" style="color:var(--ok)">USDC vol 24h</div>
				<div class="value">{fmt.usdc(tokenStats.USDC.volume)}</div>
				<div class="mono dim" style="font-size:10px">
					{fmt.num(tokenStats.USDC.count)} transfers
				</div>
			</div>
			<div class="stat">
				<div class="label" style="color:var(--info)">EURC vol 24h</div>
				<div class="value">{fmt.usdc(tokenStats.EURC.volume)}</div>
				<div class="mono dim" style="font-size:10px">
					{fmt.num(tokenStats.EURC.count)} transfers
				</div>
			</div>
			<div class="stat">
				<div class="label" style="color:var(--warn)">USYC vol 24h</div>
				<div class="value">{fmt.usdc(tokenStats.USYC.volume)}</div>
				<div class="mono dim" style="font-size:10px">
					{fmt.num(tokenStats.USYC.count)} transfers
				</div>
			</div>
			<div class="stat">
				<div class="label">Whale transfers</div>
				<div class="value" style="color:var(--warn)">{fmt.num(volume.whale_transfers)}</div>
				<div class="mono dim" style="font-size:10px">≥ $10K · 24h</div>
			</div>
			<div class="stat hide-mobile">
				<div class="label">Largest transfer</div>
				<div class="value" style="color:var(--acc)">{fmt.usdc(largestTransfer.v)}</div>
				<div class="mono dim" style="font-size:10px">
					{largestTransfer.block ? `block #${largestTransfer.block}` : '—'}
				</div>
			</div>
		</div>
	{/if}

	<!-- Charts -->
	{#if chartStats.length > 1}
		<div class="grid grid-2" style="margin-top:12px">
			<div class="card">
				<div class="card-head">
					<div class="card-title">TPS · last 200 blocks</div>
					<div class="card-sub">transactions per second</div>
				</div>
				<div class="card-body" style="padding:4px 8px 8px">
					<Chart title="TPS" labels={chartLabels} series={tpsSeries} height={160} />
				</div>
			</div>
			<div class="card">
				<div class="card-head">
					<div class="card-title">Avg fee · last 200 blocks</div>
					<div class="card-sub">USDC per transaction</div>
				</div>
				<div class="card-body" style="padding:4px 8px 8px">
					<Chart title="Fee" labels={chartLabels} series={feeSeries} height={160} />
				</div>
			</div>
		</div>
		<div class="grid grid-2" style="margin-top:12px">
			<div class="card">
				<div class="card-head">
					<div class="card-title">Tx count · last 200 blocks</div>
					<div class="card-sub">transactions per block</div>
				</div>
				<div class="card-body" style="padding:4px 8px 8px">
					<Chart title="Tx count" labels={chartLabels} series={txCountSeries} height={160} />
				</div>
			</div>
			<div class="card">
				<div class="card-head">
					<div class="card-title">Gas utilization · last 200 blocks</div>
					<div class="card-sub">block capacity used</div>
				</div>
				<div class="card-body" style="padding:4px 8px 8px">
					<Chart title="Utilization" labels={chartLabels} series={utilizationSeries} height={160} />
				</div>
			</div>
		</div>
	{/if}

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
							<TxLink hash={t.hash} />
							<span
								class="mono"
								style="font-size:10px;color:var(--info);width:80px;overflow:hidden;text-overflow:ellipsis"
								>{fmt.methodName(t.sighash)}</span
							>
							<AddrLink address={t.from_addr} />
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
				<div class="card-sub">ERC-8004 · by volume</div>
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
								<AddrLink address={a.agent_address} />
								<div class="agent-sub">{a.tx_count ?? 0} txs · {a.job_count ?? 0} jobs</div>
							</div>
							<div class="agent-stats">
								<div>
									<span class="s-lbl">volume</span>
									{fmt.usdc(a.usdc_transferred_human)}
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
				<div class="card-title">Cross-chain pulse</div>
				<div class="card-sub">CCTP · 24h by chain</div>
				<div class="card-actions">
					<a class="mono dim" style="font-size:10px" href={resolve('/crosschain/')}>SEE ALL →</a>
				</div>
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
</div>
