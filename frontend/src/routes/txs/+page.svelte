<div class="view">
	<div class="view-head">
		<div>
			<div class="view-title">Transactions</div>
			<div class="view-sub">All transaction types · streaming</div>
		</div>
		<div class="view-actions">
			<button class="btn ghost">Filter</button>
			<button class="btn">Export</button>
		</div>
	</div>

	<div class="filter-bar">
		<span class="chip on">all</span>
		<span class="chip">transfer</span>
		<span class="chip">swap</span>
		<span class="chip">mint</span>
		<span class="chip">burn</span>
	</div>

	<div class="card">
		<div class="card-body flush">
			<table class="tbl">
				<thead>
					<tr>
						<th>hash</th>
						<th>type</th>
						<th>from</th>
						<th></th>
						<th>to</th>
						<th class="num">age</th>
					</tr>
				</thead>
				<tbody>
					<tr
						><td colspan="6" style="text-align:center;color:var(--fg-4);padding:32px" class="mono"
							>loading…</td
						></tr
					>
				</tbody>
			</table>
		</div>
	</div>
</div>
