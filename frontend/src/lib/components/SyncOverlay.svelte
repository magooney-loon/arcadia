<script lang="ts">
	import { health } from '$lib/stores/health.svelte';

	const SYNC_MODAL_SHOW = 20;
	const SYNC_MODAL_HIDE = 1;
	let syncModalOpen = $state(false);
	let startBlock = $state(0);
	$effect(() => {
		const lag = health.data?.lag_blocks ?? 0;
		if (!health.data) return;
		if (syncModalOpen) {
			if (lag <= SYNC_MODAL_HIDE) syncModalOpen = false;
		} else {
			if (lag > SYNC_MODAL_SHOW) {
				syncModalOpen = true;
				startBlock = health.data?.last_indexed_block ?? 0;
			}
		}
	});

	const syncProgressPct = $derived.by(() => {
		const head = health.data?.last_indexed_block ?? 0;
		const tip = health.data?.chain_tip ?? 0;
		const span = tip - startBlock;
		if (span <= 0) return 100;
		const progress = head - startBlock;
		return Math.round(Math.min(100, Math.max(0, (progress / span) * 100)));
	});
</script>

<!-- Fullscreen indexer-syncing gate. Blocks all input until the indexer
     catches up enough (lag <= SYNC_MODAL_HIDE blocks). aria-modal + focus
     trap left intentionally simple: this is a transient state and the
     user can't actually do anything productive underneath. -->
{#if syncModalOpen}
	<div
		class="sync-modal"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="sync-modal-title"
		aria-describedby="sync-modal-desc"
	>
		<div class="sync-card">
			<div class="sync-pulse-ring">
				<svg viewBox="0 0 40 40" fill="none" aria-hidden="true">
					<circle cx="20" cy="20" r="14" stroke="var(--accent)" stroke-width="2" opacity="0.3" />
					<circle
						cx="20"
						cy="20"
						r="14"
						stroke="var(--accent)"
						stroke-width="2"
						stroke-dasharray="22 88"
						stroke-linecap="round"
						class="sync-spin"
					/>
				</svg>
			</div>
			<div class="sync-title" id="sync-modal-title">Indexer is catching up</div>
			<div class="sync-desc" id="sync-modal-desc">
				Arcadia is replaying recent blocks. The dashboard is paused so the indexer can finish
				without read contention. This usually clears within a minute.
			</div>

			<div class="sync-stats">
				<div class="sync-stat">
					<div class="sync-stat-label">behind by</div>
					<div class="sync-stat-val warn">
						{health.data?.lag_blocks ?? '—'}<span class="sync-unit"> blocks</span>
					</div>
				</div>
				<div class="sync-stat">
					<div class="sync-stat-label">indexed</div>
					<div class="sync-stat-val mono">#{health.data?.last_indexed_block ?? '—'}</div>
				</div>
				<div class="sync-stat">
					<div class="sync-stat-label">chain tip</div>
					<div class="sync-stat-val mono">#{health.data?.chain_tip ?? '—'}</div>
				</div>
			</div>

			<div class="sync-progress" aria-hidden="true">
				<div class="sync-progress-fill" style="width:{syncProgressPct}%"></div>
			</div>

			<div class="sync-footnote">Live updates over SSE — the page will resume automatically.</div>
		</div>
	</div>
{/if}

<style>
	.sync-modal {
		position: fixed;
		inset: 0;
		z-index: 10000;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 24px;
		background: rgba(8, 12, 18, 0.78);
		backdrop-filter: blur(6px);
		-webkit-backdrop-filter: blur(6px);
		animation: sync-modal-in 180ms ease-out;
	}
	@keyframes sync-modal-in {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}
	.sync-card {
		max-width: 440px;
		width: 100%;
		background: var(--bg-1);
		border: 1px solid var(--border-1);
		border-radius: 8px;
		padding: 28px 28px 22px;
		box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
		text-align: center;
	}
	.sync-pulse-ring {
		width: 56px;
		height: 56px;
		margin: 0 auto 18px;
	}
	.sync-pulse-ring svg {
		width: 100%;
		height: 100%;
	}
	.sync-spin {
		transform-origin: center;
		animation: sync-spin 1.1s linear infinite;
	}
	@keyframes sync-spin {
		to {
			transform: rotate(360deg);
		}
	}
	.sync-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--fg-0);
		letter-spacing: 0.2px;
		margin-bottom: 6px;
	}
	.sync-desc {
		font-size: 12px;
		line-height: 1.55;
		color: var(--fg-3);
		margin: 0 auto 18px;
		max-width: 360px;
	}
	.sync-stats {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 8px;
		padding: 12px 0;
		border-top: 1px solid var(--border-1);
		border-bottom: 1px solid var(--border-1);
		margin-bottom: 14px;
	}
	.sync-stat-label {
		font-size: 9px;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		color: var(--fg-3);
		margin-bottom: 3px;
	}
	.sync-stat-val {
		font-size: 14px;
		font-weight: 600;
		color: var(--fg-0);
		font-variant-numeric: tabular-nums;
	}
	.sync-stat-val.warn {
		color: var(--warn);
	}
	.sync-stat-val.mono {
		font-family: ui-monospace, 'SF Mono', Menlo, monospace;
		font-size: 12px;
		font-weight: 500;
	}
	.sync-unit {
		font-size: 10px;
		font-weight: 400;
		color: var(--fg-3);
		margin-left: 2px;
	}
	.sync-progress {
		height: 3px;
		background: var(--border-1);
		border-radius: 2px;
		overflow: hidden;
		margin-bottom: 10px;
	}
	.sync-progress-fill {
		height: 100%;
		background: linear-gradient(90deg, var(--accent), var(--info));
		transition: width 400ms ease-out;
	}
	.sync-footnote {
		font-size: 10px;
		color: var(--fg-3);
		opacity: 0.7;
	}
</style>
