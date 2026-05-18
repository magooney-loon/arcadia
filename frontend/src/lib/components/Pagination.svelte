<script lang="ts">
	interface Props {
		offset: number;
		limit: number;
		total: number;
		onPrev: () => void;
		onNext: () => void;
	}

	let { offset, limit, total, onPrev, onNext }: Props = $props();

	const page = $derived(Math.floor(offset / limit) + 1);
	const totalPages = $derived(total > 0 ? Math.max(1, Math.ceil(total / limit)) : 1);
	const hasNext = $derived(offset + limit < total);
	const hasPrev = $derived(offset > 0);
	const rangeEnd = $derived(Math.min(offset + limit, total));
</script>

<div class="filter-bar" style="margin-top:10px;justify-content:flex-end">
	<button class="btn ghost" disabled={!hasPrev} onclick={onPrev}>← prev</button>
	<span class="mono dim" style="font-size:11px">
		{#if total > 0}
			{offset + 1}–{rangeEnd} of {total.toLocaleString()} · page {page} / {totalPages}
		{:else}
			no results
		{/if}
	</span>
	<button class="btn ghost" disabled={!hasNext} onclick={onNext}>next →</button>
</div>
