<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { stats } from '$lib/stores/stats.svelte';
	import {
		liveBlocks,
		liveTransactions,
		seedLiveBlocks,
		seedLiveTransactions
	} from '$lib/stores/chain.svelte';
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
	import { setRealtimeWindow, connectCharts, disconnectCharts } from '$lib/realtime';
	import Chart from '$lib/components/Chart.svelte';
	import AddrLink from '$lib/components/AddrLink.svelte';
	import TxLink from '$lib/components/TxLink.svelte';
	import DataState from '$lib/components/DataState.svelte';

	type Window = '1h' | '24h' | '7d';
	let selectedWindow = $state<Window>('24h');

	function refreshAnalytics() {
		fetchBlockStats(50);
		fetchAnalyticsOverview({ window: selectedWindow });
		fetchAnalyticsBridgeFlow({ window: selectedWindow });
		fetchAnalyticsVolume({ window: selectedWindow });
		fetchAgentLeaderboard(5);
	}

	onMount(() => {
		setRealtimeWindow(selectedWindow);
		connectCharts();
		seedLiveBlocks(10);
		seedLiveTransactions(10);
		const lbId = setInterval(() => fetchAgentLeaderboard(5), 60000);
		return () => {
			clearInterval(lbId);
			disconnectCharts();
		};
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
	const largestTransfer = $derived({
		v: analyticsOverview.data?.largest_transfer ?? 0,
		block: analyticsOverview.data?.largest_transfer_block ?? 0
	});

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
		<div class="view-actions">
			{#each ['1h', '24h', '7d'] as Window[] as w (w)}
				<button
					class="btn ghost {selectedWindow === w ? 'active' : ''}"
					onclick={() => {
						selectedWindow = w;
						setRealtimeWindow(w);
						refreshAnalytics();
					}}>{w}</button
				>
			{/each}
		</div>
	</div>

	<!-- Stats row -->
	<div class="grid grid-stats">
		<div class="stat">
			<div class="label">TPS</div>
			<div class="value">
				{#key stats.data?.block_number}
					<span class="flash-val">{fmt.tps(stats.data?.tps)}</span>
				{/key}
				<span class="unit">tx/s</span>
			</div>
		</div>
		<div class="stat">
			<div class="label">Transfers · {selectedWindow}</div>
			<div class="value">
				{#key analyticsOverview.data?.snapshot_at}
					<span class="flash-val">{fmt.num(analyticsOverview.data?.transfers_count)}</span>
				{/key}
			</div>
		</div>
		<div class="stat">
			<div class="label">Volume · {selectedWindow}</div>
			<div class="value">
				{#key analyticsOverview.data?.snapshot_at}
					<span class="flash-val">{fmt.usdc(analyticsOverview.data?.transfer_volume)}</span>
				{/key}
			</div>
		</div>
		<div class="stat">
			<div class="label">Fees · {selectedWindow}</div>
			<div class="value">
				{#key analyticsOverview.data?.snapshot_at}
					<span class="flash-val">{fmt.usdc(analyticsOverview.data?.fees_total)}</span>
				{/key}
			</div>
		</div>
		<div class="stat">
			<div class="label">Bridge net flow · {selectedWindow}</div>
			<div class="value {(analyticsOverview.data?.bridge_net_flow ?? 0) >= 0 ? '' : 'err'}">
				{#key analyticsOverview.data?.snapshot_at}
					<span class="flash-val">{fmt.usdc(analyticsOverview.data?.bridge_net_flow)}</span>
				{/key}
			</div>
		</div>
		<div class="stat">
			<div class="label">Agents</div>
			<div class="value">
				{#key analyticsOverview.data?.snapshot_at}
					<span class="flash-val">{analyticsOverview.data?.agent_count ?? '—'}</span>
				{/key}
			</div>
		</div>
	</div>

	<!-- Fee analytics · stat cards -->
	{#if analyticsOverview.data}
		<div
			class="grid"
			style="grid-template-columns:repeat(auto-fit,minmax(140px,1fr));margin-top:12px"
		>
			<div class="stat">
				<div class="label">Fee p50</div>
				<div class="value" style="color:var(--info)">
					{#key analyticsOverview.data.snapshot_at}
						<span class="flash-val">{fmt.usdc(analyticsOverview.data.fee_p50, 6)}</span>
					{/key}
				</div>
			</div>
			<div class="stat">
				<div class="label">Fee p95</div>
				<div class="value" style="color:var(--warn)">
					{#key analyticsOverview.data.snapshot_at}
						<span class="flash-val">{fmt.usdc(analyticsOverview.data.fee_p95, 6)}</span>
					{/key}
				</div>
			</div>
			<div class="stat hide-mobile">
				<div class="label">Failed tx ratio</div>
				<div
					class="value"
					style="color:{analyticsOverview.data.failed_tx_ratio > 0.05 ? 'var(--err)' : 'var(--ok)'}"
				>
					{#key analyticsOverview.data.snapshot_at}
						<span class="flash-val"
							>{(analyticsOverview.data.failed_tx_ratio * 100).toFixed(2)}%</span
						>
					{/key}
				</div>
			</div>
		</div>
	{/if}

	<!-- Stablecoin volume breakdown + whales -->
	{#if volume}
		<div
			class="grid"
			style="grid-template-columns:repeat(auto-fit,minmax(140px,1fr));margin-top:12px"
		>
			<div class="stat">
				<div class="label" style="color:var(--ok)">USDC vol · {selectedWindow}</div>
				<div class="value">
					{#key analyticsOverview.data?.snapshot_at}
						<span class="flash-val">{fmt.usdc(tokenStats.USDC.volume)}</span>
					{/key}
				</div>
				<div class="mono dim" style="font-size:10px">
					{fmt.num(tokenStats.USDC.count)} transfers
				</div>
			</div>
			<div class="stat">
				<div class="label" style="color:var(--info)">EURC vol · {selectedWindow}</div>
				<div class="value">
					{#key analyticsOverview.data?.snapshot_at}
						<span class="flash-val">{fmt.usdc(tokenStats.EURC.volume)}</span>
					{/key}
				</div>
				<div class="mono dim" style="font-size:10px">
					{fmt.num(tokenStats.EURC.count)} transfers
				</div>
			</div>
			<div class="stat">
				<div class="label" style="color:var(--warn)">USYC vol · {selectedWindow}</div>
				<div class="value">
					{#key analyticsOverview.data?.snapshot_at}
						<span class="flash-val">{fmt.usdc(tokenStats.USYC.volume)}</span>
					{/key}
				</div>
				<div class="mono dim" style="font-size:10px">
					{fmt.num(tokenStats.USYC.count)} transfers
				</div>
			</div>
			<div class="stat">
				<div class="label">Whale transfers</div>
				<div class="value" style="color:var(--warn)">
					{#key analyticsOverview.data?.snapshot_at}
						<span class="flash-val">{fmt.num(volume.whale_transfers)}</span>
					{/key}
				</div>
				<div class="mono dim" style="font-size:10px">≥ $10K · {selectedWindow}</div>
			</div>
			<div class="stat hide-mobile">
				<div class="label">Largest transfer</div>
				<div class="value" style="color:var(--acc)">
					{#key analyticsOverview.data?.snapshot_at}
						<span class="flash-val">{fmt.usdc(largestTransfer.v)}</span>
					{/key}
				</div>
				<div class="mono dim" style="font-size:10px">
					{#if largestTransfer.block}
						<a
							href={resolve(`/blocks/${largestTransfer.block}/`)}
							style="text-decoration:none;color:inherit">block #{largestTransfer.block}</a
						>
					{:else}—{/if}
				</div>
			</div>
		</div>
	{/if}

	<!-- Charts -->
	{#if chartStats.length > 1}
		<div class="grid grid-2" style="margin-top:12px">
			<div class="card">
				<div class="card-head">
					<div class="card-title">TPS · last 50 blocks</div>
					<div class="card-sub">transactions per second</div>
				</div>
				<div class="card-body" style="padding:4px 8px 8px">
					<Chart title="TPS" labels={chartLabels} series={tpsSeries} height={160} />
				</div>
			</div>
			<div class="card">
				<div class="card-head">
					<div class="card-title">Avg fee · last 50 blocks</div>
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
					<div class="card-title">Tx count · last 50 blocks</div>
					<div class="card-sub">transactions per block</div>
				</div>
				<div class="card-body" style="padding:4px 8px 8px">
					<Chart title="Tx count" labels={chartLabels} series={txCountSeries} height={160} />
				</div>
			</div>
			<div class="card">
				<div class="card-head">
					<div class="card-title">Gas utilization · last 50 blocks</div>
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
				{#if liveBlocks.data?.blocks.length}
					{#each liveBlocks.data.blocks as b (b.number)}
						<div class="live-row">
							<a class="num" href={resolve(`/blocks/${b.number}/`)} style="text-decoration:none"
								>#{b.number}</a
							>
							<span class="age">{fmt.tsAge(b.timestamp)}</span>
							<span class="txs">{b.tx_count ?? 0} txs</span>
							<span class="fees">{fmt.pct(b.utilization_pct)}</span>
						</div>
					{/each}
				{:else}
					<DataState loading={liveBlocks.loading} error={liveBlocks.error} label="blocks" />
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
				{#if liveTransactions.data?.transactions.length}
					{#each liveTransactions.data.transactions as t (t.hash)}
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
					<DataState
						loading={liveTransactions.loading}
						error={liveTransactions.error}
						label="transactions"
					/>
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
					<DataState
						loading={analyticsAgentLeaderboard.loading}
						error={analyticsAgentLeaderboard.error}
						label="agents"
					/>
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
							<span class="chain"
								><a
									href={resolve(`/crosschain/${fmt.domainId(chain) ?? 0}/`)}
									style="text-decoration:none;color:inherit">{chain}</a
								></span
							>
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
					<DataState
						loading={analyticsBridgeFlow.loading}
						error={analyticsBridgeFlow.error}
						label="bridge data"
						compact
					/>
				{/if}
			</div>
		</div>
	</div>
</div>
