<script lang="ts">
	import { onMount } from 'svelte';
	import UPlot from 'uplot';
	import 'uplot/dist/uPlot.min.css';

	interface Series {
		label: string;
		data: (number | null)[];
		stroke?: string;
		fill?: string;
		width?: number;
	}

	interface Props {
		title?: string;
		labels: number[];
		series: Series[];
		height?: number;
	}

	let { title, labels, series, height = 200 }: Props = $props();

	let container: HTMLDivElement;
	let plot: UPlot | null = null;

	function buildChart() {
		if (plot) {
			plot.destroy();
			plot = null;
		}

		if (!container || !labels.length || !series.length) return;

		const containerWidth = container.clientWidth || 400;

		// Convert to uPlot's AlignedData format (Float64Arrays)
		const xData = Float64Array.from(labels);
		const seriesData = series.map((s) => Float64Array.from(s.data, (v) => v ?? 0));
		const data = [xData, ...seriesData];

		const opts: UPlot.Options = {
			title: title || '',
			width: containerWidth,
			height,
			class: 'arc-chart',
			pxAlign: false,
			cursor: {
				drag: { x: true, y: true, uni: 50 }
			},
			scales: {
				x: { time: true },
				y: { auto: true }
			},
			axes: [
				{
					stroke: 'rgba(255,255,255,0.15)',
					grid: { stroke: 'rgba(255,255,255,0.04)' },
					ticks: { stroke: 'rgba(255,255,255,0.1)' },
					font: '10px monospace',
					labelFont: '10px monospace',
					gap: 4,
					size: 28
				},
				{
					stroke: 'rgba(255,255,255,0.15)',
					grid: { stroke: 'rgba(255,255,255,0.04)' },
					ticks: { stroke: 'rgba(255,255,255,0.1)' },
					font: '10px monospace',
					labelFont: '10px monospace',
					gap: 4,
					size: 36
				}
			],
			series: [
				{},
				...series.map((s) => ({
					label: s.label,
					stroke: s.stroke ?? '#6dd5fa',
					fill: s.fill ?? 'rgba(109,213,250,0.08)',
					width: s.width ?? 1.5,
					spanGaps: true,
					points: { show: false }
				}))
			]
		};

		plot = new UPlot(opts, data, container);
	}

	$effect(() => {
		if (container && labels.length && series.length) {
			buildChart();
		}
	});

	onMount(() => {
		return () => {
			plot?.destroy();
		};
	});
</script>

<div bind:this={container} class="arc-chart-wrap"></div>

<style>
	.arc-chart-wrap {
		width: 100%;
		min-height: 0;
	}
	.arc-chart-wrap :global(.u-legend) {
		display: none;
	}
	.arc-chart-wrap :global(.u-title) {
		font-family: var(--mono, monospace);
		font-size: 11px;
		color: rgba(255, 255, 255, 0.3);
		text-align: left;
		margin-bottom: 4px;
	}
	.arc-chart-wrap :global(.u-select) {
		background: rgba(109, 213, 250, 0.08);
	}
</style>
