<script lang="ts">
	import { T } from '@threlte/core';
	import * as THREE from 'three';
	import { getNodePositions } from '$lib/scene-state/layout.svelte';

	const MAX = 5000;
	const dummy = new THREE.Object3D();

	let meshRef = $state<THREE.InstancedMesh | undefined>();

	/** Non-agent wallet entries, re-derived when positions change */
	const walletEntries = $derived(
		Object.entries(getNodePositions()).filter(([, pos]) => !pos.isAgent)
	);

	/** Scale: base 0.07, grows with txCount up to 0.20 */
	function nodeScale(txCount: number): number {
		return 0.07 + Math.min(txCount / 100, 1) * 0.13;
	}

	$effect(() => {
		if (!meshRef) return;

		const entries = walletEntries;
		const count = Math.min(entries.length, MAX);
		meshRef.count = count;

		for (let i = 0; i < count; i++) {
			const pos = entries[i][1];
			dummy.position.set(pos.x, pos.y, pos.z);
			dummy.scale.setScalar(nodeScale(pos.txCount));
			dummy.updateMatrix();
			meshRef.setMatrixAt(i, dummy.matrix);
		}

		meshRef.instanceMatrix.needsUpdate = true;
	});
</script>

<T.InstancedMesh bind:ref={meshRef} args={[undefined, undefined, MAX]}>
	<T.IcosahedronGeometry args={[1, 0]} />
	<T.MeshStandardMaterial color="#ffffff" metalness={0.15} roughness={0.7} />
</T.InstancedMesh>
