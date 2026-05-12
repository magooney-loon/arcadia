import { forceSimulation, forceLink, forceManyBody, forceCenter, forceRadial } from 'd3-force-3d';
import type { Edge } from '$lib/api/graph/types.js';
import { SvelteSet, SvelteMap } from 'svelte/reactivity';

export interface NodePosition {
	x: number;
	y: number;
	z: number;
	isAgent: boolean;
	txCount: number;
	usdcVolume: number;
}

let _nodePositions = $state<Record<string, NodePosition>>({});
let _orderedEdges = $state<
	Array<{ from: string; to: string; total_usdc: number; tx_count: number }>
>([]);

export function getNodePositions() {
	return _nodePositions;
}
export function getOrderedEdges() {
	return _orderedEdges;
}

const SPHERE_RADIUS = 3.5;
const RADIAL_STRENGTH = 0.18;
const LINK_DISTANCE = 0.8;
const AGENT_CHARGE = -15;
const NODE_CHARGE = -5;
const TICK_COUNT = 300;

/** d3-force node with our custom properties */
interface SimNode {
	id: string;
	x: number;
	y: number;
	z: number;
	isAgent: boolean;
	txCount: number;
	usdcVolume: number;
}

/** d3-force link with our custom properties */
interface SimLink {
	source: string | SimNode;
	target: string | SimNode;
	tx_count: number;
	total_usdc: number;
}

export function runSimulation(edges: Edge[], agentAddresses: Set<string>) {
	if (edges.length === 0) return;

	// Normalise agent addresses to lowercase for matching
	const agentSet = new SvelteSet([...agentAddresses].map((a) => a.toLowerCase()));

	// Build unique node map from edges
	const nodeMap = new SvelteMap<string, SimNode>();

	for (const edge of edges) {
		// Skip self-loops
		if (edge.from_wallet === edge.to_wallet) continue;

		for (const addr of [edge.from_wallet, edge.to_wallet]) {
			if (!nodeMap.has(addr)) {
				nodeMap.set(addr, {
					id: addr,
					x: (Math.random() - 0.5) * SPHERE_RADIUS,
					y: (Math.random() - 0.5) * SPHERE_RADIUS,
					z: (Math.random() - 0.5) * SPHERE_RADIUS,
					isAgent: agentSet.has(addr.toLowerCase()),
					txCount: 0,
					usdcVolume: 0
				});
			}
		}

		const from = nodeMap.get(edge.from_wallet)!;
		const to = nodeMap.get(edge.to_wallet)!;

		from.txCount += edge.tx_count;
		to.txCount += edge.tx_count;

		const usdc = parseFloat(edge.total_usdc ?? '0') || 0;
		from.usdcVolume += usdc;
		to.usdcVolume += usdc;
	}

	const nodes = Array.from(nodeMap.values());

	// Build link data — strength scales with tx_count
	const links: SimLink[] = edges
		.filter((e) => e.from_wallet !== e.to_wallet)
		.map((e) => ({
			source: e.from_wallet,
			target: e.to_wallet,
			tx_count: e.tx_count,
			total_usdc: parseFloat(e.total_usdc ?? '0') || 0
		}));

	// Run simulation synchronously (300 ticks, then stop — no animation)
	const simulation = forceSimulation<SimNode>(nodes)
		.force(
			'link',
			forceLink<SimNode, SimLink>(links)
				.id((d) => d.id)
				.distance(LINK_DISTANCE)
				.strength((d) => {
					const tc = (d as unknown as { tx_count: number }).tx_count ?? 0;
					return Math.min(tc / 30, 1) * 0.4 + 0.1;
				})
		)
		.force(
			'charge',
			forceManyBody<SimNode>().strength((d) => (d.isAgent ? AGENT_CHARGE : NODE_CHARGE))
		)
		.force('center', forceCenter(0, 0, 0))
		.force('radial', forceRadial<SimNode>(SPHERE_RADIUS * 0.7, 0, 0, 0).strength(RADIAL_STRENGTH))
		.stop();

	simulation.tick(TICK_COUNT);

	// Extract final positions into reactive state
	const newPos: Record<string, NodePosition> = {};
	for (const node of nodes) {
		newPos[node.id] = {
			x: node.x,
			y: node.y,
			z: node.z,
			isAgent: node.isAgent,
			txCount: node.txCount,
			usdcVolume: node.usdcVolume
		};
	}

	_nodePositions = newPos;
	_orderedEdges = links.map((l) => ({
		from: typeof l.source === 'object' ? (l.source as SimNode).id : (l.source as string),
		to: typeof l.target === 'object' ? (l.target as SimNode).id : (l.target as string),
		total_usdc: l.total_usdc,
		tx_count: l.tx_count
	}));
}
