<script lang="ts">
	import { onMount } from 'svelte';
	import { forceSimulation, forceLink, forceManyBody, forceCenter, forceCollide } from 'd3-force';
	import * as fmt from '$lib/fmt.js';

	interface Edge {
		from_wallet: string;
		to_wallet: string;
		total_usdc?: string;
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

	// Transform state for pan/zoom
	let transform = { x: 0, y: 0, k: 1 };
	let dragging = false;
	let dragStart = { x: 0, y: 0 };
	let transformStart = { x: 0, y: 0 };

	function buildGraph(edges: Edge[]) {
		const nodeMap: Record<string, SimNode> = {};

		for (const edge of edges) {
			if (!nodeMap[edge.from_wallet]) {
				nodeMap[edge.from_wallet] = {
					id: edge.from_wallet,
					isAgent: edge.from_is_agent ?? false,
					x: width / 2 + (Math.random() - 0.5) * 200,
					y: height / 2 + (Math.random() - 0.5) * 200,
					vx: 0,
					vy: 0
				};
			}
			if (!nodeMap[edge.to_wallet]) {
				nodeMap[edge.to_wallet] = {
					id: edge.to_wallet,
					isAgent: edge.to_is_agent ?? false,
					x: width / 2 + (Math.random() - 0.5) * 200,
					y: height / 2 + (Math.random() - 0.5) * 200,
					vx: 0,
					vy: 0
				};
			}
		}

		nodes = Object.values(nodeMap);
		links = edges.map((e) => ({
			source: e.from_wallet,
			target: e.to_wallet,
			volume: parseFloat(e.total_usdc ?? '0'),
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
						return 60 - norm * 30; // high volume = closer
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

		// Pre-warm
		for (let i = 0; i < 300; i++) sim.tick();
		sim.alpha(0).stop();

		draw();
	}

	function volColor(vol: number, minVol: number, volRange: number): string {
		const norm = Math.min(1, (vol - minVol) / volRange);
		const r = Math.round(60 + norm * 49); // 60 → 109
		const g = Math.round(100 + norm * 113); // 100 → 213
		const b = Math.round(140 + norm * 110); // 140 → 250
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

		// Volume range
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

			// Glow for hovered
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
		// Convert screen coords to graph coords
		const gx = (mx - transform.x) / transform.k;
		const gy = (my - transform.y) / transform.k;

		for (let i = nodes.length - 1; i >= 0; i--) {
			const n = nodes[i];
			const r = (n.isAgent ? 5 : 3) + 4; // hit radius with padding
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

	function handleWheel(e: WheelEvent) {
		e.preventDefault();
		const rect = canvas.getBoundingClientRect();
		const mx = e.clientX - rect.left;
		const my = e.clientY - rect.top;

		const delta = e.deltaY > 0 ? 0.9 : 1.1;
		const newK = Math.max(0.2, Math.min(5, transform.k * delta));

		// Zoom towards mouse position
		transform.x = mx - (mx - transform.x) * (newK / transform.k);
		transform.y = my - (my - transform.y) * (newK / transform.k);
		transform.k = newK;
		draw();
	}

	function handleClick() {
		const node = findNodeAt(mouseX, mouseY);
		if (node) {
			window.open(fmt.explorerAddr(node.id), '_blank', 'noopener');
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
		buildGraph(edges);
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

		// Resize observer
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
