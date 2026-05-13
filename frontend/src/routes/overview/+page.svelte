<script lang="ts">
	const stats = [
		{ label: 'TPS' },
		{ label: 'Block time' },
		{ label: 'Transfers 24h' },
		{ label: 'Fees paid 24h' },
		{ label: 'FX notional 24h' },
		{ label: 'Active agents' }
	];
</script>

<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Overview</div>
			<div class="view-sub">Live chain state · arc testnet</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost">Pause stream</button>
			<button class="btn">Export</button>
		</div>
	</div>

	<div class="grid grid-stats">
		{#each stats as s (s.label)}
			<div class="stat">
				<div class="label">{s.label}</div>
				<div class="value">—</div>
			</div>
		{/each}
	</div>

	<div class="grid grid-2" style="margin-top:12px">
		<div class="card">
			<div class="card-head">
				<div class="card-title">Throughput · transfer volume</div>
				<div class="card-sub">60 blocks · USDC equiv.</div>
			</div>
			<div class="card-body" style="height:180px;display:grid;place-items:center">
				<span class="mono dim" style="font-size:11px">chart coming soon</span>
			</div>
		</div>
		<div class="card">
			<div class="card-head">
				<div class="card-title">Cross-chain pulse</div>
				<div class="card-sub">CCTP · 24h</div>
			</div>
			<div class="card-body" style="height:180px;display:grid;place-items:center">
				<span class="mono dim" style="font-size:11px">loading…</span>
			</div>
		</div>
	</div>

	<div class="grid grid-2" style="margin-top:12px">
		<div class="card">
			<div class="card-head">
				<span class="dot acc"></span>
				<div class="card-title">Latest blocks</div>
				<div class="card-sub">streaming</div>
				<div class="card-actions">
					<span class="mono dim" style="font-size:10px">SEE ALL →</span>
				</div>
			</div>
			<div class="card-body" style="height:200px;display:grid;place-items:center">
				<span class="mono dim" style="font-size:11px">loading…</span>
			</div>
		</div>
		<div class="card">
			<div class="card-head">
				<span class="dot acc"></span>
				<div class="card-title">Latest transactions</div>
				<div class="card-sub">streaming · all kinds</div>
				<div class="card-actions">
					<span class="mono dim" style="font-size:10px">SEE ALL →</span>
				</div>
			</div>
			<div class="card-body" style="height:200px;display:grid;place-items:center">
				<span class="mono dim" style="font-size:11px">loading…</span>
			</div>
		</div>
	</div>

	<div class="grid grid-2" style="margin-top:12px">
		<div class="card">
			<div class="card-head">
				<div class="card-title">Top agents · last 24h</div>
				<div class="card-sub">ERC-8004</div>
				<div class="card-actions">
					<span class="mono dim" style="font-size:10px">REGISTRY →</span>
				</div>
			</div>
			<div class="card-body" style="height:180px;display:grid;place-items:center">
				<span class="mono dim" style="font-size:11px">loading…</span>
			</div>
		</div>
		<div class="card">
			<div class="card-head">
				<div class="card-title">StableFX · live trades</div>
				<div class="card-sub">RFQ · last hour</div>
				<div class="card-actions">
					<span class="mono dim" style="font-size:10px">FX BOOK →</span>
				</div>
			</div>
			<div class="card-body" style="height:180px;display:grid;place-items:center">
				<span class="mono dim" style="font-size:11px">loading…</span>
			</div>
		</div>
	</div>
</div>
