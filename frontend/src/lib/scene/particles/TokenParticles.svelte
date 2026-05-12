<script lang="ts">
	import { T, useTask } from '@threlte/core';
	import * as THREE from 'three';
	import { getParticles, tickParticles } from '$lib/scene-state/particles.svelte';
	import { getNodePositions } from '$lib/scene-state/layout.svelte';

	const MAX = 500;
	const dummy = new THREE.Object3D();

	const TOKEN_COLORS: Record<string, THREE.Color> = {
		USDC: new THREE.Color('#2775ca'),
		EURC: new THREE.Color('#e8b84b'),
		USYC: new THREE.Color('#7b61ff'),
		OTHER: new THREE.Color('#9aa0b4')
	};

	let meshRef = $state<THREE.InstancedMesh | undefined>();

	useTask((delta: number) => {
		// Advance particles
		tickParticles(delta);

		if (!meshRef) return;

		const particles = getParticles();
		const positions = getNodePositions();
		const count = Math.min(particles.length, MAX);
		meshRef.count = count;

		for (let i = 0; i < count; i++) {
			const p = particles[i];
			const from = positions[p.from];
			const to = positions[p.to];
			if (!from || !to) continue;

			// Lerp between source and destination
			const x = from.x + (to.x - from.x) * p.t;
			const y = from.y + (to.y - from.y) * p.t;
			const z = from.z + (to.z - from.z) * p.t;

			dummy.position.set(x, y, z);
			// Particle swells in the middle of its journey, small at start/end
			const swell = Math.sin(p.t * Math.PI);
			dummy.scale.setScalar(0.04 + swell * 0.06);
			dummy.updateMatrix();
			meshRef.setMatrixAt(i, dummy.matrix);
			meshRef.setColorAt(i, TOKEN_COLORS[p.token] ?? TOKEN_COLORS.OTHER);
		}

		meshRef.instanceMatrix.needsUpdate = true;
		if (meshRef.instanceColor) meshRef.instanceColor.needsUpdate = true;
	});
</script>

<!-- Token particles — instanced dots travelling along transfer edges -->
<T.InstancedMesh bind:ref={meshRef} args={[undefined, undefined, MAX]}>
	<T.SphereGeometry args={[1, 6, 4]} />
	<T.MeshBasicMaterial />
</T.InstancedMesh>
