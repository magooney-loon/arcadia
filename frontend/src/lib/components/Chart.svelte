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
	let tooltip: HTMLDivElement | null = null;

	function buildChart() {
		if (plot) {
			plot.destroy();
			plot = null;
		}

		if (tooltip) {
			tooltip.remove();
			tooltip = null;
		}

		if (!container || !labels.length || !series.length) return;

		const containerWidth = container.clientWidth || 400;

		// Create tooltip element imperatively
		const mount = container.parentElement?.querySelector('.chart-tooltip-mount');
		if (mount) {
			tooltip = document.createElement('div');
			tooltip.className = 'chart-tooltip';
			tooltip.style.display = 'none';
			mount.appendChild(tooltip);
		}

		// Convert to uPlot's AlignedData format (Float64Arrays)
		const xData = Float64Array.from(labels);
		const seriesData = series.map((s) => Float64Array.from(s.data, (v) => v ?? 0));
		const data = [xData, ...seriesData];

		const seriesDefs = series.map((s) => ({
			label: s.label,
			stroke: s.stroke ?? '#6dd5fa',
			fill: s.fill ?? 'rgba(109,213,250,0.08)',
			width: s.width ?? 1.5,
			spanGaps: true,
			points: { show: false }
		}));

		const currentTooltip = tooltip;

		const opts: UPlot.Options = {
			title: title || '',
			width: containerWidth,
			height,
			class: 'arc-chart',
			pxAlign: false,
			cursor: {
				drag: { x: true, y: true, uni: 50 },
				focus: { prox: 30 }
			},
			hooks: {
				setCursor: [
					(u) => {
						if (!currentTooltip) return;
						const idx = u.cursor.idx;
						if (idx == null) {
							currentTooltip.style.display = 'none';
							return;
						}

						const time = new Date(u.data[0][idx] * 1000);
						const pad = (n: number) => String(n).padStart(2, '0');
						const timeStr = `${pad(time.getHours())}:${pad(time.getMinutes())}:${pad(time.getSeconds())}`;

						let html = `<div class="tt-time">${timeStr}</div>`;
						for (let i = 0; i < seriesDefs.length; i++) {
							const val = u.data[i + 1][idx];
							const color = seriesDefs[i].stroke;
							const lbl = seriesDefs[i].label;
							html += `<div class="tt-row"><span class="tt-dot" style="background:${color}"></span><span class="tt-lbl">${lbl}</span><span class="tt-val">${val != null ? val.toPrecision(4) : '—'}</span></div>`;
						}

						currentTooltip.innerHTML = html;
						currentTooltip.style.display = 'block';

						// Position tooltip near cursor using u-over offset
						if (!mount) return;
						const over = u.over;
						const overRect = over.getBoundingClientRect();
						const mountRect = mount.getBoundingClientRect();
						const cursorLeft = overRect.left - mountRect.left + (u.cursor.left || 0);
						const cursorTop = overRect.top - mountRect.top + (u.cursor.top || 0);

						let left = cursorLeft + 16;
						let top = cursorTop - currentTooltip.offsetHeight - 10;
						if (top < 0) top = cursorTop + 16;
						if (left + currentTooltip.offsetWidth + 10 > mountRect.width) {
							left = cursorLeft - currentTooltip.offsetWidth - 10;
						}
						currentTooltip.style.left = left + 'px';
						currentTooltip.style.top = top + 'px';
					}
				]
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
			series: [{}, ...seriesDefs]
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
			if (tooltip) tooltip.remove();
		};
	});
</script>

<div bind:this={container} class="arc-chart-wrap"></div>
<div class="chart-tooltip-mount" style="position:relative"></div>

<style>
	.arc-chart-wrap {
		width: 100%;
		min-height: 0;
		position: relative;
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
