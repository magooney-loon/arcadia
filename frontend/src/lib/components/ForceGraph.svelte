<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { forceSimulation, forceLink, forceManyBody, forceCenter, forceCollide } from 'd3-force';
	import * as fmt from '$lib/fmt.js';

	interface Edge {
		from_wallet: string;
		to_wallet: string;
		total_usdc?: string;
		total_usdc_human?: string;
		tx_count: number;
		from_is_agent?: boolean;
		to_is_agent?: boolean;
	}

	interface Props {
		edges: Edge[];
	}

	let { edges }: Props = $props();

	let canvas: HTMLCanvasElement;
	let width = 800;
	let height = 640;
	let dpr = 1;

	interface SimNode {
		id: string;
		isAgent: boolean;
		x: number;
		y: number;
		vx: number;
		vy: number;
		index?: number;
	}

	interface SimLink {
		source: SimNode | string;
		target: SimNode | string;
		volume: number;
		txCount: number;
	}

	let nodes: SimNode[] = [];
	let links: SimLink[] = [];
	let sim: ReturnType<typeof forceSimulation<SimNode>> | null = null;

	let hoveredNode: SimNode | null = null;
	let mouseX = 0;
	let mouseY = 0;

	// Transform state for pan/zoom (intentionally NOT $state — reading it inside
	// the build-graph $effect would otherwise subscribe the effect and cause the
	// simulation to rebuild on every zoom/pan).
	let transform = { x: 0, y: 0, k: 1 };
	let zoomPct = $state(100);
	let dragging = false;
	let dragStart = { x: 0, y: 0 };
	let transformStart = { x: 0, y: 0 };

	// FNV-1a 32-bit hash → deterministic positions from the wallet id.
	function hash32(s: string): number {
		let h = 0x811c9dc5;
		for (let i = 0; i < s.length; i++) {
			h ^= s.charCodeAt(i);
			h = Math.imul(h, 0x01000193);
		}
		return h >>> 0;
	}

	function seededPosition(id: string): { x: number; y: number } {
		const h = hash32(id);
		const u = (h & 0xffff) / 0xffff;
		const v = ((h >>> 16) & 0xffff) / 0xffff;
		const angle = u * Math.PI * 2;
		const radius = Math.sqrt(v) * 220;
		return {
			x: width / 2 + Math.cos(angle) * radius,
			y: height / 2 + Math.sin(angle) * radius
		};
	}

	function buildGraph(edges: Edge[]) {
		const nodeMap: Record<string, SimNode> = {};

		for (const edge of edges) {
			if (!nodeMap[edge.from_wallet]) {
				const p = seededPosition(edge.from_wallet);
				nodeMap[edge.from_wallet] = {
					id: edge.from_wallet,
					isAgent: edge.from_is_agent ?? false,
					x: p.x,
					y: p.y,
					vx: 0,
					vy: 0
				};
			}
			if (!nodeMap[edge.to_wallet]) {
				const p = seededPosition(edge.to_wallet);
				nodeMap[edge.to_wallet] = {
					id: edge.to_wallet,
					isAgent: edge.to_is_agent ?? false,
					x: p.x,
					y: p.y,
					vx: 0,
					vy: 0
				};
			}
		}

		nodes = Object.values(nodeMap);
		links = edges.map((e) => ({
			source: e.from_wallet,
			target: e.to_wallet,
			volume: parseFloat(e.total_usdc_human ?? '0'),
			txCount: e.tx_count
		}));

		// Volume range for color scaling
		const volumes = links.map((l) => l.volume).filter((v) => v > 0);
		const maxVol = volumes.length ? Math.max(...volumes) : 1;
		const minVol = volumes.length ? Math.min(...volumes) : 0;
		const volRange = maxVol - minVol || 1;

		// Build simulation
		sim = forceSimulation<SimNode>(nodes)
			.force(
				'link',
				forceLink<SimNode, SimLink>(links)
					.id((d) => d.id)
					.distance((d) => {
						const norm = (d.volume - minVol) / volRange;
						return 60 - norm * 30;
					})
					.strength(0.4)
			)
			.force('charge', forceManyBody<SimNode>().strength(-40))
			.force('center', forceCenter<SimNode>(width / 2, height / 2).strength(0.05))
			.force(
				'collide',
				forceCollide<SimNode>().radius((d) => (d.isAgent ? 14 : 8))
			)
			.alphaDecay(0.02)
			.stop();

		// Pre-warm so the layout is settled deterministically.
		for (let i = 0; i < 300; i++) sim.tick();
		sim.alpha(0).stop();

		draw();
	}

	function volColor(vol: number, minVol: number, volRange: number): string {
		const norm = Math.min(1, (vol - minVol) / volRange);
		const r = Math.round(60 + norm * 49);
		const g = Math.round(100 + norm * 113);
		const b = Math.round(140 + norm * 110);
		return `rgba(${r},${g},${b},${0.3 + norm * 0.45})`;
	}

	function draw() {
		if (!canvas) return;
		const ctx = canvas.getContext('2d');
		if (!ctx) return;

		ctx.save();
		ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
		ctx.clearRect(0, 0, width, height);

		ctx.save();
		ctx.translate(transform.x, transform.y);
		ctx.scale(transform.k, transform.k);

		const volumes = links.map((l) => l.volume).filter((v) => v > 0);
		const maxVol = volumes.length ? Math.max(...volumes) : 1;
		const minVol = volumes.length ? Math.min(...volumes) : 0;
		const volRange = maxVol - minVol || 1;

		// Draw links
		for (const link of links) {
			const s = link.source as SimNode;
			const t = link.target as SimNode;
			if (!s.x || !t.x) continue;

			ctx.beginPath();
			ctx.moveTo(s.x, s.y);
			ctx.lineTo(t.x, t.y);
			ctx.strokeStyle = volColor(link.volume, minVol, volRange);
			ctx.lineWidth = 0.8 + Math.min(2.5, (link.volume / maxVol) * 2.5);
			ctx.stroke();
		}

		// Draw nodes
		for (const node of nodes) {
			const r = node.isAgent ? 5 : 3;
			const isHovered = hoveredNode === node;

			if (isHovered) {
				ctx.beginPath();
				ctx.arc(node.x, node.y, r + 6, 0, Math.PI * 2);
				ctx.fillStyle = 'rgba(109,213,250,0.15)';
				ctx.fill();
			}

			ctx.beginPath();
			ctx.arc(node.x, node.y, r, 0, Math.PI * 2);
			ctx.fillStyle = node.isAgent
				? isHovered
					? '#9ae4ff'
					: '#6dd5fa'
				: isHovered
					? '#888'
					: '#555';
			ctx.fill();
		}

		ctx.restore();

		// Draw tooltip
		if (hoveredNode) {
			const label = fmt.addr(hoveredNode.id);
			const tag = hoveredNode.isAgent ? ' · agent' : '';
			const text = label + tag;

			ctx.font = '11px monospace';
			const metrics = ctx.measureText(text);
			const pw = 8;
			const tw = metrics.width + pw * 2;
			const th = 20;
			let tx = mouseX + 12;
			let ty = mouseY - th - 4;
			if (tx + tw > width) tx = mouseX - tw - 12;
			if (ty < 0) ty = mouseY + 16;

			ctx.fillStyle = 'rgba(20,20,30,0.9)';
			ctx.beginPath();
			ctx.roundRect(tx, ty, tw, th, 4);
			ctx.fill();
			ctx.strokeStyle = 'rgba(109,213,250,0.3)';
			ctx.lineWidth = 1;
			ctx.stroke();

			ctx.fillStyle = '#e0e0e0';
			ctx.fillText(text, tx + pw, ty + 14);
		}

		ctx.restore();
	}

	function findNodeAt(mx: number, my: number): SimNode | null {
		const gx = (mx - transform.x) / transform.k;
		const gy = (my - transform.y) / transform.k;

		for (let i = nodes.length - 1; i >= 0; i--) {
			const n = nodes[i];
			const r = (n.isAgent ? 5 : 3) + 4;
			const dx = gx - n.x;
			const dy = gy - n.y;
			if (dx * dx + dy * dy <= r * r) return n;
		}
		return null;
	}

	function handleMouseMove(e: MouseEvent) {
		const rect = canvas.getBoundingClientRect();
		mouseX = e.clientX - rect.left;
		mouseY = e.clientY - rect.top;

		if (dragging) {
			transform.x = transformStart.x + (e.clientX - dragStart.x);
			transform.y = transformStart.y + (e.clientY - dragStart.y);
			draw();
			return;
		}

		const node = findNodeAt(mouseX, mouseY);
		if (node !== hoveredNode) {
			hoveredNode = node;
			canvas.style.cursor = node ? 'pointer' : 'grab';
			draw();
		}
	}

	function handleMouseDown(e: MouseEvent) {
		if (e.button !== 0) return;
		dragging = true;
		dragStart = { x: e.clientX, y: e.clientY };
		transformStart = { x: transform.x, y: transform.y };
		canvas.style.cursor = 'grabbing';
	}

	function handleMouseUp() {
		dragging = false;
		canvas.style.cursor = hoveredNode ? 'pointer' : 'grab';
	}

	function zoomAt(mx: number, my: number, factor: number) {
		const newK = Math.max(0.2, Math.min(5, transform.k * factor));
		transform.x = mx - (mx - transform.x) * (newK / transform.k);
		transform.y = my - (my - transform.y) * (newK / transform.k);
		transform.k = newK;
		zoomPct = Math.round(newK * 100);
		draw();
	}

	function handleWheel(e: WheelEvent) {
		e.preventDefault();
		const rect = canvas.getBoundingClientRect();
		zoomAt(e.clientX - rect.left, e.clientY - rect.top, e.deltaY > 0 ? 0.9 : 1.1);
	}

	export function zoomIn() {
		zoomAt(width / 2, height / 2, 1.25);
	}
	export function zoomOut() {
		zoomAt(width / 2, height / 2, 0.8);
	}
	export function resetView() {
		transform = { x: 0, y: 0, k: 1 };
		zoomPct = 100;
		draw();
	}
	export function fitToView() {
		if (!nodes.length) return;
		let minX = Infinity,
			minY = Infinity,
			maxX = -Infinity,
			maxY = -Infinity;
		for (const n of nodes) {
			if (n.x < minX) minX = n.x;
			if (n.y < minY) minY = n.y;
			if (n.x > maxX) maxX = n.x;
			if (n.y > maxY) maxY = n.y;
		}
		const pad = 40;
		const gw = Math.max(1, maxX - minX);
		const gh = Math.max(1, maxY - minY);
		const k = Math.min((width - pad * 2) / gw, (height - pad * 2) / gh, 3);
		const cx = (minX + maxX) / 2;
		const cy = (minY + maxY) / 2;
		transform = {
			k,
			x: width / 2 - cx * k,
			y: height / 2 - cy * k
		};
		zoomPct = Math.round(k * 100);
		draw();
	}

	function handleClick() {
		const node = findNodeAt(mouseX, mouseY);
		if (node) {
			goto(resolve(`/wallet/${node.id}/`));
		}
	}

	function handleMouseLeave() {
		hoveredNode = null;
		dragging = false;
		canvas.style.cursor = 'grab';
		draw();
	}

	$effect(() => {
		if (!canvas || !edges.length) return;
		transform = { x: 0, y: 0, k: 1 };
		zoomPct = 100;
		buildGraph(edges);
		fitToView();
	});

	onMount(() => {
		const rect = canvas.parentElement?.getBoundingClientRect();
		if (rect) {
			width = rect.width;
			height = rect.height;
		}
		dpr = window.devicePixelRatio || 1;
		canvas.width = width * dpr;
		canvas.height = height * dpr;
		canvas.style.width = width + 'px';
		canvas.style.height = height + 'px';

		const ro = new ResizeObserver((entries) => {
			for (const entry of entries) {
				const { width: w, height: h } = entry.contentRect;
				width = w;
				height = h;
				dpr = window.devicePixelRatio || 1;
				canvas.width = width * dpr;
				canvas.height = height * dpr;
				canvas.style.width = width + 'px';
				canvas.style.height = height + 'px';
				draw();
			}
		});
		if (canvas.parentElement) ro.observe(canvas.parentElement);

		return () => {
			ro.disconnect();
			sim?.stop();
		};
	});
</script>

<div class="fg-wrap">
	<canvas
		bind:this={canvas}
		onmousemove={handleMouseMove}
		onmousedown={handleMouseDown}
		onmouseup={handleMouseUp}
		onclick={handleClick}
		onwheel={handleWheel}
		onmouseleave={handleMouseLeave}
		style="cursor:grab;width:100%;height:100%;display:block"
	></canvas>

	<div class="fg-controls" aria-label="Graph controls">
		<button class="fg-btn" onclick={zoomIn} title="Zoom in" aria-label="Zoom in">+</button>
		<button class="fg-btn" onclick={zoomOut} title="Zoom out" aria-label="Zoom out">−</button>
		<button class="fg-btn" onclick={fitToView} title="Fit to view" aria-label="Fit to view">
			<svg
				viewBox="0 0 14 14"
				width="12"
				height="12"
				fill="none"
				stroke="currentColor"
				stroke-width="1.4"
			>
				<path d="M2 5 V2 H5 M9 2 H12 V5 M12 9 V12 H9 M5 12 H2 V9" />
			</svg>
		</button>
		<button class="fg-btn" onclick={resetView} title="Reset view" aria-label="Reset view">
			<svg
				viewBox="0 0 14 14"
				width="12"
				height="12"
				fill="none"
				stroke="currentColor"
				stroke-width="1.4"
			>
				<path d="M11 4 A4 4 0 1 0 12 8" />
				<path d="M11 1.5 V4 H8.5" />
			</svg>
		</button>
		<div class="fg-zoom-label mono" aria-live="polite">{zoomPct}%</div>
	</div>
</div>

<style>
	.fg-wrap {
		position: relative;
		width: 100%;
		height: 100%;
	}
	.fg-controls {
		position: absolute;
		top: 10px;
		right: 10px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		background: rgba(20, 20, 28, 0.7);
		backdrop-filter: blur(8px);
		border: 1px solid var(--border-2, #2a2a35);
		border-radius: 6px;
		padding: 4px;
		z-index: 2;
	}
	.fg-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		background: transparent;
		border: 1px solid transparent;
		border-radius: 4px;
		color: var(--fg-2, #c0c0c8);
		font-size: 14px;
		font-family: inherit;
		line-height: 1;
		cursor: pointer;
		padding: 0;
		transition:
			background 120ms ease,
			color 120ms ease,
			border-color 120ms ease;
	}
	.fg-btn:hover {
		background: rgba(109, 213, 250, 0.12);
		color: #9ae4ff;
		border-color: rgba(109, 213, 250, 0.3);
	}
	.fg-zoom-label {
		text-align: center;
		font-size: 10px;
		color: var(--fg-3, #909098);
		padding-top: 2px;
		border-top: 1px solid var(--border-1, #1c1c24);
		margin-top: 2px;
	}
</style>
