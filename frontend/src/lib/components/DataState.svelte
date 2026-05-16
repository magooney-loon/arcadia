<script lang="ts">
	// Inline empty/loading/error placeholder. Reads the global `health`
	// store so empty states during a heavy indexer sync show "catching
	// up · N blocks behind" instead of a permanent "loading…", which
	// reassures users that REST stalls are temporary and explains why.
	//
	// Use inside the `{:else}` branch of a `{#if data.length}` block:
	//
	//   {#if blocks.data?.blocks.length}
	//     …rows…
	//   {:else}
	//     <DataState loading={blocks.loading} error={blocks.error} />
	//   {/if}

	import { health } from '$lib/stores/health.svelte';

	interface Props {
		loading: boolean;
		error?: string | null;
		compact?: boolean;
		label?: string;
		// When set, renders as <tr><td colspan>…</td></tr> for use
		// inside a <tbody>. Otherwise renders as a centered <div>.
		colspan?: number;
	}

	let { loading, error = null, compact = false, label = 'data', colspan = 0 }: Props = $props();

	const syncing = $derived(health.data?.syncing ?? false);
	const lag = $derived(health.data?.lag_blocks ?? 0);
</script>

{#snippet body()}
	{#if error}
		<span class="dot err"></span>
		<span class="mono err-text">{error}</span>
	{:else if loading}
		<span class="spinner"></span>
		<span class="mono muted">loading {label}…</span>
	{:else if syncing}
		<span class="pulse-dot warn"></span>
		<span class="mono">indexer catching up</span>
		<span class="mono muted">· {lag} blocks behind · {label} will appear shortly</span>
	{:else}
		<span class="mono muted">no {label} yet</span>
	{/if}
{/snippet}

{#if colspan > 0}
	<tr>
		<td {colspan} class="ds-cell">
			<div class="ds inline" class:compact>{@render body()}</div>
		</td>
	</tr>
{:else}
	<div class="ds" class:compact>{@render body()}</div>
{/if}

<style>
	.ds {
		display: flex;
		align-items: center;
		gap: 8px;
		justify-content: center;
		padding: 32px 16px;
		font-size: 11px;
		flex-wrap: wrap;
		text-align: center;
	}
	.ds.compact {
		padding: 14px 12px;
	}
	.ds.inline {
		padding: 24px 16px;
	}
	.ds-cell {
		text-align: center;
		background: var(--bg-1, transparent);
	}
	.spinner {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		border: 1.5px solid var(--border-2, #333);
		border-top-color: var(--accent);
		animation: spin 0.7s linear infinite;
		flex-shrink: 0;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	.pulse-dot.warn {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--warn);
		box-shadow: 0 0 0 0 var(--warn);
		animation: pulse 1.6s ease-out infinite;
		flex-shrink: 0;
	}
	@keyframes pulse {
		0% {
			box-shadow: 0 0 0 0 rgba(240, 180, 41, 0.6);
		}
		100% {
			box-shadow: 0 0 0 8px rgba(240, 180, 41, 0);
		}
	}
	.dot.err {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: var(--err);
		flex-shrink: 0;
	}
	.err-text {
		color: var(--err);
	}
</style>
