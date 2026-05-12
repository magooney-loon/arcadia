<script lang="ts">
	import { T, useTask } from '@threlte/core';
	import * as THREE from 'three';
	import { getNodePositions } from '$lib/scene-state/layout.svelte';

	const MAX = 500;
	const dummy = new THREE.Object3D();
	const agentColor = new THREE.Color('#7ee5a8');

	let meshRef = $state<THREE.InstancedMesh | undefined>();
	let t = 0;

	/** Agent wallet entries, re-derived when positions change */
	const agentEntries = $derived(
		Object.entries(getNodePositions()).filter(([, pos]) => pos.isAgent)
	);

	/** Scale: base 0.10, grows with txCount up to 0.28 */
	function agentScale(txCount: number): number {
		return 0.1 + Math.min(txCount / 80, 1) * 0.18;
	}

	$effect(() => {
		if (!meshRef) return;

		const entries = agentEntries;
		const count = Math.min(entries.length, MAX);
		meshRef.count = count;

		for (let i = 0; i < count; i++) {
			const pos = entries[i][1];
			dummy.position.set(pos.x, pos.y, pos.z);
			dummy.scale.setScalar(agentScale(pos.txCount));
			dummy.updateMatrix();
			meshRef.setMatrixAt(i, dummy.matrix);
			meshRef.setColorAt(i, agentColor);
		}

		meshRef.instanceMatrix.needsUpdate = true;
		if (meshRef.instanceColor) meshRef.instanceColor.needsUpdate = true;
	});

	/** Pulse all agent nodes — green "heartbeat" */
	useTask((delta: number) => {
		if (!meshRef) return;
		t += delta * 1.8;

		const entries = agentEntries;
		const count = Math.min(entries.length, MAX);
		const pulse = 1 + Math.sin(t) * 0.12;

		for (let i = 0; i < count; i++) {
			const pos = entries[i][1];
			dummy.position.set(pos.x, pos.y, pos.z);
			dummy.scale.setScalar(agentScale(pos.txCount) * pulse);
			dummy.updateMatrix();
			meshRef.setMatrixAt(i, dummy.matrix);
		}

		meshRef.instanceMatrix.needsUpdate = true;
	});
</script>

<T.InstancedMesh bind:ref={meshRef} args={[undefined, undefined, MAX]}>
	<T.IcosahedronGeometry args={[1, 1]} />
	<T.MeshStandardMaterial
		color="#7ee5a8"
		emissive="#7ee5a8"
		emissiveIntensity={0.5}
		metalness={0.3}
		roughness={0.4}
	/>
</T.InstancedMesh>
