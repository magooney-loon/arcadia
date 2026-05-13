<script lang="ts">
	const agentStats = [
		{ label: 'Total registered' },
		{ label: 'Active 24h' },
		{ label: 'Jobs in-flight' },
		{ label: 'Avg trust score' }
	];
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Agent registry</div>
			<div class="view-sub">ERC-8004 · registered agents on arc testnet</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost">Filter</button>
			<button class="btn acc">+ Register agent</button>
		</div>
	</div>

	<div class="grid" style="grid-template-columns:repeat(4,1fr);margin-bottom:12px">
		{#each agentStats as s (s.label)}
			<div class="stat">
				<div class="label">{s.label}</div>
				<div class="value">—</div>
			</div>
		{/each}
	</div>

	<div class="filter-bar">
		<span class="chip on">all</span>
		<span class="chip">active</span>
		<span class="chip">idle</span>
		<span class="chip">market_making</span>
		<span class="chip">settlement</span>
		<span class="chip">compliance</span>
	</div>

	<div class="card">
		<div class="card-body flush">
			<div class="mono dim" style="padding:32px;text-align:center;font-size:11px">loading…</div>
		</div>
	</div>
</div>
