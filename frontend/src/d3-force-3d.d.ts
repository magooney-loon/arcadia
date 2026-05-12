declare module 'd3-force-3d' {
	interface SimulationNodeDatum {
		x: number;
		y: number;
		z: number;
		vx?: number;
		vy?: number;
		vz?: number;
		index?: number;
	}

	interface SimulationLinkDatum {
		source: unknown;
		target: unknown;
		index?: number;
		[key: string]: unknown;
	}

	interface ForceLink<N = SimulationNodeDatum, L = SimulationLinkDatum> {
		id(fn: (node: N, i: number, nodesData: N[]) => string): ForceLink;
		distance(fnOrValue: number | ((link: L, i: number, linksData: L[]) => number)): ForceLink;
		strength(fnOrValue: number | ((link: L, i: number, linksData: L[]) => number)): ForceLink;
		links(links: L[]): ForceLink;
	}

	interface ForceManyBody<N = SimulationNodeDatum> {
		strength(fnOrValue: number | ((node: N, i: number, nodesData: N[]) => number)): ForceManyBody;
	}

	interface ForceRadial<N = SimulationNodeDatum> {
		strength(fnOrValue: number | ((node: N, i: number, nodesData: N[]) => number)): ForceRadial;
	}

	interface ForceCenter {
		x(x: number): ForceCenter;
		y(y: number): ForceCenter;
		z(z: number): ForceCenter;
	}

	interface Simulation<N = SimulationNodeDatum> {
		force(name: string, force: unknown): Simulation;
		stop(): Simulation;
		tick(count?: number): Simulation;
		nodes(): N[];
		alpha(value: number): Simulation;
		on(type: string, callback: (...args: unknown[]) => void): Simulation;
	}

	export function forceSimulation<N = SimulationNodeDatum>(nodes?: N[]): Simulation<N>;
	export function forceLink<N = SimulationNodeDatum, L = SimulationLinkDatum>(
		links?: L[]
	): ForceLink<N, L>;
	export function forceManyBody<N = SimulationNodeDatum>(): ForceManyBody<N>;
	export function forceCenter(x?: number, y?: number, z?: number): ForceCenter;
	export function forceRadial<N = SimulationNodeDatum>(
		radius: number,
		x?: number,
		y?: number,
		z?: number
	): ForceRadial<N>;
	export function forceCollide(radius?: number): unknown;
	export function forceX(x?: number): unknown;
	export function forceY(y?: number): unknown;
	export function forceZ(z?: number): unknown;
}
