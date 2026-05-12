import { getNodePositions } from './layout.svelte.js';
import type { Transfer } from '$lib/api/transfers/types.js';

export interface LiveParticle {
	/** Source wallet address */
	from: string;
	/** Destination wallet address */
	to: string;
	/** Token type — determines color */
	token: 'USDC' | 'EURC' | 'USYC' | 'OTHER';
	/** Progress 0→1 along the edge */
	t: number;
	/** Speed — how fast t increments per second */
	speed: number;
}

const BUFFER_SIZE = 500;

let _particles = $state<LiveParticle[]>([]);

export function getParticles() {
	return _particles;
}

/** Spawn a batch of particles from recent transfers */
export function spawnParticles(transfers: Transfer[]) {
	const positions = getNodePositions();
	const newParticles: LiveParticle[] = [];

	for (const tx of transfers) {
		const from = tx.from_addr;
		const to = tx.to_addr;
		if (!from || !to) continue;
		// Only spawn if both endpoints exist in the graph
		if (!positions[from] || !positions[to]) continue;

		newParticles.push({
			from,
			to,
			token: tx.token_symbol ?? 'OTHER',
			t: 0,
			speed: 0.25 + Math.random() * 0.35 // random speed variation
		});
	}

	if (newParticles.length === 0) return;

	// Append to ring buffer, trim to max size
	_particles = [..._particles, ...newParticles].slice(-BUFFER_SIZE);
}

/** Advance all particles by delta seconds, remove dead ones */
export function tickParticles(delta: number) {
	const updated: LiveParticle[] = [];
	for (const p of _particles) {
		p.t += p.speed * delta;
		if (p.t < 1) {
			updated.push(p);
		}
		// t >= 1 → particle is dead, dropped
	}
	_particles = updated;
}
